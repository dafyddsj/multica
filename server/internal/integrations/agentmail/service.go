package agentmail

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/util"
	"github.com/multica-ai/multica/server/internal/util/secretbox"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

const (
	sourceHosted = "hosted"
	sourceBYO    = "bring_your_own"

	stateProvisioning = "provisioning"
	stateMintingKey   = "minting_key"
	stateActive       = "active"
	stateDisabling    = "disabling"
	stateDisabled     = "disabled"

	authorityHostedOrg = "hosted_org"
	authorityBYOOrg    = "byo_org"
	authorityBYOPod    = "byo_pod"

	defaultInboxLimit = 5
	defaultMCPURL     = "https://mcp.agentmail.to/mcp"
)

var (
	ErrUnavailable       = errors.New("agentmail: unavailable")
	ErrNotConnected      = errors.New("agentmail: workspace not connected")
	ErrModeConflict      = errors.New("agentmail: disconnect before switching mode")
	ErrInboxQuota        = errors.New("agentmail: workspace inbox limit reached")
	ErrHostedUnavailable = errors.New("agentmail: hosted mode not configured")
	ErrBadOrgKey         = errors.New("agentmail: organization key rejected")
	ErrInboxNotActive    = errors.New("agentmail: inbox not active")
)

// WorkspaceCredential is a closed connect input. Hosted cannot carry a key.
type WorkspaceCredential interface {
	workspaceCredential()
}

type hostedCredential struct{}

type byoCredential struct {
	orgKey string
}

func (hostedCredential) workspaceCredential() {}
func (byoCredential) workspaceCredential()    {}

func HostedCredential() WorkspaceCredential {
	return hostedCredential{}
}

func ParseBYOCredential(raw string) (WorkspaceCredential, error) {
	key := strings.TrimSpace(raw)
	if key == "" {
		return nil, ErrBadOrgKey
	}
	return byoCredential{orgKey: key}, nil
}

type Config struct {
	Box                 *secretbox.Box
	HostedOrgKey        string
	WorkspaceInboxLimit int
	MCPURL              string
	APIBaseURL          string
	HTTPClient          *http.Client
}

type WorkspaceStatus struct {
	Source string
	State  string
	Domain string
}

type Inbox struct {
	AgentID     string
	State       string
	Address     string
	DisplayName string
}

type Thread struct {
	ID           string
	Subject      string
	Preview      string
	Senders      []string
	Recipients   []string
	Timestamp    string
	MessageCount int
}

type ThreadPage struct {
	Threads       []Thread
	NextPageToken string
}

type Message struct {
	ID        string
	From      string
	To        []string
	Timestamp string
	Subject   string
	Text      string
}

type ThreadDetail struct {
	Thread
	Messages []Message
}

type Service struct {
	cfg Config
	q   *db.Queries
	api apiClient

	failPersistActiveOnce bool
}

func New(cfg Config, q *db.Queries) (*Service, error) {
	return newService(cfg, q, newLiveClient(cfg.HTTPClient, cfg.APIBaseURL))
}

// NewMemory builds a service backed by an in-process remote. Production uses New.
func NewMemory(cfg Config, q *db.Queries) (*Service, error) {
	return newService(cfg, q, newMemoryClient())
}

func newService(cfg Config, q *db.Queries, api apiClient) (*Service, error) {
	if q == nil {
		return nil, errors.New("agentmail: queries required")
	}
	if cfg.WorkspaceInboxLimit <= 0 {
		cfg.WorkspaceInboxLimit = defaultInboxLimit
	}
	if cfg.MCPURL == "" {
		cfg.MCPURL = defaultMCPURL
	}
	return &Service{cfg: cfg, q: q, api: api}, nil
}

func (s *Service) HostedAvailable() bool {
	return strings.TrimSpace(s.cfg.HostedOrgKey) != ""
}

func (s *Service) Available() bool {
	return s.cfg.Box != nil
}

func (s *Service) GetWorkspace(ctx context.Context, wsID pgtype.UUID) (WorkspaceStatus, error) {
	row, err := s.q.GetAgentMailConnectionByWorkspace(ctx, wsID)
	if isNoRows(err) {
		return WorkspaceStatus{}, nil
	}
	if err != nil {
		return WorkspaceStatus{}, err
	}
	return connectionStatus(row), nil
}

func (s *Service) Connect(ctx context.Context, wsID, actorID pgtype.UUID, cred WorkspaceCredential) (WorkspaceStatus, error) {
	if !s.Available() {
		return WorkspaceStatus{}, ErrUnavailable
	}
	switch c := cred.(type) {
	case hostedCredential:
		return s.connectHosted(ctx, wsID, actorID)
	case byoCredential:
		return s.connectBYO(ctx, wsID, actorID, c.orgKey)
	default:
		return WorkspaceStatus{}, ErrUnavailable
	}
}

func (s *Service) connectHosted(ctx context.Context, wsID, actorID pgtype.UUID) (WorkspaceStatus, error) {
	if !s.HostedAvailable() {
		return WorkspaceStatus{}, ErrHostedUnavailable
	}
	existing, err := s.q.GetAgentMailConnectionByWorkspace(ctx, wsID)
	if err != nil && !isNoRows(err) {
		return WorkspaceStatus{}, err
	}
	if err == nil && existing.State == stateActive {
		if existing.Source == sourceHosted {
			return connectionStatus(existing), nil
		}
		return WorkspaceStatus{}, ErrModeConflict
	}

	clientID := util.UUIDToString(wsID)
	_, err = s.q.UpsertAgentMailConnection(ctx, db.UpsertAgentMailConnectionParams{
		WorkspaceID:   wsID,
		Source:        sourceHosted,
		State:         stateProvisioning,
		AuthorityKind: authorityHostedOrg,
		PodClientID:   clientID,
		ConnectedByID: actorID,
	})
	if err != nil {
		return WorkspaceStatus{}, err
	}

	podID, err := s.api.ensurePod(ctx, s.cfg.HostedOrgKey, clientID)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	row, err := s.q.UpsertAgentMailConnection(ctx, db.UpsertAgentMailConnectionParams{
		WorkspaceID:   wsID,
		Source:        sourceHosted,
		State:         stateActive,
		AuthorityKind: authorityHostedOrg,
		PodID:         textValue(podID),
		PodClientID:   clientID,
		ConnectedByID: actorID,
	})
	if err != nil {
		return WorkspaceStatus{}, err
	}
	return connectionStatus(row), nil
}

func (s *Service) connectBYO(ctx context.Context, wsID, actorID pgtype.UUID, orgKey string) (WorkspaceStatus, error) {
	info, err := s.api.inspectKey(ctx, orgKey)
	if err != nil {
		if errors.Is(err, ErrBadOrgKey) {
			return WorkspaceStatus{}, err
		}
		return WorkspaceStatus{}, ErrBadOrgKey
	}

	existing, err := s.q.GetAgentMailConnectionByWorkspace(ctx, wsID)
	if err != nil && !isNoRows(err) {
		return WorkspaceStatus{}, err
	}
	if err == nil && existing.State == stateActive {
		if existing.Source == sourceBYO {
			return connectionStatus(existing), nil
		}
		return WorkspaceStatus{}, ErrModeConflict
	}

	sealed, err := sealOrgKey(s.cfg.Box, orgKey)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	kind := info.authorityKind
	if kind == "" {
		kind = authorityBYOOrg
	}
	row, err := s.q.UpsertAgentMailConnection(ctx, db.UpsertAgentMailConnectionParams{
		WorkspaceID:     wsID,
		Source:          sourceBYO,
		State:           stateActive,
		AuthorityKind:   kind,
		PodID:           textValue(info.podID),
		OrgKeyEncrypted: textValue(sealed),
		PodClientID:     util.UUIDToString(wsID),
		Domain:          info.domain,
		ConnectedByID:   actorID,
	})
	if err != nil {
		return WorkspaceStatus{}, err
	}
	return connectionStatus(row), nil
}

func (s *Service) Disconnect(ctx context.Context, wsID pgtype.UUID) error {
	if !s.Available() {
		return ErrUnavailable
	}
	conn, err := s.q.GetAgentMailConnectionByWorkspace(ctx, wsID)
	if isNoRows(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if conn.State == stateDisabled {
		return nil
	}

	inboxes, err := s.q.ListAgentMailInboxesByWorkspace(ctx, wsID)
	if err != nil {
		return err
	}
	for _, inbox := range inboxes {
		if err := s.revokeInbox(ctx, s.q, conn, inbox); err != nil {
			return err
		}
	}

	if conn.Source == sourceHosted && textString(conn.PodID) != "" {
		if err := s.insertPurge(ctx, s.q, wsID, "pod", textString(conn.PodID), conn); err != nil {
			return err
		}
		if err := s.api.deletePod(ctx, s.cfg.HostedOrgKey, textString(conn.PodID)); err != nil && !errors.Is(err, errRemoteNotFound) {
			return err
		}
	}

	_, err = s.q.UpsertAgentMailConnection(ctx, db.UpsertAgentMailConnectionParams{
		WorkspaceID:   wsID,
		Source:        conn.Source,
		State:         stateDisabled,
		AuthorityKind: conn.AuthorityKind,
		PodID:         conn.PodID,
		PodClientID:   conn.PodClientID,
		Domain:        conn.Domain,
		ConnectedByID: conn.ConnectedByID,
	})
	return err
}

func (s *Service) GetInbox(ctx context.Context, wsID, agentID pgtype.UUID) (*Inbox, error) {
	row, err := s.q.GetAgentMailInbox(ctx, db.GetAgentMailInboxParams{
		WorkspaceID: wsID,
		AgentID:     agentID,
	})
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	view := inboxView(row)
	return &view, nil
}

func (s *Service) ListInboxes(ctx context.Context, wsID pgtype.UUID) ([]Inbox, error) {
	rows, err := s.q.ListAgentMailInboxesByWorkspace(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := make([]Inbox, 0, len(rows))
	for _, row := range rows {
		out = append(out, inboxView(row))
	}
	return out, nil
}

func (s *Service) GrantInbox(ctx context.Context, wsID, agentID, actorID pgtype.UUID, agentName string) (Inbox, error) {
	if !s.Available() {
		return Inbox{}, ErrUnavailable
	}
	conn, err := s.q.GetAgentMailConnectionByWorkspace(ctx, wsID)
	if isNoRows(err) || (err == nil && conn.State != stateActive) {
		return Inbox{}, ErrNotConnected
	}
	if err != nil {
		return Inbox{}, err
	}

	existing, err := s.q.GetAgentMailInbox(ctx, db.GetAgentMailInboxParams{
		WorkspaceID: wsID,
		AgentID:     agentID,
	})
	hasRow := err == nil
	if err != nil && !isNoRows(err) {
		return Inbox{}, err
	}
	if hasRow && existing.State == stateActive {
		return inboxView(existing), nil
	}

	if conn.Source == sourceHosted {
		used, err := s.q.CountAgentMailInboxesInFlight(ctx, wsID)
		if err != nil {
			return Inbox{}, err
		}
		inFlight := hasRow && isInFlight(existing.State)
		if used >= int64(s.cfg.WorkspaceInboxLimit) && !inFlight {
			return Inbox{}, ErrInboxQuota
		}
	}

	clientID := util.UUIDToString(agentID)
	if hasRow && existing.ClientID != "" {
		clientID = existing.ClientID
	}

	cred, err := s.authority(conn)
	if err != nil {
		return Inbox{}, err
	}

	if hasRow && existing.State == stateMintingKey && textString(existing.RemoteInboxID) != "" {
		if err := s.api.deleteInbox(ctx, cred, textString(existing.RemoteInboxID)); err != nil && !errors.Is(err, errRemoteNotFound) {
			return Inbox{}, err
		}
	}

	_, err = s.q.UpsertAgentMailInbox(ctx, db.UpsertAgentMailInboxParams{
		WorkspaceID: wsID,
		AgentID:     agentID,
		ClientID:    clientID,
		State:       stateProvisioning,
		DisplayName: agentName,
		CreatedByID: actorID,
	})
	if err != nil {
		return Inbox{}, err
	}

	remote, err := s.api.ensureInbox(ctx, cred, clientID, agentName)
	if err != nil {
		return Inbox{}, err
	}

	attempt := util.MustParseUUID(uuid.NewString())
	_, err = s.q.UpsertAgentMailInbox(ctx, db.UpsertAgentMailInboxParams{
		WorkspaceID:   wsID,
		AgentID:       agentID,
		ClientID:      clientID,
		State:         stateMintingKey,
		RemoteInboxID: textValue(remote.id),
		Address:       textValue(remote.address),
		DisplayName:   agentName,
		KeyAttemptID:  attempt,
		CreatedByID:   actorID,
	})
	if err != nil {
		return Inbox{}, err
	}

	plainKey, err := s.api.createInboxKey(ctx, cred, remote.id)
	if err != nil {
		return Inbox{}, err
	}
	if s.failPersistActiveOnce {
		s.failPersistActiveOnce = false
		return Inbox{}, errors.New("agentmail: persist active failed")
	}

	sealed, err := sealInboxKey(s.cfg.Box, plainKey)
	if err != nil {
		return Inbox{}, err
	}
	if remote.id == "" || remote.address == "" || sealed == "" {
		return Inbox{}, errors.New("agentmail: persist active missing fields")
	}
	row, err := s.q.UpsertAgentMailInbox(ctx, db.UpsertAgentMailInboxParams{
		WorkspaceID:       wsID,
		AgentID:           agentID,
		ClientID:          clientID,
		State:             stateActive,
		RemoteInboxID:     textValue(remote.id),
		Address:           textValue(remote.address),
		DisplayName:       agentName,
		InboxKeyEncrypted: textValue(sealed),
		CreatedByID:       actorID,
	})
	if err != nil {
		return Inbox{}, err
	}
	return inboxView(row), nil
}

func (s *Service) RevokeInbox(ctx context.Context, wsID, agentID pgtype.UUID) error {
	if !s.Available() {
		return ErrUnavailable
	}
	conn, err := s.q.GetAgentMailConnectionByWorkspace(ctx, wsID)
	if isNoRows(err) {
		return nil
	}
	if err != nil {
		return err
	}
	inbox, err := s.q.GetAgentMailInbox(ctx, db.GetAgentMailInboxParams{
		WorkspaceID: wsID,
		AgentID:     agentID,
	})
	if isNoRows(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.revokeInbox(ctx, s.q, conn, inbox)
}

func (s *Service) ClaimOverlay(ctx context.Context, wsID, agentID pgtype.UUID) (json.RawMessage, error) {
	if !s.Available() {
		return nil, ErrUnavailable
	}
	enc, err := s.q.GetAgentMailActiveInboxKey(ctx, db.GetAgentMailActiveInboxKeyParams{
		WorkspaceID: wsID,
		AgentID:     agentID,
	})
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !enc.Valid || enc.String == "" {
		return nil, nil
	}
	key, err := openInboxKey(s.cfg.Box, enc.String)
	if err != nil {
		return nil, err
	}
	return claimOverlayJSON(s.cfg.MCPURL, key)
}

func (s *Service) ListThreads(ctx context.Context, wsID, agentID pgtype.UUID, pageToken string) (ThreadPage, error) {
	inboxID, key, err := s.activeInboxAccess(ctx, wsID, agentID)
	if err != nil {
		return ThreadPage{}, err
	}
	return s.api.listThreads(ctx, key, inboxID, pageToken)
}

func (s *Service) GetThread(ctx context.Context, wsID, agentID pgtype.UUID, threadID string) (ThreadDetail, error) {
	if strings.TrimSpace(threadID) == "" {
		return ThreadDetail{}, errRemoteNotFound
	}
	inboxID, key, err := s.activeInboxAccess(ctx, wsID, agentID)
	if err != nil {
		return ThreadDetail{}, err
	}
	return s.api.getThread(ctx, key, inboxID, threadID)
}

func (s *Service) activeInboxAccess(ctx context.Context, wsID, agentID pgtype.UUID) (string, string, error) {
	if !s.Available() {
		return "", "", ErrUnavailable
	}
	conn, err := s.q.GetAgentMailConnectionByWorkspace(ctx, wsID)
	if isNoRows(err) || (err == nil && conn.State != stateActive) {
		return "", "", ErrNotConnected
	}
	if err != nil {
		return "", "", err
	}
	row, err := s.q.GetAgentMailInbox(ctx, db.GetAgentMailInboxParams{
		WorkspaceID: wsID,
		AgentID:     agentID,
	})
	if isNoRows(err) || (err == nil && row.State != stateActive) {
		return "", "", ErrInboxNotActive
	}
	if err != nil {
		return "", "", err
	}
	remote := textString(row.RemoteInboxID)
	if remote == "" || !row.InboxKeyEncrypted.Valid || row.InboxKeyEncrypted.String == "" {
		return "", "", ErrInboxNotActive
	}
	key, err := openInboxKey(s.cfg.Box, row.InboxKeyEncrypted.String)
	if err != nil {
		return "", "", err
	}
	return remote, key, nil
}

func (s *Service) SweepWorkspace(ctx context.Context, qtx *db.Queries, wsID pgtype.UUID) error {
	q := s.store(qtx)
	conn, err := q.GetAgentMailConnectionByWorkspace(ctx, wsID)
	hasConn := err == nil
	if err != nil && !isNoRows(err) {
		return err
	}
	inboxes, err := q.ListAgentMailInboxesByWorkspace(ctx, wsID)
	if err != nil {
		return err
	}
	for _, inbox := range inboxes {
		if remote := textString(inbox.RemoteInboxID); remote != "" && hasConn {
			if err := s.insertPurge(ctx, q, wsID, "inbox", remote, conn); err != nil {
				return err
			}
		}
	}
	if hasConn && conn.Source == sourceHosted && textString(conn.PodID) != "" {
		if err := s.insertPurge(ctx, q, wsID, "pod", textString(conn.PodID), conn); err != nil {
			return err
		}
	}
	if err := q.DeleteAgentMailInboxesByWorkspace(ctx, wsID); err != nil {
		return err
	}
	return q.DeleteAgentMailConnectionByWorkspace(ctx, wsID)
}

func (s *Service) SweepAgent(ctx context.Context, qtx *db.Queries, wsID, agentID pgtype.UUID) error {
	q := s.store(qtx)
	inbox, err := q.GetAgentMailInbox(ctx, db.GetAgentMailInboxParams{
		WorkspaceID: wsID,
		AgentID:     agentID,
	})
	if isNoRows(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if remote := textString(inbox.RemoteInboxID); remote != "" {
		conn, cerr := q.GetAgentMailConnectionByWorkspace(ctx, wsID)
		if cerr != nil && !isNoRows(cerr) {
			return cerr
		}
		if cerr == nil {
			if err := s.insertPurge(ctx, q, wsID, "inbox", remote, conn); err != nil {
				return err
			}
		}
	}
	return q.DeleteAgentMailInbox(ctx, db.DeleteAgentMailInboxParams{
		WorkspaceID: wsID,
		AgentID:     agentID,
	})
}

func (s *Service) revokeInbox(ctx context.Context, q *db.Queries, conn db.AgentmailConnection, inbox db.AgentmailInbox) error {
	if inbox.State == stateDisabled {
		return nil
	}
	_, err := q.UpsertAgentMailInbox(ctx, db.UpsertAgentMailInboxParams{
		WorkspaceID:   inbox.WorkspaceID,
		AgentID:       inbox.AgentID,
		ClientID:      inbox.ClientID,
		State:         stateDisabling,
		RemoteInboxID: inbox.RemoteInboxID,
		Address:       inbox.Address,
		DisplayName:   inbox.DisplayName,
		CreatedByID:   inbox.CreatedByID,
	})
	if err != nil {
		return err
	}

	if remote := textString(inbox.RemoteInboxID); remote != "" {
		if err := s.insertPurge(ctx, q, inbox.WorkspaceID, "inbox", remote, conn); err != nil {
			return err
		}
		cred, err := s.authority(conn)
		if err != nil {
			return err
		}
		if err := s.api.deleteInbox(ctx, cred, remote); err != nil && !errors.Is(err, errRemoteNotFound) {
			return err
		}
	}

	_, err = q.UpsertAgentMailInbox(ctx, db.UpsertAgentMailInboxParams{
		WorkspaceID:   inbox.WorkspaceID,
		AgentID:       inbox.AgentID,
		ClientID:      inbox.ClientID,
		State:         stateDisabled,
		RemoteInboxID: inbox.RemoteInboxID,
		Address:       inbox.Address,
		DisplayName:   inbox.DisplayName,
		CreatedByID:   inbox.CreatedByID,
	})
	return err
}

func (s *Service) insertPurge(ctx context.Context, q *db.Queries, wsID pgtype.UUID, kind, remoteID string, conn db.AgentmailConnection) error {
	var sealed pgtype.Text
	if conn.Source == sourceBYO {
		sealed = conn.OrgKeyEncrypted
	}
	_, err := q.InsertAgentMailPurge(ctx, db.InsertAgentMailPurgeParams{
		WorkspaceID:     wsID,
		Kind:            kind,
		RemoteID:        remoteID,
		Source:          conn.Source,
		OrgKeyEncrypted: sealed,
	})
	return err
}

func (s *Service) authority(conn db.AgentmailConnection) (clientCred, error) {
	if conn.Source == sourceHosted {
		return clientCred{apiKey: s.cfg.HostedOrgKey, podID: textString(conn.PodID)}, nil
	}
	if !conn.OrgKeyEncrypted.Valid || conn.OrgKeyEncrypted.String == "" {
		return clientCred{}, ErrNotConnected
	}
	key, err := openOrgKey(s.cfg.Box, conn.OrgKeyEncrypted.String)
	if err != nil {
		return clientCred{}, err
	}
	return clientCred{apiKey: key, podID: textString(conn.PodID)}, nil
}

func (s *Service) store(qtx *db.Queries) *db.Queries {
	if qtx != nil {
		return qtx
	}
	return s.q
}

func connectionStatus(row db.AgentmailConnection) WorkspaceStatus {
	return WorkspaceStatus{Source: row.Source, State: row.State, Domain: row.Domain}
}

func inboxView(row db.AgentmailInbox) Inbox {
	return Inbox{
		AgentID:     util.UUIDToString(row.AgentID),
		State:       row.State,
		Address:     textString(row.Address),
		DisplayName: row.DisplayName,
	}
}

func isInFlight(state string) bool {
	switch state {
	case stateProvisioning, stateMintingKey, stateActive, stateDisabling:
		return true
	default:
		return false
	}
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func textString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func textValue(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}
