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

type connectAgentMailRequest struct {
	Mode   string `json:"mode"`
	OrgKey string `json:"org_key"`
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
	inbox, err := h.AgentMail.GrantInbox(r.Context(), agent.WorkspaceID, agent.ID, parseUUID(requestUserID(r)), agent.Name)
	if err != nil {
		writeAgentMailError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, agentMailInboxResponse(inbox))
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
	default:
		writeError(w, http.StatusInternalServerError, "agentmail request failed")
	}
}
