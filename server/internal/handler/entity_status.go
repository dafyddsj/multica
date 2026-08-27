package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/entitystatus"
	"github.com/multica-ai/multica/server/internal/logger"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// Initiative and Project status catalog API.
//
// Reading is open to any workspace member. Mutating is owner/admin only.

type EntityStatusResponse struct {
	ID           string  `json:"id"`
	WorkspaceID  string  `json:"workspace_id"`
	ResourceType string  `json:"resource_type"`
	Key          string  `json:"key"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Category     string  `json:"category"`
	Color        string  `json:"color"`
	IsSystem     bool    `json:"is_system"`
	Position     float64 `json:"position"`
	ArchivedAt   *string `json:"archived_at"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func entityStatusToResponse(s db.EntityStatus) EntityStatusResponse {
	return EntityStatusResponse{
		ID:           uuidToString(s.ID),
		WorkspaceID:  uuidToString(s.WorkspaceID),
		ResourceType: s.ResourceType,
		Key:          s.Key,
		Name:         s.Name,
		Description:  s.Description,
		Category:     s.Category,
		Color:        s.Color,
		IsSystem:     s.IsSystem,
		Position:     s.Position,
		ArchivedAt:   timestampToPtr(s.ArchivedAt),
		CreatedAt:    timestampToString(s.CreatedAt),
		UpdatedAt:    timestampToString(s.UpdatedAt),
	}
}

type CreateEntityStatusRequest struct {
	ResourceType string `json:"resource_type"`
	Key          string `json:"key"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	Color        string `json:"color"`
}

type UpdateEntityStatusRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Color       *string  `json:"color"`
	Position    *float64 `json:"position"`
}

func parseEntityResourceType(w http.ResponseWriter, raw string) (string, bool) {
	resourceType := strings.ToLower(strings.TrimSpace(raw))
	if !entitystatus.IsResourceType(resourceType) {
		writeError(w, http.StatusBadRequest, "resource_type must be initiative or project")
		return "", false
	}
	return resourceType, true
}

func (h *Handler) ListEntityStatuses(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return
	}
	if _, ok := h.requireWorkspaceMember(w, r, workspaceID, "workspace not found"); !ok {
		return
	}
	resourceType, ok := parseEntityResourceType(w, r.URL.Query().Get("resource_type"))
	if !ok {
		return
	}

	if err := entitystatus.Ensure(r.Context(), h.Queries, wsUUID); err != nil {
		slog.Warn("failed to ensure entity status catalog", append(logger.RequestAttrs(r), "error", err)...)
	}

	includeArchived := strings.EqualFold(r.URL.Query().Get("include_archived"), "true")
	entries, err := h.Queries.ListEntityStatusEntries(r.Context(), db.ListEntityStatusEntriesParams{
		WorkspaceID:     wsUUID,
		ResourceType:    resourceType,
		IncludeArchived: includeArchived,
	})
	if err != nil {
		slog.Warn("ListEntityStatuses failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to list statuses")
		return
	}

	resp := make([]EntityStatusResponse, len(entries))
	for i, e := range entries {
		resp[i] = entityStatusToResponse(e)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"statuses":      resp,
		"resource_type": resourceType,
		"categories":    entitystatus.Canonical(),
		"total":         len(resp),
	})
}

func (h *Handler) CreateEntityStatus(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return
	}
	member, ok := h.requireWorkspaceRole(w, r, workspaceID, "workspace not found", "owner", "admin")
	if !ok {
		return
	}

	var req CreateEntityStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resourceType, ok := parseEntityResourceType(w, req.ResourceType)
	if !ok {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 64 {
		writeError(w, http.StatusBadRequest, "name must be 1-64 characters")
		return
	}
	if len([]rune(req.Description)) > 256 {
		writeError(w, http.StatusBadRequest, "description must be at most 256 characters")
		return
	}
	if !entitystatus.IsCategory(req.Category) {
		writeError(w, http.StatusBadRequest, "category must be one of: "+strings.Join(entitystatus.Canonical(), ", "))
		return
	}
	color, err := normalizeColor(req.Color)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var key string
	if strings.TrimSpace(req.Key) != "" {
		key, err = entitystatus.ValidateKey(req.Key)
	} else {
		key, err = entitystatus.SlugifyKey(name)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := entitystatus.Ensure(r.Context(), h.Queries, wsUUID); err != nil {
		slog.Warn("failed to ensure entity status catalog", append(logger.RequestAttrs(r), "error", err)...)
	}

	entry, err := h.Queries.CreateEntityStatusEntry(r.Context(), db.CreateEntityStatusEntryParams{
		WorkspaceID:  wsUUID,
		ResourceType: resourceType,
		Key:          key,
		Name:         name,
		Description:  req.Description,
		Category:     req.Category,
		Color:        strings.ToLower(color),
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "a status with this key or name already exists")
			return
		}
		slog.Warn("CreateEntityStatus failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to create status")
		return
	}
	h.publishEntityStatusChanged(workspaceID, member, resourceType, "created")
	writeJSON(w, http.StatusCreated, entityStatusToResponse(entry))
}

func (h *Handler) UpdateEntityStatus(w http.ResponseWriter, r *http.Request) {
	entry, wsUUID, member, ok := h.loadEntityStatusForAdmin(w, r)
	if !ok {
		return
	}

	var req UpdateEntityStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if entry.ArchivedAt.Valid {
		writeError(w, http.StatusConflict, "archived statuses cannot be modified")
		return
	}

	var name pgtype.Text
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" || len([]rune(trimmed)) > 64 {
			writeError(w, http.StatusBadRequest, "name must be 1-64 characters")
			return
		}
		name = pgtype.Text{String: trimmed, Valid: true}
	}
	var description pgtype.Text
	if req.Description != nil {
		if len([]rune(*req.Description)) > 256 {
			writeError(w, http.StatusBadRequest, "description must be at most 256 characters")
			return
		}
		description = pgtype.Text{String: *req.Description, Valid: true}
	}
	var color pgtype.Text
	if req.Color != nil {
		normalized, err := normalizeColor(*req.Color)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		color = pgtype.Text{String: strings.ToLower(normalized), Valid: true}
	}
	var position pgtype.Float8
	if req.Position != nil && !entry.IsSystem {
		position = pgtype.Float8{Float64: *req.Position, Valid: true}
	}

	updated, err := h.Queries.UpdateEntityStatusEntry(r.Context(), db.UpdateEntityStatusEntryParams{
		ID:          entry.ID,
		WorkspaceID: wsUUID,
		Name:        name,
		Description: description,
		Color:       color,
		Position:    position,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusConflict, "status is no longer editable")
			return
		}
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "a status with this name already exists")
			return
		}
		slog.Warn("UpdateEntityStatus failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to update status")
		return
	}
	h.publishEntityStatusChanged(uuidToString(wsUUID), member, entry.ResourceType, "updated")
	writeJSON(w, http.StatusOK, entityStatusToResponse(updated))
}

func (h *Handler) ArchiveEntityStatus(w http.ResponseWriter, r *http.Request) {
	entry, wsUUID, member, ok := h.loadEntityStatusForAdmin(w, r)
	if !ok {
		return
	}
	if entry.IsSystem {
		writeError(w, http.StatusForbidden, "built-in statuses cannot be archived")
		return
	}
	if entry.ArchivedAt.Valid {
		writeJSON(w, http.StatusOK, entityStatusToResponse(entry))
		return
	}

	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		slog.Warn("ArchiveEntityStatus begin failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to archive status")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)

	if err := qtx.LockEntityStatusCatalog(r.Context(), db.LockEntityStatusCatalogParams{
		WorkspaceID:  wsUUID,
		ResourceType: entry.ResourceType,
	}); err != nil {
		slog.Warn("ArchiveEntityStatus lock failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to archive status")
		return
	}

	archived, err := qtx.ArchiveEntityStatusEntry(r.Context(), db.ArchiveEntityStatusEntryParams{
		ID:          entry.ID,
		WorkspaceID: wsUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusConflict, "status is no longer archivable")
			return
		}
		slog.Warn("ArchiveEntityStatus failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to archive status")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Warn("ArchiveEntityStatus commit failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to archive status")
		return
	}
	h.publishEntityStatusChanged(uuidToString(wsUUID), member, entry.ResourceType, "archived")
	writeJSON(w, http.StatusOK, entityStatusToResponse(archived))
}

func (h *Handler) loadEntityStatusForAdmin(w http.ResponseWriter, r *http.Request) (db.EntityStatus, pgtype.UUID, db.Member, bool) {
	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return db.EntityStatus{}, pgtype.UUID{}, db.Member{}, false
	}
	member, ok := h.requireWorkspaceRole(w, r, workspaceID, "workspace not found", "owner", "admin")
	if !ok {
		return db.EntityStatus{}, pgtype.UUID{}, db.Member{}, false
	}
	idUUID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "status id")
	if !ok {
		return db.EntityStatus{}, pgtype.UUID{}, db.Member{}, false
	}
	entry, err := h.Queries.GetEntityStatusEntryByID(r.Context(), db.GetEntityStatusEntryByIDParams{
		ID:          idUUID,
		WorkspaceID: wsUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "status not found")
			return db.EntityStatus{}, pgtype.UUID{}, db.Member{}, false
		}
		slog.Warn("load entity status failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to load status")
		return db.EntityStatus{}, pgtype.UUID{}, db.Member{}, false
	}
	return entry, wsUUID, member, true
}

func (h *Handler) publishEntityStatusChanged(workspaceID string, actor db.Member, resourceType, action string) {
	h.publish(protocol.EventEntityStatusChanged, workspaceID, "member", uuidToString(actor.UserID), map[string]any{
		"action":        action,
		"resource_type": resourceType,
	})
}

type ReorderEntityStatusesRequest struct {
	ResourceType string   `json:"resource_type"`
	Category     string   `json:"category"`
	IDs          []string `json:"ids"`
}

func (h *Handler) ReorderEntityStatuses(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return
	}
	member, ok := h.requireWorkspaceRole(w, r, workspaceID, "workspace not found", "owner", "admin")
	if !ok {
		return
	}

	var req ReorderEntityStatusesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resourceType, ok := parseEntityResourceType(w, req.ResourceType)
	if !ok {
		return
	}
	if !entitystatus.IsCategory(req.Category) {
		writeError(w, http.StatusBadRequest, "category must be one of: "+strings.Join(entitystatus.Canonical(), ", "))
		return
	}
	if len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "ids must not be empty")
		return
	}
	ids := make([]pgtype.UUID, 0, len(req.IDs))
	seen := make(map[string]struct{}, len(req.IDs))
	for _, raw := range req.IDs {
		if _, duplicate := seen[raw]; duplicate {
			writeError(w, http.StatusBadRequest, "duplicate ids")
			return
		}
		seen[raw] = struct{}{}
		idUUID, err := util.ParseUUID(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid status id")
			return
		}
		ids = append(ids, idUUID)
	}

	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		slog.Warn("ReorderEntityStatuses begin failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to reorder statuses")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)

	if err := qtx.LockEntityStatusCatalogShared(r.Context(), db.LockEntityStatusCatalogSharedParams{
		WorkspaceID:  wsUUID,
		ResourceType: resourceType,
	}); err != nil {
		slog.Warn("ReorderEntityStatuses lock failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to reorder statuses")
		return
	}

	active, err := qtx.ListActiveCustomEntityStatusEntries(r.Context(), db.ListActiveCustomEntityStatusEntriesParams{
		WorkspaceID:  wsUUID,
		ResourceType: resourceType,
		Category:     req.Category,
	})
	if err != nil {
		slog.Warn("ReorderEntityStatuses list failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to reorder statuses")
		return
	}
	activeIDs := make(map[string]struct{}, len(active))
	for _, entry := range active {
		activeIDs[util.UUIDToString(entry.ID)] = struct{}{}
	}
	for i, raw := range req.IDs {
		if _, isActive := activeIDs[raw]; isActive {
			continue
		}
		entry, err := qtx.GetEntityStatusEntryByID(r.Context(), db.GetEntityStatusEntryByIDParams{
			ID:          ids[i],
			WorkspaceID: wsUUID,
		})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			writeError(w, http.StatusNotFound, "status not found")
		case err != nil:
			slog.Warn("load status for reorder failed", append(logger.RequestAttrs(r), "error", err)...)
			writeError(w, http.StatusInternalServerError, "failed to reorder statuses")
		case entry.IsSystem:
			writeError(w, http.StatusForbidden, "built-in statuses cannot be reordered")
		case entry.ArchivedAt.Valid:
			writeError(w, http.StatusConflict, "archived statuses cannot be reordered")
		case entry.Category != req.Category || entry.ResourceType != resourceType:
			writeError(w, http.StatusBadRequest, "ids must all belong to the requested category")
		default:
			writeError(w, http.StatusConflict, "status catalog changed during reorder")
		}
		return
	}
	if len(active) != len(ids) {
		writeError(w, http.StatusConflict, "ids must name every active custom status in the category")
		return
	}

	affected, err := qtx.ReorderEntityStatusEntries(r.Context(), db.ReorderEntityStatusEntriesParams{
		Ids:          ids,
		WorkspaceID:  wsUUID,
		ResourceType: resourceType,
	})
	if err != nil {
		slog.Warn("ReorderEntityStatuses failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to reorder statuses")
		return
	}
	if affected != int64(len(ids)) {
		writeError(w, http.StatusConflict, "status catalog changed during reorder")
		return
	}

	entries, err := qtx.ListEntityStatusEntries(r.Context(), db.ListEntityStatusEntriesParams{
		WorkspaceID:     wsUUID,
		ResourceType:    resourceType,
		IncludeArchived: true,
	})
	if err != nil {
		slog.Warn("list statuses after reorder failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to list statuses")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Warn("ReorderEntityStatuses commit failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to reorder statuses")
		return
	}
	h.publishEntityStatusChanged(workspaceID, member, resourceType, "reordered")

	resp := make([]EntityStatusResponse, len(entries))
	for i, e := range entries {
		resp[i] = entityStatusToResponse(e)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"statuses":      resp,
		"resource_type": resourceType,
		"categories":    entitystatus.Canonical(),
		"total":         len(resp),
	})
}

func (h *Handler) resolveWritableEntityStatus(w http.ResponseWriter, r *http.Request, workspaceID, resourceType, status string) (string, bool) {
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return "", false
	}
	if err := entitystatus.Ensure(r.Context(), h.Queries, wsUUID); err != nil {
		slog.Warn("failed to ensure entity status catalog", append(logger.RequestAttrs(r), "error", err)...)
	}
	entry, err := entitystatus.Resolve(r.Context(), h.Queries, wsUUID, resourceType, status)
	if err != nil {
		keys, keyErr := entitystatus.ActiveKeys(r.Context(), h.Queries, wsUUID, resourceType)
		if keyErr != nil || len(keys) == 0 {
			keys = entitystatus.Canonical()
		}
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid status %q; valid values: %s", status, strings.Join(keys, ", ")))
		return "", false
	}
	return entry.Key, true
}
