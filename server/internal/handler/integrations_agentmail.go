package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/integrations/agentmail"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type AgentMailWorkspaceResponse struct {
	Available       bool                     `json:"available"`
	HostedAvailable bool                     `json:"hosted_available"`
	Connected       bool                     `json:"connected"`
	Source          string                   `json:"source,omitempty"`
	State           string                   `json:"state,omitempty"`
	Domain          string                   `json:"domain,omitempty"`
	CanManage       bool                     `json:"can_manage"`
	Inboxes         []AgentMailInboxResponse `json:"inboxes"`
}

type AgentMailInboxResponse struct {
	AgentID     string `json:"agent_id"`
	Enabled     bool   `json:"enabled"`
	State       string `json:"state,omitempty"`
	Address     string `json:"address,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

type AgentMailThreadResponse struct {
	ThreadID     string   `json:"thread_id"`
	Subject      string   `json:"subject,omitempty"`
	Preview      string   `json:"preview,omitempty"`
	Senders      []string `json:"senders"`
	Recipients   []string `json:"recipients"`
	Timestamp    string   `json:"timestamp,omitempty"`
	MessageCount int      `json:"message_count"`
}

type AgentMailThreadListResponse struct {
	Threads       []AgentMailThreadResponse `json:"threads"`
	NextPageToken string                    `json:"next_page_token,omitempty"`
}

type AgentMailMessageResponse struct {
	MessageID string   `json:"message_id"`
	From      string   `json:"from"`
	To        []string `json:"to"`
	Timestamp string   `json:"timestamp,omitempty"`
	Subject   string   `json:"subject,omitempty"`
	Text      string   `json:"text"`
}

type AgentMailThreadDetailResponse struct {
	AgentMailThreadResponse
	Messages []AgentMailMessageResponse `json:"messages"`
}

type connectAgentMailRequest struct {
	Mode   string `json:"mode"`
	OrgKey string `json:"org_key"`
}

type grantAgentMailRequest struct {
	Username string `json:"username"`
	Domain   string `json:"domain"`
}

type AgentMailDomainListResponse struct {
	Domains []string `json:"domains"`
}

type AgentMailFolderListResponse struct {
	Folders []string `json:"folders"`
}

type AgentMailMailboxItemResponse struct {
	Kind         string   `json:"kind"`
	ID           string   `json:"id"`
	Subject      string   `json:"subject,omitempty"`
	Preview      string   `json:"preview,omitempty"`
	Participants []string `json:"participants"`
	Timestamp    string   `json:"timestamp,omitempty"`
}

type AgentMailMailboxResponse struct {
	Items         []AgentMailMailboxItemResponse `json:"items"`
	NextPageToken string                         `json:"next_page_token,omitempty"`
}

func (h *Handler) GetAgentMail(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	member, ok := h.workspaceMember(w, r, workspaceID)
	if !ok {
		return
	}

	resp := AgentMailWorkspaceResponse{
		Inboxes: []AgentMailInboxResponse{},
	}
	if h.AgentMail == nil {
		writeJSON(w, http.StatusOK, resp)
		return
	}
	resp.Available = h.AgentMail.Available()
	resp.HostedAvailable = h.AgentMail.HostedAvailable()
	resp.CanManage = roleAllowed(member.Role, "owner", "admin")

	status, err := h.AgentMail.GetWorkspace(r.Context(), member.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load agentmail")
		return
	}
	resp.Source = status.Source
	resp.State = status.State
	resp.Domain = status.Domain
	resp.Connected = status.State == "active"

	inboxes, err := h.AgentMail.ListInboxes(r.Context(), member.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list agentmail inboxes")
		return
	}
	for _, inbox := range inboxes {
		resp.Inboxes = append(resp.Inboxes, agentMailInboxResponse(inbox))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListAgentMailDomains(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	member, ok := h.workspaceMember(w, r, workspaceID)
	if !ok {
		return
	}
	if h.AgentMail == nil || !h.AgentMail.Available() {
		writeJSON(w, http.StatusOK, AgentMailDomainListResponse{Domains: []string{"agentmail.to"}})
		return
	}
	domains, err := h.AgentMail.ListDomains(r.Context(), member.WorkspaceID)
	if err != nil {
		writeAgentMailError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, AgentMailDomainListResponse{Domains: nonNilStrings(domains)})
}

func (h *Handler) ConnectAgentMail(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	if !h.requireHumanAgentMailActor(w, r, workspaceID) {
		return
	}
	member, ok := h.workspaceMember(w, r, workspaceID)
	if !ok {
		return
	}
	if !roleAllowed(member.Role, "owner", "admin") {
		writeError(w, http.StatusForbidden, "only a workspace owner or admin can connect agentmail")
		return
	}
	if h.AgentMail == nil || !h.AgentMail.Available() {
		writeError(w, http.StatusServiceUnavailable, "agentmail not configured")
		return
	}

	var req connectAgentMailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cred, err := agentMailCredential(req, h.AgentMail.HostedAvailable())
	if err != nil {
		writeAgentMailError(w, err)
		return
	}

	status, err := h.AgentMail.Connect(r.Context(), member.WorkspaceID, member.UserID, cred)
	if err != nil {
		writeAgentMailError(w, err)
		return
	}
	resp := AgentMailWorkspaceResponse{
		Available:       true,
		HostedAvailable: h.AgentMail.HostedAvailable(),
		Connected:       status.State == "active",
		Source:          status.Source,
		State:           status.State,
		Domain:          status.Domain,
		CanManage:       true,
		Inboxes:         []AgentMailInboxResponse{},
	}
	if inboxes, err := h.AgentMail.ListInboxes(r.Context(), member.WorkspaceID); err == nil {
		for _, inbox := range inboxes {
			resp.Inboxes = append(resp.Inboxes, agentMailInboxResponse(inbox))
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) DisconnectAgentMail(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "id")
	if !h.requireHumanAgentMailActor(w, r, workspaceID) {
		return
	}
	member, ok := h.workspaceMember(w, r, workspaceID)
	if !ok {
		return
	}
	if !roleAllowed(member.Role, "owner", "admin") {
		writeError(w, http.StatusForbidden, "only a workspace owner or admin can disconnect agentmail")
		return
	}
	if h.AgentMail == nil || !h.AgentMail.Available() {
		writeError(w, http.StatusServiceUnavailable, "agentmail not configured")
		return
	}
	if err := h.AgentMail.Disconnect(r.Context(), member.WorkspaceID); err != nil {
		writeAgentMailError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetAgentMailInbox(w http.ResponseWriter, r *http.Request) {
	agent, ok := h.authorizeAgentMailRead(w, r)
	if !ok {
		return
	}
	if h.AgentMail == nil {
		writeJSON(w, http.StatusOK, AgentMailInboxResponse{AgentID: uuidToString(agent.ID)})
		return
	}
	inbox, err := h.AgentMail.GetInbox(r.Context(), agent.WorkspaceID, agent.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load agentmail inbox")
		return
	}
	if inbox == nil {
		writeJSON(w, http.StatusOK, AgentMailInboxResponse{AgentID: uuidToString(agent.ID)})
		return
	}
	writeJSON(w, http.StatusOK, agentMailInboxResponse(*inbox))
}

func (h *Handler) GrantAgentMailInbox(w http.ResponseWriter, r *http.Request) {
	agent, ok := h.authorizeAgentMailWrite(w, r)
	if !ok {
		return
	}
	if h.AgentMail == nil || !h.AgentMail.Available() {
		writeError(w, http.StatusServiceUnavailable, "agentmail not configured")
		return
	}
	var req grantAgentMailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	inbox, err := h.AgentMail.GrantInbox(r.Context(), agent.WorkspaceID, agent.ID, parseUUID(requestUserID(r)), agent.Name, agentmail.InboxAddress{
		Username: req.Username,
		Domain:   req.Domain,
	})
	if err != nil {
		writeAgentMailError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, agentMailInboxResponse(inbox))
}

func (h *Handler) ListAgentMailMailbox(w http.ResponseWriter, r *http.Request) {
	agent, ok := h.authorizeAgentMailRead(w, r)
	if !ok {
		return
	}
	if h.AgentMail == nil || !h.AgentMail.Available() {
		writeError(w, http.StatusServiceUnavailable, "agentmail not configured")
		return
	}
	q := r.URL.Query()
	page, err := h.AgentMail.ListMailbox(r.Context(), agent.WorkspaceID, agent.ID, q.Get("section"), q.Get("label"), q.Get("page_token"))
	if err != nil {
		writeAgentMailError(w, err)
		return
	}
	resp := AgentMailMailboxResponse{
		Items:         make([]AgentMailMailboxItemResponse, 0, len(page.Items)),
		NextPageToken: page.NextPageToken,
	}
	for _, item := range page.Items {
		resp.Items = append(resp.Items, AgentMailMailboxItemResponse{
			Kind:         item.Kind,
			ID:           item.ID,
			Subject:      item.Subject,
			Preview:      item.Preview,
			Participants: nonNilStrings(item.Participants),
			Timestamp:    item.Timestamp,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListAgentMailFolders(w http.ResponseWriter, r *http.Request) {
	agent, ok := h.authorizeAgentMailRead(w, r)
	if !ok {
		return
	}
	if h.AgentMail == nil || !h.AgentMail.Available() {
		writeError(w, http.StatusServiceUnavailable, "agentmail not configured")
		return
	}
	folders, err := h.AgentMail.ListFolders(r.Context(), agent.WorkspaceID, agent.ID)
	if err != nil {
		writeAgentMailError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, AgentMailFolderListResponse{Folders: nonNilStrings(folders)})
}

func (h *Handler) ListAgentMailThreads(w http.ResponseWriter, r *http.Request) {
	agent, ok := h.authorizeAgentMailRead(w, r)
	if !ok {
		return
	}
	if h.AgentMail == nil || !h.AgentMail.Available() {
		writeError(w, http.StatusServiceUnavailable, "agentmail not configured")
		return
	}
	page, err := h.AgentMail.ListThreads(r.Context(), agent.WorkspaceID, agent.ID, r.URL.Query().Get("page_token"))
	if err != nil {
		writeAgentMailError(w, err)
		return
	}
	resp := AgentMailThreadListResponse{
		Threads:       make([]AgentMailThreadResponse, 0, len(page.Threads)),
		NextPageToken: page.NextPageToken,
	}
	for _, thread := range page.Threads {
		resp.Threads = append(resp.Threads, agentMailThreadResponse(thread))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetAgentMailThread(w http.ResponseWriter, r *http.Request) {
	agent, ok := h.authorizeAgentMailRead(w, r)
	if !ok {
		return
	}
	if h.AgentMail == nil || !h.AgentMail.Available() {
		writeError(w, http.StatusServiceUnavailable, "agentmail not configured")
		return
	}
	detail, err := h.AgentMail.GetThread(r.Context(), agent.WorkspaceID, agent.ID, chi.URLParam(r, "threadId"))
	if err != nil {
		writeAgentMailError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, agentMailThreadDetailResponse(detail))
}

func (h *Handler) RevokeAgentMailInbox(w http.ResponseWriter, r *http.Request) {
	agent, ok := h.authorizeAgentMailWrite(w, r)
	if !ok {
		return
	}
	if h.AgentMail == nil || !h.AgentMail.Available() {
		writeError(w, http.StatusServiceUnavailable, "agentmail not configured")
		return
	}
	if err := h.AgentMail.RevokeInbox(r.Context(), agent.WorkspaceID, agent.ID); err != nil {
		writeAgentMailError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) authorizeAgentMailRead(w http.ResponseWriter, r *http.Request) (db.Agent, bool) {
	agent, ok := h.loadAgentForUser(w, r, chi.URLParam(r, "id"))
	if !ok {
		return db.Agent{}, false
	}
	workspaceID := uuidToString(agent.WorkspaceID)
	userID := requestUserID(r)
	actorType, _ := h.resolveActor(r, userID, workspaceID)
	if actorType == "agent" {
		writeError(w, http.StatusForbidden, "agents may not access agentmail")
		return db.Agent{}, false
	}
	member, ok := h.workspaceMember(w, r, workspaceID)
	if !ok {
		return db.Agent{}, false
	}
	if !canViewAgentSecrets(agent, userID, member.Role) {
		writeError(w, http.StatusForbidden, "only the agent owner or a workspace owner/admin can view this inbox")
		return db.Agent{}, false
	}
	return agent, true
}

func (h *Handler) authorizeAgentMailWrite(w http.ResponseWriter, r *http.Request) (db.Agent, bool) {
	agent, ok := h.loadAgentForUser(w, r, chi.URLParam(r, "id"))
	if !ok {
		return db.Agent{}, false
	}
	workspaceID := uuidToString(agent.WorkspaceID)
	if !h.requireHumanAgentMailActor(w, r, workspaceID) {
		return db.Agent{}, false
	}
	if !h.canManageAgent(w, r, agent) {
		return db.Agent{}, false
	}
	return agent, true
}

func (h *Handler) requireHumanAgentMailActor(w http.ResponseWriter, r *http.Request, workspaceID string) bool {
	if isMachineCredentialActor(r) {
		writeError(w, http.StatusForbidden, "this endpoint is only available to human actors")
		return false
	}
	actorType, _ := h.resolveActor(r, requestUserID(r), workspaceID)
	if actorType == "agent" {
		writeError(w, http.StatusForbidden, "agents may not access agentmail")
		return false
	}
	return true
}

func agentMailCredential(req connectAgentMailRequest, hostedAvailable bool) (agentmail.WorkspaceCredential, error) {
	switch req.Mode {
	case "", "hosted":
		if req.OrgKey != "" {
			return nil, agentmail.ErrBadOrgKey
		}
		if !hostedAvailable {
			return nil, agentmail.ErrHostedUnavailable
		}
		return agentmail.HostedCredential(), nil
	case "bring_your_own":
		return agentmail.ParseBYOCredential(req.OrgKey)
	default:
		return nil, errAgentMailBadMode
	}
}

var errAgentMailBadMode = errors.New("agentmail: unknown connect mode")

func agentMailInboxResponse(inbox agentmail.Inbox) AgentMailInboxResponse {
	resp := AgentMailInboxResponse{
		AgentID:     inbox.AgentID,
		Enabled:     inbox.State == "active",
		State:       inbox.State,
		DisplayName: inbox.DisplayName,
	}
	if inbox.State == "active" {
		resp.Address = inbox.Address
	}
	return resp
}

func agentMailThreadResponse(thread agentmail.Thread) AgentMailThreadResponse {
	return AgentMailThreadResponse{
		ThreadID:     thread.ID,
		Subject:      thread.Subject,
		Preview:      thread.Preview,
		Senders:      nonNilStrings(thread.Senders),
		Recipients:   nonNilStrings(thread.Recipients),
		Timestamp:    thread.Timestamp,
		MessageCount: thread.MessageCount,
	}
}

func agentMailThreadDetailResponse(detail agentmail.ThreadDetail) AgentMailThreadDetailResponse {
	resp := AgentMailThreadDetailResponse{
		AgentMailThreadResponse: agentMailThreadResponse(detail.Thread),
		Messages:                make([]AgentMailMessageResponse, 0, len(detail.Messages)),
	}
	for _, msg := range detail.Messages {
		resp.Messages = append(resp.Messages, AgentMailMessageResponse{
			MessageID: msg.ID,
			From:      msg.From,
			To:        nonNilStrings(msg.To),
			Timestamp: msg.Timestamp,
			Subject:   msg.Subject,
			Text:      msg.Text,
		})
	}
	return resp
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func writeAgentMailError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, agentmail.ErrUnavailable):
		writeError(w, http.StatusServiceUnavailable, "agentmail not configured")
	case errors.Is(err, agentmail.ErrHostedUnavailable):
		writeError(w, http.StatusBadRequest, "hosted agentmail is not configured")
	case errors.Is(err, agentmail.ErrBadOrgKey), errors.Is(err, errAgentMailBadMode):
		writeError(w, http.StatusBadRequest, "invalid agentmail credential")
	case errors.Is(err, agentmail.ErrModeConflict):
		writeError(w, http.StatusConflict, "disconnect before switching agentmail mode")
	case errors.Is(err, agentmail.ErrNotConnected):
		writeError(w, http.StatusConflict, "workspace is not connected to agentmail")
	case errors.Is(err, agentmail.ErrInboxQuota):
		writeError(w, http.StatusConflict, "workspace inbox limit reached")
	case errors.Is(err, agentmail.ErrInboxNotActive):
		writeError(w, http.StatusConflict, "inbox is not active")
	case errors.Is(err, agentmail.ErrRemoteNotFound):
		writeError(w, http.StatusNotFound, "mail not found")
	case errors.Is(err, agentmail.ErrBadAddress):
		writeError(w, http.StatusBadRequest, "choose a valid email username and domain")
	case errors.Is(err, agentmail.ErrAddressTaken):
		writeError(w, http.StatusConflict, "that email address is taken")
	case errors.Is(err, agentmail.ErrBadMailbox):
		writeError(w, http.StatusBadRequest, "unknown mailbox section")
	default:
		writeError(w, http.StatusInternalServerError, "agentmail request failed")
	}
}
