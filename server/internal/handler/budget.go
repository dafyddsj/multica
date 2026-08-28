package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/budgetpolicy"
	"github.com/multica-ai/multica/server/internal/logger"
	"github.com/multica-ai/multica/server/internal/service"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

type BudgetPeriodResponse struct {
	PeriodStart       string `json:"period_start"`
	PeriodEnd         string `json:"period_end"`
	SpentUsdTicks     int64  `json:"spent_usd_ticks"`
	UnpricedLineCount int32  `json:"unpriced_line_count"`
	State             string `json:"state"`
}

type BudgetResponse struct {
	ID              string                `json:"id"`
	Scope           string                `json:"scope"`
	OwnerID         string                `json:"owner_id"`
	LimitUsdTicks   int64                 `json:"limit_usd_ticks"`
	SoftenAtPercent *int16                `json:"soften_at_percent"`
	OverLimit       string                `json:"over_limit"`
	CurrentPeriod   *BudgetPeriodResponse `json:"current_period"`
}

type BudgetWaiverResponse struct {
	ID        string  `json:"id"`
	Scope     string  `json:"scope"`
	OwnerID   string  `json:"owner_id"`
	StartsAt  string  `json:"starts_at"`
	EndsAt    string  `json:"ends_at"`
	CreatedBy string  `json:"created_by"`
	Reason    *string `json:"reason"`
}

type CreateBudgetRequest struct {
	Scope           string `json:"scope"`
	OwnerID         string `json:"owner_id"`
	LimitUsdTicks   int64  `json:"limit_usd_ticks"`
	SoftenAtPercent *int16 `json:"soften_at_percent"`
	OverLimit       string `json:"over_limit"`
}

type UpdateBudgetRequest struct {
	LimitUsdTicks   *int64  `json:"limit_usd_ticks"`
	SoftenAtPercent *int16  `json:"soften_at_percent"`
	OverLimit       *string `json:"over_limit"`
}

type CreateBudgetWaiverRequest struct {
	Scope    string  `json:"scope"`
	OwnerID  string  `json:"owner_id"`
	StartsAt *string `json:"starts_at"`
	EndsAt   *string `json:"ends_at"`
	Reason   *string `json:"reason"`
}

func budgetToResponse(v service.BudgetView) BudgetResponse {
	resp := BudgetResponse{
		ID:            uuidToString(v.Budget.ID),
		Scope:         v.Budget.Scope,
		OwnerID:       uuidToString(v.Budget.OwnerID),
		LimitUsdTicks: v.Budget.LimitUsdTicks,
		OverLimit:     v.Budget.OverLimit,
	}
	if v.Budget.SoftenAtPercent.Valid {
		n := v.Budget.SoftenAtPercent.Int16
		resp.SoftenAtPercent = &n
	}
	if v.Period != nil {
		resp.CurrentPeriod = &BudgetPeriodResponse{
			PeriodStart:       timestampToString(v.Period.PeriodStart),
			PeriodEnd:         timestampToString(v.Period.PeriodEnd),
			SpentUsdTicks:     v.Period.SpentUsdTicks,
			UnpricedLineCount: v.Period.UnpricedLineCount,
			State:             string(v.State),
		}
	}
	return resp
}

func waiverToResponse(row db.BudgetWaiver) BudgetWaiverResponse {
	return BudgetWaiverResponse{
		ID:        uuidToString(row.ID),
		Scope:     row.Scope,
		OwnerID:   uuidToString(row.OwnerID),
		StartsAt:  timestampToString(row.StartsAt),
		EndsAt:    timestampToString(row.EndsAt),
		CreatedBy: uuidToString(row.CreatedBy),
		Reason:    textToPtr(row.Reason),
	}
}

func (h *Handler) ListBudgets(w http.ResponseWriter, r *http.Request) {
	_, wsUUID, _, ok := h.requireBudgetMember(w, r)
	if !ok {
		return
	}
	views, err := h.Budgets.List(r.Context(), wsUUID)
	if err != nil {
		slog.Warn("ListBudgets failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to list budgets")
		return
	}
	resp := make([]BudgetResponse, 0, len(views))
	for _, v := range views {
		resp = append(resp, budgetToResponse(v))
	}
	writeJSON(w, http.StatusOK, map[string]any{"budgets": resp})
}

func (h *Handler) CreateBudget(w http.ResponseWriter, r *http.Request) {
	_, wsUUID, member, ok := h.requireBudgetMember(w, r)
	if !ok {
		return
	}
	var req CreateBudgetRequest
	rawFields, err := decodeJSONBodyWithRawFields(r.Body, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Scope = strings.ToLower(strings.TrimSpace(req.Scope))
	req.OverLimit = strings.ToLower(strings.TrimSpace(req.OverLimit))
	ownerID, ok := parseUUIDOrBadRequest(w, req.OwnerID, "owner_id")
	if !ok {
		return
	}
	if !h.authorizeBudgetWrite(w, r, uuidToString(wsUUID), req.Scope, ownerID) {
		return
	}

	write := service.BudgetWrite{
		WorkspaceID: wsUUID,
		Scope:       req.Scope,
		OwnerID:     ownerID,
		LimitTicks:  req.LimitUsdTicks,
		OverLimit:   req.OverLimit,
		CreatedBy:   member.UserID,
		Soften:      softenFromRaw(rawFields, req.SoftenAtPercent),
	}
	view, created, err := h.Budgets.Create(r.Context(), write)
	if err != nil {
		if writeBudgetError(w, err) {
			return
		}
		slog.Warn("CreateBudget failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to create budget")
		return
	}
	action := "updated"
	status := http.StatusOK
	if created {
		action = "created"
		status = http.StatusCreated
	}
	h.publishBudgetUpdated(uuidToString(wsUUID), member, action)
	writeJSON(w, status, budgetToResponse(view))
}

func (h *Handler) PatchBudget(w http.ResponseWriter, r *http.Request) {
	_, wsUUID, member, ok := h.requireBudgetMember(w, r)
	if !ok {
		return
	}
	budgetID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}
	existing, err := h.Budgets.Get(r.Context(), wsUUID, budgetID)
	if err != nil {
		if writeBudgetError(w, err) {
			return
		}
		slog.Warn("PatchBudget load failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to load budget")
		return
	}
	if !h.authorizeBudgetWrite(w, r, uuidToString(wsUUID), existing.Budget.Scope, existing.Budget.OwnerID) {
		return
	}

	var req UpdateBudgetRequest
	rawFields, err := decodeJSONBodyWithRawFields(r.Body, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.OverLimit != nil {
		v := strings.ToLower(strings.TrimSpace(*req.OverLimit))
		req.OverLimit = &v
	}
	view, err := h.Budgets.Update(r.Context(), wsUUID, budgetID, service.BudgetPatch{
		LimitTicks: req.LimitUsdTicks,
		Soften:     softenFromRaw(rawFields, req.SoftenAtPercent),
		OverLimit:  req.OverLimit,
	})
	if err != nil {
		if writeBudgetError(w, err) {
			return
		}
		slog.Warn("PatchBudget failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to update budget")
		return
	}
	h.publishBudgetUpdated(uuidToString(wsUUID), member, "updated")
	writeJSON(w, http.StatusOK, budgetToResponse(view))
}

func (h *Handler) DeleteBudget(w http.ResponseWriter, r *http.Request) {
	_, wsUUID, member, ok := h.requireBudgetMember(w, r)
	if !ok {
		return
	}
	budgetID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}
	existing, err := h.Budgets.Get(r.Context(), wsUUID, budgetID)
	if err != nil {
		if writeBudgetError(w, err) {
			return
		}
		slog.Warn("DeleteBudget load failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to load budget")
		return
	}
	if !h.authorizeBudgetWrite(w, r, uuidToString(wsUUID), existing.Budget.Scope, existing.Budget.OwnerID) {
		return
	}
	if err := h.Budgets.Delete(r.Context(), wsUUID, budgetID); err != nil {
		if writeBudgetError(w, err) {
			return
		}
		slog.Warn("DeleteBudget failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to delete budget")
		return
	}
	h.publishBudgetUpdated(uuidToString(wsUUID), member, "deleted")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListBudgetWaivers(w http.ResponseWriter, r *http.Request) {
	_, wsUUID, _, ok := h.requireBudgetMember(w, r)
	if !ok {
		return
	}
	rows, err := h.Budgets.ListWaivers(r.Context(), wsUUID)
	if err != nil {
		slog.Warn("ListBudgetWaivers failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to list waivers")
		return
	}
	resp := make([]BudgetWaiverResponse, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, waiverToResponse(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"waivers": resp})
}

func (h *Handler) CreateBudgetWaiver(w http.ResponseWriter, r *http.Request) {
	workspaceID, wsUUID, member, ok := h.requireBudgetMember(w, r)
	if !ok {
		return
	}
	if !roleAllowed(member.Role, "owner", "admin") {
		writeError(w, http.StatusForbidden, "only a workspace owner or admin can waive budget teeth")
		return
	}
	var req CreateBudgetWaiverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Scope = strings.ToLower(strings.TrimSpace(req.Scope))
	if req.Scope != string(budgetpolicy.ScopeProject) && req.Scope != string(budgetpolicy.ScopeInitiative) {
		writeError(w, http.StatusBadRequest, "waiver scope must be project or initiative")
		return
	}
	ownerID, ok := parseUUIDOrBadRequest(w, req.OwnerID, "owner_id")
	if !ok {
		return
	}
	if !h.waiverOwnerExists(w, r, wsUUID, req.Scope, ownerID) {
		return
	}

	now := time.Now().UTC()
	startsAt := now
	if req.StartsAt != nil && strings.TrimSpace(*req.StartsAt) != "" {
		parsed, err := parseRFC3339(*req.StartsAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid starts_at")
			return
		}
		startsAt = parsed
	}
	_, monthEnd := budgetpolicy.MonthWindow(now)
	endsAt := monthEnd
	if req.EndsAt != nil && strings.TrimSpace(*req.EndsAt) != "" {
		parsed, err := parseRFC3339(*req.EndsAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid ends_at")
			return
		}
		endsAt = parsed
	}

	row, err := h.Budgets.CreateWaiver(r.Context(), service.WaiverWrite{
		WorkspaceID: wsUUID,
		Scope:       req.Scope,
		OwnerID:     ownerID,
		StartsAt:    startsAt,
		EndsAt:      endsAt,
		CreatedBy:   member.UserID,
		Reason:      req.Reason,
	})
	if err != nil {
		if writeBudgetError(w, err) {
			return
		}
		slog.Warn("CreateBudgetWaiver failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to create waiver")
		return
	}
	h.publishBudgetUpdated(workspaceID, member, "waiver_created")
	writeJSON(w, http.StatusCreated, waiverToResponse(row))
}

func (h *Handler) DeleteBudgetWaiver(w http.ResponseWriter, r *http.Request) {
	workspaceID, wsUUID, member, ok := h.requireBudgetMember(w, r)
	if !ok {
		return
	}
	if !roleAllowed(member.Role, "owner", "admin") {
		writeError(w, http.StatusForbidden, "only a workspace owner or admin can waive budget teeth")
		return
	}
	waiverID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}
	if err := h.Budgets.DeleteWaiver(r.Context(), wsUUID, waiverID); err != nil {
		if writeBudgetError(w, err) {
			return
		}
		slog.Warn("DeleteBudgetWaiver failed", append(logger.RequestAttrs(r), "error", err)...)
		writeError(w, http.StatusInternalServerError, "failed to delete waiver")
		return
	}
	h.publishBudgetUpdated(workspaceID, member, "waiver_deleted")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) requireBudgetMember(w http.ResponseWriter, r *http.Request) (string, pgtype.UUID, db.Member, bool) {
	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return "", pgtype.UUID{}, db.Member{}, false
	}
	member, ok := h.requireWorkspaceMember(w, r, workspaceID, "workspace not found")
	if !ok {
		return "", pgtype.UUID{}, db.Member{}, false
	}
	if h.Budgets == nil {
		writeError(w, http.StatusInternalServerError, "budgets are not configured")
		return "", pgtype.UUID{}, db.Member{}, false
	}
	return workspaceID, wsUUID, member, true
}

func (h *Handler) authorizeBudgetWrite(w http.ResponseWriter, r *http.Request, workspaceID, scope string, ownerID pgtype.UUID) bool {
	wsUUID, err := parseWorkspaceUUID(workspaceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workspace id")
		return false
	}
	switch scope {
	case string(budgetpolicy.ScopeAgent):
		agent, err := h.Queries.GetAgentInWorkspace(r.Context(), db.GetAgentInWorkspaceParams{
			ID:          ownerID,
			WorkspaceID: wsUUID,
		})
		if err != nil {
			writeError(w, http.StatusNotFound, "agent not found")
			return false
		}
		return h.canManageAgent(w, r, agent)
	case string(budgetpolicy.ScopeSquad):
		squad, err := h.Queries.GetSquadInWorkspace(r.Context(), db.GetSquadInWorkspaceParams{
			ID:          ownerID,
			WorkspaceID: wsUUID,
		})
		if err != nil {
			writeError(w, http.StatusNotFound, "squad not found")
			return false
		}
		member, ok := h.requireWorkspaceMember(w, r, workspaceID, "workspace not found")
		if !ok {
			return false
		}
		if roleAllowed(member.Role, "owner", "admin") {
			return true
		}
		if uuidToString(squad.CreatorID) == requestUserID(r) {
			return true
		}
		writeError(w, http.StatusForbidden, "only the squad creator or a workspace admin can manage this budget")
		return false
	case string(budgetpolicy.ScopeProject):
		if _, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{
			ID:          ownerID,
			WorkspaceID: wsUUID,
		}); err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return false
		}
		return true
	case string(budgetpolicy.ScopeInitiative):
		if _, err := h.Queries.GetInitiativeInWorkspace(r.Context(), db.GetInitiativeInWorkspaceParams{
			ID:          ownerID,
			WorkspaceID: wsUUID,
		}); err != nil {
			writeError(w, http.StatusNotFound, "initiative not found")
			return false
		}
		return true
	default:
		writeError(w, http.StatusBadRequest, "scope must be agent, squad, project, or initiative")
		return false
	}
}

func (h *Handler) waiverOwnerExists(w http.ResponseWriter, r *http.Request, wsUUID pgtype.UUID, scope string, ownerID pgtype.UUID) bool {
	switch scope {
	case string(budgetpolicy.ScopeProject):
		if _, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{
			ID:          ownerID,
			WorkspaceID: wsUUID,
		}); err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return false
		}
	case string(budgetpolicy.ScopeInitiative):
		if _, err := h.Queries.GetInitiativeInWorkspace(r.Context(), db.GetInitiativeInWorkspaceParams{
			ID:          ownerID,
			WorkspaceID: wsUUID,
		}); err != nil {
			writeError(w, http.StatusNotFound, "initiative not found")
			return false
		}
	default:
		writeError(w, http.StatusBadRequest, "waiver scope must be project or initiative")
		return false
	}
	return true
}

func (h *Handler) publishBudgetUpdated(workspaceID string, actor db.Member, action string) {
	h.publish(protocol.EventBudgetUpdated, workspaceID, "member", uuidToString(actor.UserID), map[string]any{
		"action": action,
	})
}

func softenFromRaw(rawFields map[string]json.RawMessage, decoded *int16) service.SoftenPatch {
	raw, ok := rawFields["soften_at_percent"]
	if !ok {
		return service.SoftenPatch{}
	}
	if string(raw) == "null" {
		return service.SoftenPatch{Set: true}
	}
	return service.SoftenPatch{Set: true, Value: decoded}
}

func writeBudgetError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, service.ErrBudgetNotFound):
		writeError(w, http.StatusNotFound, "budget not found")
	case errors.Is(err, service.ErrWaiverNotFound):
		writeError(w, http.StatusNotFound, "waiver not found")
	case errors.Is(err, service.ErrInvalidScope):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInvalidOverLimit):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInvalidLimit):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInvalidSoften):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrWaiverScope):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrWaiverWindow):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrWaiverOverlap):
		writeError(w, http.StatusConflict, err.Error())
	default:
		return false
	}
	return true
}

func parseRFC3339(raw string) (time.Time, error) {
	s := strings.TrimSpace(raw)
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

func parseWorkspaceUUID(workspaceID string) (pgtype.UUID, error) {
	return util.ParseUUID(workspaceID)
}
