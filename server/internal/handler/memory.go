package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/logger"
	"github.com/multica-ai/multica/server/internal/memory"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type MemoryEntryResponse struct {
	ID            string         `json:"id"`
	WorkspaceID   *string        `json:"workspace_id"`
	Scope         string         `json:"scope"`
	OwnerID       string         `json:"owner_id"`
	Body          string         `json:"body"`
	Kind          string         `json:"kind"`
	Provenance    map[string]any `json:"provenance"`
	CreatedByType *string        `json:"created_by_type,omitempty"`
	CreatedByID   *string        `json:"created_by_id,omitempty"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
}

type MemoryListResponse struct {
	Entries []MemoryEntryResponse `json:"entries"`
	Total   int64                 `json:"total"`
}

type MemoryRecallResponse struct {
	Hits []memory.Hit `json:"hits"`
}

type CreateMemoryRequest struct {
	Scope      string         `json:"scope"`
	OwnerID    string         `json:"owner_id"`
	Body       string         `json:"body"`
	Kind       string         `json:"kind"`
	Provenance map[string]any `json:"provenance"`
}

type UpdateMemoryRequest struct {
	Body       *string        `json:"body"`
	Kind       *string        `json:"kind"`
	Provenance map[string]any `json:"provenance"`
}

func memoryEntryToResponse(row db.MemoryEntry) MemoryEntryResponse {
	var wsID *string
	if row.WorkspaceID.Valid {
		s := uuidToString(row.WorkspaceID)
		wsID = &s
	}
	prov := map[string]any{}
	if len(row.Provenance) > 0 {
		_ = json.Unmarshal(row.Provenance, &prov)
	}
	return MemoryEntryResponse{
		ID:            uuidToString(row.ID),
		WorkspaceID:   wsID,
		Scope:         row.Scope,
		OwnerID:       uuidToString(row.OwnerID),
		Body:          row.Body,
		Kind:          row.Kind,
		Provenance:    prov,
		CreatedByType: textToPtr(row.CreatedByType),
		CreatedByID:   uuidToPtr(row.CreatedByID),
		CreatedAt:     timestampToString(row.CreatedAt),
		UpdatedAt:     timestampToString(row.UpdatedAt),
	}
}

func (h *Handler) requireMemoryAvailable(w http.ResponseWriter, r *http.Request) (db.Workspace, bool) {
	workspaceID := h.resolveWorkspaceID(r)
	if workspaceID == "" {
		writeError(w, http.StatusBadRequest, "workspace is required")
		return db.Workspace{}, false
	}
	ws, err := h.Queries.GetWorkspace(r.Context(), parseUUID(workspaceID))
	if err != nil {
		writeError(w, http.StatusNotFound, "workspace not found")
		return db.Workspace{}, false
	}
	if !memory.Available(r.Context(), h.FeatureFlags, ws.Settings) {
		writeError(w, http.StatusNotFound, "memory is not enabled")
		return db.Workspace{}, false
	}
	return ws, true
}

func (h *Handler) ListMemory(w http.ResponseWriter, r *http.Request) {
	ws, ok := h.requireMemoryAvailable(w, r)
	if !ok {
		return
	}
	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	ownerRaw := strings.TrimSpace(r.URL.Query().Get("owner_id"))
	if !memory.ValidScope(scope) {
		writeError(w, http.StatusBadRequest, "scope is required")
		return
	}
	ownerID, ok := parseUUIDOrBadRequest(w, ownerRaw, "owner_id")
	if !ok {
		return
	}
	if !h.authorizeMemory(w, r, ws, scope, ownerID, false) {
		return
	}
	limit := memory.ClampListLimit(queryInt32(r, "limit"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	rows, err := h.Queries.ListMemoryEntries(r.Context(), db.ListMemoryEntriesParams{
		Scope:       scope,
		OwnerID:     ownerID,
		Limit:       limit,
		WorkspaceID: listWorkspaceArg(ws.ID, scope),
		Query:       optionalTextArg(q),
	})
	if err != nil {
		slog.Warn("ListMemory failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to list memory")
		return
	}
	total, err := h.Queries.CountMemoryEntriesInBank(r.Context(), db.CountMemoryEntriesInBankParams{
		Scope:       scope,
		OwnerID:     ownerID,
		WorkspaceID: listWorkspaceArg(ws.ID, scope),
	})
	if err != nil {
		slog.Warn("CountMemory failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to list memory")
		return
	}
	out := MemoryListResponse{Entries: make([]MemoryEntryResponse, 0, len(rows)), Total: total}
	for _, row := range rows {
		out.Entries = append(out.Entries, memoryEntryToResponse(row))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) CreateMemory(w http.ResponseWriter, r *http.Request) {
	ws, ok := h.requireMemoryAvailable(w, r)
	if !ok {
		return
	}
	var req CreateMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Scope = strings.TrimSpace(req.Scope)
	req.Kind = memory.NormalizeKind(req.Kind)
	if !memory.ValidScope(req.Scope) {
		writeError(w, http.StatusBadRequest, "scope is required")
		return
	}
	if !memory.ValidKind(req.Kind) {
		writeError(w, http.StatusBadRequest, "kind must be fact, preference, procedure, or observation")
		return
	}
	if msg := memory.ValidateBody(req.Body); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	ownerID, ok := parseUUIDOrBadRequest(w, strings.TrimSpace(req.OwnerID), "owner_id")
	if !ok {
		return
	}
	if !h.authorizeMemory(w, r, ws, req.Scope, ownerID, true) {
		return
	}
	if !h.ownerExists(w, r, ws.ID, req.Scope, ownerID) {
		return
	}
	count, err := h.Queries.CountMemoryEntriesInBank(r.Context(), db.CountMemoryEntriesInBankParams{
		Scope:       req.Scope,
		OwnerID:     ownerID,
		WorkspaceID: listWorkspaceArg(ws.ID, req.Scope),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create memory")
		return
	}
	if count >= memory.MaxBankEntries {
		writeError(w, http.StatusBadRequest, "this bank already has 200 entries")
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	actorType, actorID := h.resolveActor(r, userID, uuidToString(ws.ID))
	prov, err := marshalProvenance(req.Provenance)
	if err != nil {
		writeError(w, http.StatusBadRequest, "provenance must be a JSON object")
		return
	}
	row, err := h.Queries.CreateMemoryEntry(r.Context(), db.CreateMemoryEntryParams{
		WorkspaceID:   createWorkspaceArg(ws.ID, req.Scope),
		Scope:         req.Scope,
		OwnerID:       ownerID,
		Body:          strings.TrimSpace(req.Body),
		Kind:          req.Kind,
		Provenance:    prov,
		CreatedByType: pgtype.Text{String: actorType, Valid: actorType != ""},
		CreatedByID:   parseUUID(actorID),
	})
	if err != nil {
		slog.Warn("CreateMemory failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to create memory")
		return
	}
	writeJSON(w, http.StatusCreated, memoryEntryToResponse(row))
}

func (h *Handler) GetMemory(w http.ResponseWriter, r *http.Request) {
	ws, ok := h.requireMemoryAvailable(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}
	row, err := h.Queries.GetMemoryEntry(r.Context(), db.GetMemoryEntryParams{
		ID: id, WorkspaceID: ws.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "memory not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load memory")
		return
	}
	if !h.authorizeMemory(w, r, ws, row.Scope, row.OwnerID, false) {
		return
	}
	writeJSON(w, http.StatusOK, memoryEntryToResponse(row))
}

func (h *Handler) UpdateMemory(w http.ResponseWriter, r *http.Request) {
	ws, ok := h.requireMemoryAvailable(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}
	var req UpdateMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	row, err := h.Queries.GetMemoryEntry(r.Context(), db.GetMemoryEntryParams{
		ID: id, WorkspaceID: ws.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "memory not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load memory")
		return
	}
	if !h.authorizeMemory(w, r, ws, row.Scope, row.OwnerID, true) {
		return
	}
	params := db.UpdateMemoryEntryParams{ID: id, WorkspaceID: ws.ID}
	if req.Body != nil {
		if msg := memory.ValidateBody(*req.Body); msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		params.Body = pgtype.Text{String: strings.TrimSpace(*req.Body), Valid: true}
	}
	if req.Kind != nil {
		kind := memory.NormalizeKind(*req.Kind)
		if !memory.ValidKind(kind) {
			writeError(w, http.StatusBadRequest, "kind must be fact, preference, procedure, or observation")
			return
		}
		params.Kind = pgtype.Text{String: kind, Valid: true}
	}
	if req.Provenance != nil {
		prov, err := marshalProvenance(req.Provenance)
		if err != nil {
			writeError(w, http.StatusBadRequest, "provenance must be a JSON object")
			return
		}
		params.Provenance = prov
	}
	updated, err := h.Queries.UpdateMemoryEntry(r.Context(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "memory not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update memory")
		return
	}
	writeJSON(w, http.StatusOK, memoryEntryToResponse(updated))
}

func (h *Handler) DeleteMemory(w http.ResponseWriter, r *http.Request) {
	ws, ok := h.requireMemoryAvailable(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}
	row, err := h.Queries.GetMemoryEntry(r.Context(), db.GetMemoryEntryParams{
		ID: id, WorkspaceID: ws.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "memory not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load memory")
		return
	}
	if !h.authorizeMemory(w, r, ws, row.Scope, row.OwnerID, true) {
		return
	}
	if _, err := h.Queries.SoftDeleteMemoryEntry(r.Context(), db.SoftDeleteMemoryEntryParams{
		ID: id, WorkspaceID: ws.ID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "memory not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to forget memory")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RecallMemory(w http.ResponseWriter, r *http.Request) {
	ws, ok := h.requireMemoryAvailable(w, r)
	if !ok {
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	agentID := strings.TrimSpace(r.URL.Query().Get("agent_id"))
	if agentID != "" {
		agentUUID, parsed := parseUUIDOrBadRequest(w, agentID, "agent_id")
		if !parsed {
			return
		}
		if !h.authorizeMemory(w, r, ws, memory.ScopeAgent, agentUUID, false) {
			return
		}
	}
	hits, err := h.recallMemoryHits(r.Context(), ws, memory.SearchQuery{
		WorkspaceID:  uuidToString(ws.ID),
		Query:        strings.TrimSpace(r.URL.Query().Get("q")),
		InitiativeID: strings.TrimSpace(r.URL.Query().Get("initiative_id")),
		ProjectID:    strings.TrimSpace(r.URL.Query().Get("project_id")),
		IssueID:      strings.TrimSpace(r.URL.Query().Get("issue_id")),
		SquadID:      strings.TrimSpace(r.URL.Query().Get("squad_id")),
		AgentID:      agentID,
		UserID:       userID,
		Limit:        memory.MaxRecallHits,
	})
	if err != nil {
		slog.Warn("RecallMemory failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to recall memory")
		return
	}
	writeJSON(w, http.StatusOK, MemoryRecallResponse{Hits: hits})
}

func (h *Handler) recallMemoryHits(ctx context.Context, ws db.Workspace, q memory.SearchQuery) ([]memory.Hit, error) {
	engine := h.MemoryEngine
	if engine == nil {
		engine = memory.NativeEngine{Queries: h.Queries}
	}
	return engine.Search(ctx, q)
}

func (h *Handler) authorizeMemory(w http.ResponseWriter, r *http.Request, ws db.Workspace, scope string, ownerID pgtype.UUID, write bool) bool {
	userID, ok := requireUserID(w, r)
	if !ok {
		return false
	}
	member, err := h.Queries.GetMemberByUserAndWorkspace(r.Context(), db.GetMemberByUserAndWorkspaceParams{
		UserID: parseUUID(userID), WorkspaceID: ws.ID,
	})
	if err != nil {
		writeError(w, http.StatusForbidden, "not a workspace member")
		return false
	}
	actorType, actorID := h.resolveActor(r, userID, uuidToString(ws.ID))
	switch scope {
	case memory.ScopeWorkspace:
		if write && member.Role != "owner" && member.Role != "admin" {
			writeError(w, http.StatusForbidden, "only owners and admins can write workspace memory")
			return false
		}
		return true
	case memory.ScopeInitiative, memory.ScopeProject, memory.ScopeIssue:
		return true
	case memory.ScopeSquad:
		if !write {
			return true
		}
		if member.Role == "owner" || member.Role == "admin" {
			return true
		}
		squad, err := h.Queries.GetSquadInWorkspace(r.Context(), db.GetSquadInWorkspaceParams{
			ID: ownerID, WorkspaceID: ws.ID,
		})
		if err != nil {
			writeError(w, http.StatusNotFound, "squad not found")
			return false
		}
		if actorType == "agent" && actorID == uuidToString(squad.LeaderID) {
			return true
		}
		leader, err := h.Queries.GetAgentInWorkspace(r.Context(), db.GetAgentInWorkspaceParams{
			ID: squad.LeaderID, WorkspaceID: ws.ID,
		})
		if err == nil && uuidToString(leader.OwnerID) == userID {
			return true
		}
		writeError(w, http.StatusForbidden, "only the squad leader can write squad memory")
		return false
	case memory.ScopeAgent:
		agent, err := h.Queries.GetAgentInWorkspace(r.Context(), db.GetAgentInWorkspaceParams{
			ID: ownerID, WorkspaceID: ws.ID,
		})
		if err != nil {
			writeError(w, http.StatusNotFound, "agent not found")
			return false
		}
		if actorType == "agent" && actorID == uuidToString(agent.ID) {
			return true
		}
		if uuidToString(agent.OwnerID) == userID || member.Role == "owner" || member.Role == "admin" {
			return true
		}
		if h.canInvokeAgent(r.Context(), agent, actorType, actorID, userID, uuidToString(ws.ID)) {
			return true
		}
		writeError(w, http.StatusForbidden, "not allowed to access this agent's memory")
		return false
	case memory.ScopeUser:
		if uuidToString(ownerID) != userID {
			writeError(w, http.StatusForbidden, "user memory is private")
			return false
		}
		return true
	default:
		writeError(w, http.StatusBadRequest, "scope is required")
		return false
	}
}

func (h *Handler) ownerExists(w http.ResponseWriter, r *http.Request, workspaceID pgtype.UUID, scope string, ownerID pgtype.UUID) bool {
	switch scope {
	case memory.ScopeWorkspace:
		if uuidToString(ownerID) != uuidToString(workspaceID) {
			writeError(w, http.StatusBadRequest, "workspace memory owner_id must be the workspace id")
			return false
		}
		return true
	case memory.ScopeInitiative:
		if _, err := h.Queries.GetInitiativeInWorkspace(r.Context(), db.GetInitiativeInWorkspaceParams{
			ID: ownerID, WorkspaceID: workspaceID,
		}); err != nil {
			writeError(w, http.StatusNotFound, "initiative not found")
			return false
		}
	case memory.ScopeProject:
		if _, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{
			ID: ownerID, WorkspaceID: workspaceID,
		}); err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return false
		}
	case memory.ScopeIssue:
		if _, err := h.Queries.GetIssueInWorkspace(r.Context(), db.GetIssueInWorkspaceParams{
			ID: ownerID, WorkspaceID: workspaceID,
		}); err != nil {
			writeError(w, http.StatusNotFound, "issue not found")
			return false
		}
	case memory.ScopeSquad:
		if _, err := h.Queries.GetSquadInWorkspace(r.Context(), db.GetSquadInWorkspaceParams{
			ID: ownerID, WorkspaceID: workspaceID,
		}); err != nil {
			writeError(w, http.StatusNotFound, "squad not found")
			return false
		}
	case memory.ScopeAgent:
		if _, err := h.Queries.GetAgentInWorkspace(r.Context(), db.GetAgentInWorkspaceParams{
			ID: ownerID, WorkspaceID: workspaceID,
		}); err != nil {
			writeError(w, http.StatusNotFound, "agent not found")
			return false
		}
	case memory.ScopeUser:
		if _, err := h.Queries.GetUser(r.Context(), ownerID); err != nil {
			writeError(w, http.StatusNotFound, "user not found")
			return false
		}
	}
	return true
}

func listWorkspaceArg(workspaceID pgtype.UUID, scope string) pgtype.UUID {
	if scope == memory.ScopeUser {
		return pgtype.UUID{}
	}
	return workspaceID
}

func createWorkspaceArg(workspaceID pgtype.UUID, scope string) pgtype.UUID {
	return listWorkspaceArg(workspaceID, scope)
}

func optionalTextArg(s string) pgtype.Text {
	if strings.TrimSpace(s) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func queryInt32(r *http.Request, key string) int32 {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return int32(n)
}

func marshalProvenance(v map[string]any) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}

func deleteOwnerMemory(ctx context.Context, q *db.Queries, scope string, ownerID, workspaceID pgtype.UUID) error {
	return q.DeleteMemoryEntriesByOwner(ctx, db.DeleteMemoryEntriesByOwnerParams{
		Scope:       scope,
		OwnerID:     ownerID,
		WorkspaceID: workspaceID,
	})
}

func (h *Handler) attachClaimMemory(ctx context.Context, resp *AgentTaskResponse, runtime db.AgentRuntime, ws db.Workspace) {
	if !memory.Available(ctx, h.FeatureFlags, ws.Settings) {
		return
	}
	resp.MemoryEnabled = true
	q := memory.SearchQuery{
		WorkspaceID: resp.WorkspaceID,
		AgentID:     resp.AgentID,
		Limit:       memory.MaxRecallHits,
	}
	if runtime.OwnerID.Valid {
		q.UserID = uuidToString(runtime.OwnerID)
	}
	if resp.IssueID != "" {
		q.IssueID = resp.IssueID
	}
	if resp.ProjectID != "" {
		q.ProjectID = resp.ProjectID
	}
	if resp.InitiativeID != "" {
		q.InitiativeID = resp.InitiativeID
	}
	if resp.IsLeaderTask && resp.SquadID != "" {
		q.SquadID = resp.SquadID
	} else if resp.IssueID != "" {
		if issue, err := h.Queries.GetIssue(ctx, parseUUID(resp.IssueID)); err == nil &&
			issue.AssigneeType.Valid && issue.AssigneeType.String == "squad" && issue.AssigneeID.Valid {
			q.SquadID = uuidToString(issue.AssigneeID)
		}
	}
	hits, err := h.recallMemoryHits(ctx, ws, q)
	if err != nil {
		slog.Warn("task claim: memory recall failed",
			"task_id", resp.ID,
			"workspace_id", resp.WorkspaceID,
			"error", err,
		)
		return
	}
	if len(hits) > 0 {
		resp.MemoryHits = hits
	}
}
