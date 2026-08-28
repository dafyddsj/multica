package agentmail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultAPIBase = "https://api.agentmail.to/v0"
	defaultTimeout = 30 * time.Second
	userAgent      = "multica-agentmail/1"
	maxErrorBody   = 240
)

// Read, send, and draft only. Management scopes stay off this whitelist.
var inboxKeyPermissions = map[string]bool{
	"inbox_read":   true,
	"message_read": true,
	"message_send": true,
	"draft_read":   true,
	"draft_create": true,
	"draft_send":   true,
}

var ErrRemoteNotFound = errors.New("agentmail: remote not found")

var errRemoteNotFound = ErrRemoteNotFound

type clientCred struct {
	apiKey string
	podID  string
}

type remoteInbox struct {
	id      string
	address string
}

type inspectedKey struct {
	authorityKind string
	podID         string
	domain        string
}

type apiClient interface {
	ensurePod(ctx context.Context, orgKey, clientID string) (string, error)
	ensureInbox(ctx context.Context, cred clientCred, clientID, displayName string, addr InboxAddress) (remoteInbox, error)
	createInboxKey(ctx context.Context, cred clientCred, inboxID string) (string, error)
	deleteInbox(ctx context.Context, cred clientCred, inboxID string) error
	deletePod(ctx context.Context, orgKey, podID string) error
	validateOrgKey(ctx context.Context, orgKey string) error
	inspectKey(ctx context.Context, orgKey string) (inspectedKey, error)
	listDomains(ctx context.Context, cred clientCred) ([]RemoteDomain, error)
	listThreads(ctx context.Context, inboxKey, inboxID string, query threadQuery) (ThreadPage, error)
	getThread(ctx context.Context, inboxKey, inboxID, threadID string) (ThreadDetail, error)
	listDrafts(ctx context.Context, inboxKey, inboxID string, query draftQuery) (DraftPage, error)
}

type liveClient struct {
	http *http.Client
	base string
}

func newLiveClient(httpClient *http.Client, baseURL string) liveClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = defaultAPIBase
	}
	return liveClient{http: httpClient, base: base}
}

func (c liveClient) ensurePod(ctx context.Context, orgKey, clientID string) (string, error) {
	var out podResponse
	err := c.do(ctx, http.MethodPost, "/pods", orgKey, map[string]string{
		"client_id": clientID,
		"name":      clientID,
	}, &out)
	if isConflict(err) {
		if decodeErr := c.decodeConflict(err, &out); decodeErr == nil && out.PodID != "" {
			return out.PodID, nil
		}
	}
	if err != nil {
		return "", err
	}
	if out.PodID == "" {
		return "", errors.New("agentmail: create pod missing pod_id")
	}
	return out.PodID, nil
}

func (c liveClient) ensureInbox(ctx context.Context, cred clientCred, clientID, displayName string, addr InboxAddress) (remoteInbox, error) {
	body := map[string]string{"client_id": clientID}
	if displayName != "" {
		body["display_name"] = displayName
	}
	if addr.Username != "" {
		body["username"] = addr.Username
	}
	if addr.Domain != "" {
		body["domain"] = addr.Domain
	}
	path := "/inboxes"
	if cred.podID != "" {
		path = "/pods/" + url.PathEscape(cred.podID) + "/inboxes"
	}
	var out inboxResponse
	err := c.do(ctx, http.MethodPost, path, cred.apiKey, body, &out)
	if isConflict(err) {
		if decodeErr := c.decodeConflict(err, &out); decodeErr == nil && out.ID() != "" && out.Email != "" {
			return remoteInbox{id: out.ID(), address: out.Email}, nil
		}
		if addr.Username != "" {
			return remoteInbox{}, fmt.Errorf("%w: %s", ErrAddressTaken, remoteErrorMessage(err))
		}
	}
	if err != nil {
		if addr.Username != "" && isUnprocessable(err) {
			return remoteInbox{}, fmt.Errorf("%w: %s", ErrBadAddress, remoteErrorMessage(err))
		}
		return remoteInbox{}, err
	}
	if out.ID() == "" || out.Email == "" {
		return remoteInbox{}, errors.New("agentmail: create inbox missing id or email")
	}
	return remoteInbox{id: out.ID(), address: out.Email}, nil
}

func (c liveClient) createInboxKey(ctx context.Context, cred clientCred, inboxID string) (string, error) {
	if inboxID == "" {
		return "", errors.New("agentmail: create inbox key missing inbox id")
	}
	var out apiKeyResponse
	err := c.do(ctx, http.MethodPost, "/inboxes/"+url.PathEscape(inboxID)+"/api-keys", cred.apiKey, map[string]any{
		"name":        "multica",
		"permissions": inboxKeyPermissions,
	}, &out)
	if err != nil {
		return "", err
	}
	if out.APIKey == "" {
		return "", errors.New("agentmail: create inbox key missing api_key")
	}
	return out.APIKey, nil
}

func (c liveClient) deleteInbox(ctx context.Context, cred clientCred, inboxID string) error {
	if inboxID == "" {
		return errRemoteNotFound
	}
	// AgentMail rejects inbox delete while scoped keys still exist. The
	// grant path always mints a "multica" inbox key.
	if err := c.clearInboxKeys(ctx, cred, inboxID); err != nil && !errors.Is(err, errRemoteNotFound) {
		return err
	}
	var last error
	for _, path := range inboxDeletePaths(cred, inboxID) {
		err := c.do(ctx, http.MethodDelete, path, cred.apiKey, nil, nil)
		if err == nil || errors.Is(err, errRemoteNotFound) {
			return err
		}
		last = err
	}
	return last
}

func inboxDeletePaths(cred clientCred, inboxID string) []string {
	escaped := url.PathEscape(inboxID)
	paths := make([]string, 0, 4)
	if cred.podID != "" {
		paths = append(paths, "/pods/"+url.PathEscape(cred.podID)+"/inboxes/"+escaped)
	}
	paths = append(paths, "/inboxes/"+escaped)
	if escaped != inboxID && !strings.Contains(inboxID, "/") {
		if cred.podID != "" {
			paths = append(paths, "/pods/"+url.PathEscape(cred.podID)+"/inboxes/"+inboxID)
		}
		paths = append(paths, "/inboxes/"+inboxID)
	}
	return paths
}

func (c liveClient) clearInboxKeys(ctx context.Context, cred clientCred, inboxID string) error {
	var out apiKeyListResponse
	err := c.do(ctx, http.MethodGet, "/inboxes/"+url.PathEscape(inboxID)+"/api-keys?limit=100", cred.apiKey, nil, &out)
	if errors.Is(err, errRemoteNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var last error
	for _, key := range out.APIKeys {
		if key.APIKeyID == "" {
			continue
		}
		delErr := c.do(ctx, http.MethodDelete, "/inboxes/"+url.PathEscape(inboxID)+"/api-keys/"+url.PathEscape(key.APIKeyID), cred.apiKey, nil, nil)
		if delErr != nil && !errors.Is(delErr, errRemoteNotFound) {
			last = delErr
		}
	}
	return last
}

func (c liveClient) deletePod(ctx context.Context, orgKey, podID string) error {
	if podID == "" {
		return errRemoteNotFound
	}
	err := c.do(ctx, http.MethodDelete, "/pods/"+url.PathEscape(podID), orgKey, nil, nil)
	if errors.Is(err, errRemoteNotFound) {
		return errRemoteNotFound
	}
	return err
}

func (c liveClient) validateOrgKey(ctx context.Context, orgKey string) error {
	_, err := c.inspectKey(ctx, orgKey)
	return err
}

func (c liveClient) inspectKey(ctx context.Context, orgKey string) (inspectedKey, error) {
	if strings.TrimSpace(orgKey) == "" {
		return inspectedKey{}, ErrBadOrgKey
	}
	var me authMeResponse
	if err := c.do(ctx, http.MethodGet, "/auth/me", orgKey, nil, &me); err != nil {
		if isAuthRejected(err) {
			return inspectedKey{}, ErrBadOrgKey
		}
		return inspectedKey{}, err
	}
	switch strings.ToLower(me.ScopeType) {
	case "organization":
		return inspectedKey{authorityKind: authorityBYOOrg, domain: c.lookupDomain(ctx, orgKey)}, nil
	case "pod":
		podID := firstNonEmpty(me.PodID, me.ScopeID)
		if podID == "" {
			return inspectedKey{}, ErrBadOrgKey
		}
		return inspectedKey{authorityKind: authorityBYOPod, podID: podID, domain: c.lookupDomain(ctx, orgKey)}, nil
	default:
		return inspectedKey{}, ErrBadOrgKey
	}
}

func (c liveClient) listDomains(ctx context.Context, cred clientCred) ([]RemoteDomain, error) {
	var out domainListResponse
	err := c.do(ctx, http.MethodGet, "/domains?limit=100", cred.apiKey, nil, &out)
	if err != nil && cred.podID != "" {
		err = c.do(ctx, http.MethodGet, "/pods/"+url.PathEscape(cred.podID)+"/domains?limit=100", cred.apiKey, nil, &out)
	}
	if err != nil {
		return nil, err
	}
	domains := make([]RemoteDomain, 0, len(out.Domains))
	for _, row := range out.Domains {
		name := strings.ToLower(strings.TrimSpace(row.Domain))
		if name == "" {
			continue
		}
		domains = append(domains, RemoteDomain{Name: name, SubdomainsEnabled: row.SubdomainsEnabled})
	}
	return domains, nil
}

func (c liveClient) listThreads(ctx context.Context, inboxKey, inboxID string, query threadQuery) (ThreadPage, error) {
	if inboxID == "" {
		return ThreadPage{}, errRemoteNotFound
	}
	values := url.Values{}
	values.Set("limit", "50")
	if query.PageToken != "" {
		values.Set("page_token", query.PageToken)
	}
	for _, label := range query.Labels {
		if label != "" {
			values.Add("labels", label)
		}
	}
	if query.IncludeTrash {
		values.Set("include_trash", "true")
	}
	path := "/inboxes/" + url.PathEscape(inboxID) + "/threads?" + values.Encode()
	var out threadListResponse
	if err := c.do(ctx, http.MethodGet, path, inboxKey, nil, &out); err != nil {
		return ThreadPage{}, err
	}
	page := ThreadPage{NextPageToken: out.NextPageToken, Threads: make([]Thread, 0, len(out.Threads))}
	for _, row := range out.Threads {
		page.Threads = append(page.Threads, row.thread())
	}
	return page, nil
}

func (c liveClient) listDrafts(ctx context.Context, inboxKey, inboxID string, query draftQuery) (DraftPage, error) {
	if inboxID == "" {
		return DraftPage{}, errRemoteNotFound
	}
	values := url.Values{}
	values.Set("limit", "50")
	if query.PageToken != "" {
		values.Set("page_token", query.PageToken)
	}
	for _, label := range query.Labels {
		if label != "" {
			values.Add("labels", label)
		}
	}
	path := "/inboxes/" + url.PathEscape(inboxID) + "/drafts?" + values.Encode()
	var out draftListResponse
	if err := c.do(ctx, http.MethodGet, path, inboxKey, nil, &out); err != nil {
		return DraftPage{}, err
	}
	page := DraftPage{NextPageToken: out.NextPageToken, Drafts: make([]Draft, 0, len(out.Drafts))}
	for _, row := range out.Drafts {
		page.Drafts = append(page.Drafts, row.draft())
	}
	return page, nil
}

func (c liveClient) getThread(ctx context.Context, inboxKey, inboxID, threadID string) (ThreadDetail, error) {
	if inboxID == "" || threadID == "" {
		return ThreadDetail{}, errRemoteNotFound
	}
	var out threadDetailResponse
	path := "/inboxes/" + url.PathEscape(inboxID) + "/threads/" + url.PathEscape(threadID)
	if err := c.do(ctx, http.MethodGet, path, inboxKey, nil, &out); err != nil {
		return ThreadDetail{}, err
	}
	detail := ThreadDetail{Thread: out.thread()}
	for _, msg := range out.Messages {
		detail.Messages = append(detail.Messages, Message{
			ID:        msg.MessageID,
			From:      msg.From,
			To:        msg.To,
			Timestamp: msg.Timestamp,
			Subject:   msg.Subject,
			Text:      firstNonEmpty(msg.Text, msg.ExtractedText, msg.Preview),
		})
	}
	return detail, nil
}

func (c liveClient) lookupDomain(ctx context.Context, apiKey string) string {
	var out domainListResponse
	if err := c.do(ctx, http.MethodGet, "/domains?limit=1", apiKey, nil, &out); err != nil {
		return ""
	}
	if len(out.Domains) == 0 {
		return ""
	}
	return out.Domains[0].Domain
}

func (c liveClient) do(ctx context.Context, method, path, apiKey string, body any, out any) error {
	if c.http == nil {
		return errors.New("agentmail: live client missing http")
	}
	if strings.TrimSpace(apiKey) == "" {
		return ErrBadOrgKey
	}
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return errRemoteNotFound
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &remoteAPIError{Status: resp.StatusCode, Message: remoteMessage(payload)}
	}
	if resp.StatusCode == http.StatusConflict {
		return &remoteAPIError{Status: resp.StatusCode, Message: remoteMessage(payload), Body: payload}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &remoteAPIError{Status: resp.StatusCode, Message: remoteMessage(payload)}
	}
	if out == nil || len(payload) == 0 {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("agentmail: decode %s %s: %w", method, path, err)
	}
	return nil
}

func (liveClient) decodeConflict(err error, out any) error {
	var remote *remoteAPIError
	if !errors.As(err, &remote) || len(remote.Body) == 0 {
		return err
	}
	if unmarshalErr := json.Unmarshal(remote.Body, out); unmarshalErr != nil {
		return err
	}
	return nil
}

type remoteAPIError struct {
	Status  int
	Message string
	Body    []byte
}

func (e *remoteAPIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("agentmail: remote status %d", e.Status)
	}
	return fmt.Sprintf("agentmail: remote status %d: %s", e.Status, e.Message)
}

func isConflict(err error) bool {
	var remote *remoteAPIError
	return errors.As(err, &remote) && remote.Status == http.StatusConflict
}

func isUnprocessable(err error) bool {
	var remote *remoteAPIError
	return errors.As(err, &remote) && (remote.Status == http.StatusUnprocessableEntity || remote.Status == http.StatusBadRequest)
}

func remoteErrorMessage(err error) string {
	if msg, ok := RemoteMessage(err); ok {
		return msg
	}
	return err.Error()
}

// RemoteMessage returns a vendor error string when err is a live AgentMail reply.
func RemoteMessage(err error) (string, bool) {
	var remote *remoteAPIError
	if errors.As(err, &remote) && strings.TrimSpace(remote.Message) != "" {
		return remote.Message, true
	}
	return "", false
}

func isAuthRejected(err error) bool {
	if errors.Is(err, ErrBadOrgKey) {
		return true
	}
	var remote *remoteAPIError
	return errors.As(err, &remote) && (remote.Status == http.StatusUnauthorized || remote.Status == http.StatusForbidden)
}

func remoteMessage(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}
	var payload struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
		Name    string          `json:"name"`
	}
	if json.Unmarshal(body, &payload) == nil {
		if payload.Message != "" {
			return clip(payload.Message, maxErrorBody)
		}
		if payload.Name != "" {
			return clip(payload.Name, maxErrorBody)
		}
		if msg := jsonString(payload.Error); msg != "" {
			return clip(msg, maxErrorBody)
		}
	}
	return clip(trimmed, maxErrorBody)
}

func jsonString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var obj struct {
		Message string `json:"message"`
		Name    string `json:"name"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		return firstNonEmpty(obj.Message, obj.Name)
	}
	return ""
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type podResponse struct {
	PodID string `json:"pod_id"`
}

type inboxResponse struct {
	InboxID string `json:"inbox_id"`
	Email   string `json:"email"`
}

func (r inboxResponse) ID() string {
	return r.InboxID
}

type apiKeyResponse struct {
	APIKey string `json:"api_key"`
}

type apiKeyListResponse struct {
	APIKeys []struct {
		APIKeyID string `json:"api_key_id"`
	} `json:"api_keys"`
}

type authMeResponse struct {
	ScopeType string `json:"scope_type"`
	ScopeID   string `json:"scope_id"`
	PodID     string `json:"pod_id"`
}

type domainListResponse struct {
	Domains []struct {
		Domain            string `json:"domain"`
		SubdomainsEnabled bool   `json:"subdomains_enabled"`
	} `json:"domains"`
}

type threadListResponse struct {
	Threads       []threadSummaryResponse `json:"threads"`
	NextPageToken string                  `json:"next_page_token"`
}

type threadSummaryResponse struct {
	ThreadID     string   `json:"thread_id"`
	Subject      string   `json:"subject"`
	Preview      string   `json:"preview"`
	Senders      []string `json:"senders"`
	Recipients   []string `json:"recipients"`
	Timestamp    string   `json:"timestamp"`
	MessageCount int      `json:"message_count"`
	Labels       []string `json:"labels"`
}

func (r threadSummaryResponse) thread() Thread {
	return Thread{
		ID:           r.ThreadID,
		Subject:      r.Subject,
		Preview:      r.Preview,
		Senders:      r.Senders,
		Recipients:   r.Recipients,
		Timestamp:    r.Timestamp,
		MessageCount: r.MessageCount,
		Labels:       r.Labels,
	}
}

type draftListResponse struct {
	Drafts        []draftSummaryResponse `json:"drafts"`
	NextPageToken string                 `json:"next_page_token"`
}

type draftSummaryResponse struct {
	DraftID   string   `json:"draft_id"`
	Subject   string   `json:"subject"`
	Preview   string   `json:"preview"`
	To        []string `json:"to"`
	UpdatedAt string   `json:"updated_at"`
	SendAt    string   `json:"send_at"`
	Labels    []string `json:"labels"`
}

func (r draftSummaryResponse) draft() Draft {
	return Draft{
		ID:        r.DraftID,
		Subject:   r.Subject,
		Preview:   r.Preview,
		To:        r.To,
		Timestamp: firstNonEmpty(r.SendAt, r.UpdatedAt),
		SendAt:    r.SendAt,
		Labels:    r.Labels,
	}
}

type threadDetailResponse struct {
	threadSummaryResponse
	Messages []struct {
		MessageID     string   `json:"message_id"`
		From          string   `json:"from"`
		To            []string `json:"to"`
		Timestamp     string   `json:"timestamp"`
		Subject       string   `json:"subject"`
		Text          string   `json:"text"`
		ExtractedText string   `json:"extracted_text"`
		Preview       string   `json:"preview"`
	} `json:"messages"`
}
