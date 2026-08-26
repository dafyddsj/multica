package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/logger"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

type InitiativeResponse struct {
	ID           string  `json:"id"`
	WorkspaceID  string  `json:"workspace_id"`
	Title        string  `json:"title"`
	Description  *string `json:"description"`
	Icon         *string `json:"icon"`
	Status       string  `json:"status"`
	Priority     string  `json:"priority"`
	LeadType     *string `json:"lead_type"`
	LeadID       *string `json:"lead_id"`
	StartDate    *string `json:"start_date"`
	DueDate      *string `json:"due_date"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	ProjectCount int64   `json:"project_count"`
	IssueCount   int64   `json:"issue_count"`
	DoneCount    int64   `json:"done_count"`
}

func initiativeToResponse(i db.Initiative) InitiativeResponse {
	return InitiativeResponse{
		ID:          uuidToString(i.ID),
		WorkspaceID: uuidToString(i.WorkspaceID),
		Title:       i.Title,
		Description: textToPtr(i.Description),
		Icon:        textToPtr(i.Icon),
		Status:      i.Status,
		Priority:    i.Priority,
		LeadType:    textToPtr(i.LeadType),
		LeadID:      uuidToPtr(i.LeadID),
		StartDate:   dateToPtr(i.StartDate),
		DueDate:     dateToPtr(i.DueDate),
		CreatedAt:   timestampToString(i.CreatedAt),
		UpdatedAt:   timestampToString(i.UpdatedAt),
	}
}

func (h *Handler) loadInitiativeStats(ctx context.Context, initiativeID pgtype.UUID) (projectCount, issueCount, doneCount int64) {
	ids := []pgtype.UUID{initiativeID}
	counts, err := h.Queries.GetInitiativeProjectCounts(ctx, ids)
	if err == nil {
		for _, c := range counts {
			if uuidToString(c.InitiativeID) == uuidToString(initiativeID) {
				projectCount = c.ProjectCount
			}
		}
	}
	stats, err := h.Queries.GetInitiativeIssueStats(ctx, ids)
	if err == nil {
		for _, s := range stats {
			if uuidToString(s.InitiativeID) == uuidToString(initiativeID) {
				issueCount = s.TotalCount
				doneCount = s.DoneCount
			}
		}
	}
	return projectCount, issueCount, doneCount
}

type CreateInitiativeRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	LeadType    *string `json:"lead_type"`
	LeadID      *string `json:"lead_id"`
	StartDate   *string `json:"start_date"`
	DueDate     *string `json:"due_date"`
}

type UpdateInitiativeRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	LeadType    *string `json:"lead_type"`
	LeadID      *string `json:"lead_id"`
	StartDate   *string `json:"start_date"`
	DueDate     *string `json:"due_date"`
}

func (h *Handler) resolveInitiativeInWorkspace(w http.ResponseWriter, r *http.Request, workspaceID pgtype.UUID, raw *string) (pgtype.UUID, bool) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return pgtype.UUID{}, true
	}
	id, ok := parseUUIDOrBadRequest(w, strings.TrimSpace(*raw), "initiative_id")
	if !ok {
		return pgtype.UUID{}, false
	}
	if _, err := h.Queries.GetInitiativeInWorkspace(r.Context(), db.GetInitiativeInWorkspaceParams{
		ID: id, WorkspaceID: workspaceID,
	}); err != nil {
		writeError(w, http.StatusBadRequest, "initiative not found in workspace")
		return pgtype.UUID{}, false
	}
	return id, true
}

func (h *Handler) writeInitiativeWriteError(w http.ResponseWriter, r *http.Request, err error, action string) {
	if isCheckViolation(err) {
		writeError(w, http.StatusBadRequest, "initiative "+action+" rejected: a field value failed a database constraint")
		return
	}
	slog.Error("initiative "+action+" failed", append(logger.RequestAttrs(r), "error", err)...)
	writeError(w, http.StatusInternalServerError, "failed to "+action+" initiative")
}

func (h *Handler) fillInitiativeStats(ctx context.Context, resp *InitiativeResponse, id pgtype.UUID) {
	resp.ProjectCount, resp.IssueCount, resp.DoneCount = h.loadInitiativeStats(ctx, id)
}

func (h *Handler) ListInitiatives(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}
	var statusFilter pgtype.Text
	if s := r.URL.Query().Get("status"); s != "" {
		statusFilter = pgtype.Text{String: s, Valid: true}
	}
	var priorityFilter pgtype.Text
	if p := r.URL.Query().Get("priority"); p != "" {
		priorityFilter = pgtype.Text{String: p, Valid: true}
	}
	initiatives, err := h.Queries.ListInitiatives(r.Context(), db.ListInitiativesParams{
		WorkspaceID: wsUUID,
		Status:      statusFilter,
		Priority:    priorityFilter,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list initiatives")
		return
	}

	projectCountMap := map[string]int64{}
	statsMap := map[string]db.GetInitiativeIssueStatsRow{}
	if len(initiatives) > 0 {
		ids := make([]pgtype.UUID, len(initiatives))
		for i, row := range initiatives {
			ids[i] = row.ID
		}
		counts, err := h.Queries.GetInitiativeProjectCounts(r.Context(), ids)
		if err == nil {
			for _, c := range counts {
				projectCountMap[uuidToString(c.InitiativeID)] = c.ProjectCount
			}
		}
		stats, err := h.Queries.GetInitiativeIssueStats(r.Context(), ids)
		if err == nil {
			for _, s := range stats {
				statsMap[uuidToString(s.InitiativeID)] = s
			}
		}
	}

	resp := make([]InitiativeResponse, len(initiatives))
	for i, row := range initiatives {
		resp[i] = initiativeToResponse(row)
		resp[i].ProjectCount = projectCountMap[resp[i].ID]
		if s, ok := statsMap[resp[i].ID]; ok {
			resp[i].IssueCount = s.TotalCount
			resp[i].DoneCount = s.DoneCount
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"initiatives": resp, "total": len(resp)})
}

func (h *Handler) GetInitiative(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	idUUID, ok := parseUUIDOrBadRequest(w, id, "initiative id")
	if !ok {
		return
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return
	}
	initiative, err := h.Queries.GetInitiativeInWorkspace(r.Context(), db.GetInitiativeInWorkspaceParams{
		ID: idUUID, WorkspaceID: wsUUID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "initiative not found")
		return
	}
	resp := initiativeToResponse(initiative)
	h.fillInitiativeStats(r.Context(), &resp, initiative.ID)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateInitiative(w http.ResponseWriter, r *http.Request) {
	var req CreateInitiativeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	workspaceID := h.resolveWorkspaceID(r)
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	status := req.Status
	if status == "" {
		status = "planned"
	}
	if !validateProjectEnum(w, "status", status, validProjectStatuses) {
		return
	}
	priority := req.Priority
	if priority == "" {
		priority = "none"
	}
	if !validateProjectEnum(w, "priority", priority, validProjectPriorities) {
		return
	}
	var leadType pgtype.Text
	var leadID pgtype.UUID
	if req.LeadType != nil {
		leadType = pgtype.Text{String: *req.LeadType, Valid: true}
	}
	if req.LeadID != nil {
		parsed, ok := parseUUIDOrBadRequest(w, *req.LeadID, "lead_id")
		if !ok {
			return
		}
		leadID = parsed
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}
	var startDate pgtype.Date
	if req.StartDate != nil && *req.StartDate != "" {
		d, err := util.ParseCalendarDate(*req.StartDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid start_date format, expected YYYY-MM-DD")
			return
		}
		startDate = d
	}
	var dueDate pgtype.Date
	if req.DueDate != nil && *req.DueDate != "" {
		d, err := util.ParseCalendarDate(*req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid due_date format, expected YYYY-MM-DD")
			return
		}
		dueDate = d
	}

	initiative, err := h.Queries.CreateInitiative(r.Context(), db.CreateInitiativeParams{
		WorkspaceID: wsUUID,
		Title:       req.Title,
		Description: ptrToText(req.Description),
		Icon:        ptrToText(req.Icon),
		Status:      status,
		LeadType:    leadType,
		LeadID:      leadID,
		Priority:    priority,
		StartDate:   startDate,
		DueDate:     dueDate,
	})
	if err != nil {
		h.writeInitiativeWriteError(w, r, err, "create")
		return
	}
	resp := initiativeToResponse(initiative)
	h.publish(protocol.EventInitiativeCreated, workspaceID, "member", userID, map[string]any{"initiative": resp})
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) UpdateInitiative(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	idUUID, ok := parseUUIDOrBadRequest(w, id, "initiative id")
	if !ok {
		return
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return
	}
	prev, err := h.Queries.GetInitiativeInWorkspace(r.Context(), db.GetInitiativeInWorkspaceParams{
		ID: idUUID, WorkspaceID: wsUUID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "initiative not found")
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	var req UpdateInitiativeRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var rawFields map[string]json.RawMessage
	json.Unmarshal(bodyBytes, &rawFields)

	params := db.UpdateInitiativeParams{
		ID:          prev.ID,
		Description: prev.Description,
		Icon:        prev.Icon,
		LeadType:    prev.LeadType,
		LeadID:      prev.LeadID,
		StartDate:   prev.StartDate,
		DueDate:     prev.DueDate,
	}
	if req.Title != nil {
		params.Title = pgtype.Text{String: *req.Title, Valid: true}
	}
	if req.Status != nil {
		if !validateProjectEnum(w, "status", *req.Status, validProjectStatuses) {
			return
		}
		params.Status = pgtype.Text{String: *req.Status, Valid: true}
	}
	if req.Priority != nil {
		if !validateProjectEnum(w, "priority", *req.Priority, validProjectPriorities) {
			return
		}
		params.Priority = pgtype.Text{String: *req.Priority, Valid: true}
	}
	if _, ok := rawFields["description"]; ok {
		if req.Description != nil {
			params.Description = pgtype.Text{String: *req.Description, Valid: true}
		} else {
			params.Description = pgtype.Text{Valid: false}
		}
	}
	if _, ok := rawFields["icon"]; ok {
		if req.Icon != nil {
			params.Icon = pgtype.Text{String: *req.Icon, Valid: true}
		} else {
			params.Icon = pgtype.Text{Valid: false}
		}
	}
	if _, ok := rawFields["lead_type"]; ok {
		if req.LeadType != nil {
			params.LeadType = pgtype.Text{String: *req.LeadType, Valid: true}
		} else {
			params.LeadType = pgtype.Text{Valid: false}
		}
	}
	if _, ok := rawFields["lead_id"]; ok {
		if req.LeadID != nil {
			leadUUID, ok := parseUUIDOrBadRequest(w, *req.LeadID, "lead_id")
			if !ok {
				return
			}
			params.LeadID = leadUUID
		} else {
			params.LeadID = pgtype.UUID{Valid: false}
		}
	}
	if _, ok := rawFields["start_date"]; ok {
		if req.StartDate != nil && *req.StartDate != "" {
			d, err := util.ParseCalendarDate(*req.StartDate)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid start_date format, expected YYYY-MM-DD")
				return
			}
			params.StartDate = d
		} else {
			params.StartDate = pgtype.Date{Valid: false}
		}
	}
	if _, ok := rawFields["due_date"]; ok {
		if req.DueDate != nil && *req.DueDate != "" {
			d, err := util.ParseCalendarDate(*req.DueDate)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid due_date format, expected YYYY-MM-DD")
				return
			}
			params.DueDate = d
		} else {
			params.DueDate = pgtype.Date{Valid: false}
		}
	}
	initiative, err := h.Queries.UpdateInitiative(r.Context(), params)
	if err != nil {
		h.writeInitiativeWriteError(w, r, err, "update")
		return
	}
	resp := initiativeToResponse(initiative)
	h.fillInitiativeStats(r.Context(), &resp, initiative.ID)
	h.publish(protocol.EventInitiativeUpdated, workspaceID, "member", userID, map[string]any{"initiative": resp})
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) DeleteInitiative(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	idUUID, ok := parseUUIDOrBadRequest(w, id, "initiative id")
	if !ok {
		return
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return
	}
	initiative, err := h.Queries.GetInitiativeInWorkspace(r.Context(), db.GetInitiativeInWorkspaceParams{
		ID: idUUID, WorkspaceID: wsUUID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "initiative not found")
		return
	}
	requester, ok := h.requireWorkspaceRole(w, r, uuidToString(initiative.WorkspaceID), "initiative not found", "owner", "admin")
	if !ok {
		return
	}
	userID := uuidToString(requester.UserID)
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)

	if _, err := qtx.LockInitiativeForDelete(r.Context(), db.LockInitiativeForDeleteParams{
		ID:          initiative.ID,
		WorkspaceID: initiative.WorkspaceID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "initiative not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to lock initiative")
		return
	}
	if err := qtx.ClearProjectInitiativeByInitiative(r.Context(), db.ClearProjectInitiativeByInitiativeParams{
		InitiativeID: initiative.ID,
		WorkspaceID:  initiative.WorkspaceID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to detach projects")
		return
	}
	if err := qtx.DeletePinnedItemsByItem(r.Context(), db.DeletePinnedItemsByItemParams{
		ItemType: "initiative",
		ItemID:   initiative.ID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete initiative pins")
		return
	}
	if err := qtx.DeleteInitiative(r.Context(), db.DeleteInitiativeParams{
		ID:          initiative.ID,
		WorkspaceID: initiative.WorkspaceID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete initiative")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit initiative delete")
		return
	}
	h.publish(protocol.EventInitiativeDeleted, workspaceID, "member", userID, map[string]any{"initiative_id": uuidToString(initiative.ID)})
	w.WriteHeader(http.StatusNoContent)
}

type SearchInitiativeResponse struct {
	InitiativeResponse
	MatchSource    string  `json:"match_source"`
	MatchedSnippet *string `json:"matched_snippet,omitempty"`
}

func buildInitiativeSearchQuery(phrase string, terms []string, includeClosed bool) (string, []any) {
	phrase = strings.ToLower(phrase)
	for i, t := range terms {
		terms[i] = strings.ToLower(t)
	}

	argIdx := 1
	args := []any{}
	nextArg := func(val any) string {
		args = append(args, val)
		s := fmt.Sprintf("$%d", argIdx)
		argIdx++
		return s
	}

	escapedPhrase := escapeLike(phrase)
	phraseParam := nextArg(escapedPhrase)
	phraseContains := "'%' || " + phraseParam + " || '%'"
	phraseStartsWith := phraseParam + " || '%'"
	wsParam := nextArg(nil)

	var termParams []string
	if len(terms) > 1 {
		for _, t := range terms {
			et := escapeLike(t)
			termParams = append(termParams, nextArg(et))
		}
	}

	var whereParts []string
	phraseMatch := fmt.Sprintf(
		"(LOWER(i.title) LIKE %s OR LOWER(COALESCE(i.description, '')) LIKE %s)",
		phraseContains, phraseContains,
	)
	whereParts = append(whereParts, phraseMatch)
	if len(termParams) > 1 {
		var termConditions []string
		for _, tp := range termParams {
			tc := "'%' || " + tp + " || '%'"
			termConditions = append(termConditions, fmt.Sprintf(
				"(LOWER(i.title) LIKE %s OR LOWER(COALESCE(i.description, '')) LIKE %s)",
				tc, tc,
			))
		}
		whereParts = append(whereParts, "("+strings.Join(termConditions, " AND ")+")")
	}
	whereClause := "(" + strings.Join(whereParts, " OR ") + ")"
	if !includeClosed {
		whereClause += " AND i.status NOT IN ('completed', 'cancelled')"
	}

	var rankCases []string
	rankCases = append(rankCases, fmt.Sprintf("WHEN LOWER(i.title) = %s THEN 0", phraseParam))
	rankCases = append(rankCases, fmt.Sprintf("WHEN LOWER(i.title) LIKE %s THEN 1", phraseStartsWith))
	rankCases = append(rankCases, fmt.Sprintf("WHEN LOWER(i.title) LIKE %s THEN 2", phraseContains))
	if len(termParams) > 1 {
		var titleTerms []string
		for _, tp := range termParams {
			titleTerms = append(titleTerms, fmt.Sprintf("LOWER(i.title) LIKE '%s' || %s || '%s'", "%", tp, "%"))
		}
		rankCases = append(rankCases, fmt.Sprintf("WHEN (%s) THEN 3", strings.Join(titleTerms, " AND ")))
	}
	rankCases = append(rankCases, fmt.Sprintf("WHEN LOWER(COALESCE(i.description, '')) LIKE %s THEN 4", phraseContains))
	rankExpr := "CASE " + strings.Join(rankCases, " ") + " ELSE 5 END"
	cancelledRank := fmt.Sprintf(
		"CASE WHEN i.status = 'cancelled' AND LOWER(i.title) <> %s THEN 1 ELSE 0 END",
		phraseParam,
	)
	matchSourceExpr := fmt.Sprintf(`CASE
		WHEN LOWER(i.title) LIKE %s THEN 'title'
		ELSE 'description'
	END`, phraseContains)
	if len(termParams) > 1 {
		var titleTerms []string
		for _, tp := range termParams {
			titleTerms = append(titleTerms, fmt.Sprintf("LOWER(i.title) LIKE '%s' || %s || '%s'", "%", tp, "%"))
		}
		matchSourceExpr = fmt.Sprintf(`CASE
			WHEN LOWER(i.title) LIKE %s THEN 'title'
			WHEN (%s) THEN 'title'
			ELSE 'description'
		END`,
			phraseContains, strings.Join(titleTerms, " AND "),
		)
	}

	limitParam := nextArg(nil)
	offsetParam := nextArg(nil)
	query := fmt.Sprintf(`SELECT i.id, i.workspace_id, i.title, i.description, i.icon,
		i.status, i.priority, i.lead_type, i.lead_id,
		i.start_date, i.due_date,
		i.created_at, i.updated_at,
		COUNT(*) OVER() AS total_count,
		%s AS match_source
	FROM initiative i
	WHERE i.workspace_id = %s AND %s
	ORDER BY %s, %s, i.updated_at DESC
	LIMIT %s OFFSET %s`,
		matchSourceExpr,
		wsParam,
		whereClause,
		cancelledRank,
		rankExpr,
		limitParam,
		offsetParam,
	)
	return query, args
}

func (h *Handler) SearchInitiatives(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workspaceID := h.resolveWorkspaceID(r)
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "q parameter is required")
		return
	}
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 50 {
		limit = 50
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	includeClosed := r.URL.Query().Get("include_closed") == "true"
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}
	terms := splitSearchTerms(q)
	sqlQuery, args := buildInitiativeSearchQuery(q, terms, includeClosed)
	args[1] = wsUUID
	args[len(args)-2] = limit
	args[len(args)-1] = offset

	type initiativeSearchRow struct {
		initiative  db.Initiative
		totalCount  int64
		matchSource string
	}
	var results []initiativeSearchRow
	err := runSearchQuery(ctx, h.TxStarter, sqlQuery, args, func(rows pgx.Rows) error {
		for rows.Next() {
			var row initiativeSearchRow
			if err := rows.Scan(
				&row.initiative.ID,
				&row.initiative.WorkspaceID,
				&row.initiative.Title,
				&row.initiative.Description,
				&row.initiative.Icon,
				&row.initiative.Status,
				&row.initiative.Priority,
				&row.initiative.LeadType,
				&row.initiative.LeadID,
				&row.initiative.StartDate,
				&row.initiative.DueDate,
				&row.initiative.CreatedAt,
				&row.initiative.UpdatedAt,
				&row.totalCount,
				&row.matchSource,
			); err != nil {
				return fmt.Errorf("scan: %w", err)
			}
			results = append(results, row)
		}
		return rows.Err()
	})
	if err != nil {
		if isSearchStatementTimeout(err) {
			slog.Warn("search initiatives timed out",
				"workspace_id", workspaceID,
				"query", q,
				"timeout", searchStatementTimeout)
			writeError(w, http.StatusServiceUnavailable, "search timed out; please refine your query or try again")
			return
		}
		slog.Warn("search initiatives failed", "error", err, "workspace_id", workspaceID, "query", q)
		writeError(w, http.StatusInternalServerError, "failed to search initiatives")
		return
	}

	var total int64
	if len(results) > 0 {
		total = results[0].totalCount
	}
	projectCountMap := map[string]int64{}
	statsMap := map[string]db.GetInitiativeIssueStatsRow{}
	if len(results) > 0 {
		ids := make([]pgtype.UUID, len(results))
		for i, row := range results {
			ids[i] = row.initiative.ID
		}
		counts, err := h.Queries.GetInitiativeProjectCounts(ctx, ids)
		if err == nil {
			for _, c := range counts {
				projectCountMap[uuidToString(c.InitiativeID)] = c.ProjectCount
			}
		}
		stats, err := h.Queries.GetInitiativeIssueStats(ctx, ids)
		if err == nil {
			for _, s := range stats {
				statsMap[uuidToString(s.InitiativeID)] = s
			}
		}
	}

	resp := make([]SearchInitiativeResponse, len(results))
	for i, row := range results {
		ir := initiativeToResponse(row.initiative)
		ir.ProjectCount = projectCountMap[ir.ID]
		if s, ok := statsMap[ir.ID]; ok {
			ir.IssueCount = s.TotalCount
			ir.DoneCount = s.DoneCount
		}
		spr := SearchInitiativeResponse{
			InitiativeResponse: ir,
			MatchSource:        row.matchSource,
		}
		if row.matchSource == "description" {
			desc := ""
			if row.initiative.Description.Valid {
				desc = row.initiative.Description.String
			}
			if desc != "" {
				snippet := extractSnippet(desc, q)
				spr.MatchedSnippet = &snippet
			}
		}
		resp[i] = spr
	}
	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	writeJSON(w, http.StatusOK, map[string]any{
		"initiatives": resp,
		"total":       total,
	})
}
