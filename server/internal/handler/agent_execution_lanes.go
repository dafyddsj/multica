package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/executionlane"
	"github.com/multica-ai/multica/server/internal/featureflags"
	"github.com/multica-ai/multica/server/pkg/agent"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func (h *Handler) agentExecutionLanesEnabled(ctx context.Context) bool {
	return featureflags.AgentExecutionLanesEnabled(ctx, h.FeatureFlags)
}

func agentLanesFromDB(a db.Agent) executionlane.AgentLanes {
	return executionlane.AgentLanes{
		Model:                    a.Model.String,
		ThinkingLevel:            a.ThinkingLevel.String,
		ServiceTier:              a.ServiceTier.String,
		LightweightModel:         a.LightweightModel.String,
		LightweightThinkingLevel: a.LightweightThinkingLevel.String,
		StartLightweight:         a.StartLightweight,
		FailoverRuntimeID:        uuidToString(a.FailoverRuntimeID),
		FailoverModel:            a.FailoverModel.String,
		FailoverThinkingLevel:    a.FailoverThinkingLevel.String,
		FailoverServiceTier:      a.FailoverServiceTier.String,
	}
}

func taskRuntimeMatchesAgent(task db.AgentTaskQueue, a db.Agent) bool {
	if a.RuntimeID == task.RuntimeID {
		return true
	}
	return executionlane.ParseLane(task.ExecutionLane) == executionlane.LaneFailover &&
		task.RetryOfTaskID.Valid &&
		a.FailoverRuntimeID.Valid &&
		a.FailoverRuntimeID == task.RuntimeID
}

func applyClaimLaneSelection(a db.Agent, task db.AgentTaskQueue, enabled bool) executionlane.Selection {
	if !enabled {
		return executionlane.ResolveClaim(agentLanesFromDB(a), executionlane.LanePrimary, "")
	}
	return executionlane.ResolveClaim(
		agentLanesFromDB(a),
		executionlane.ParseLane(task.ExecutionLane),
		task.ModelOverride.String,
	)
}

type laneWriteError struct {
	status  int
	message string
}

func writeLaneError(w http.ResponseWriter, err *laneWriteError) {
	writeError(w, err.status, err.message)
}

type laneClearFlags struct {
	lightweightModel         bool
	lightweightThinkingLevel bool
	failoverRuntimeID        bool
	failoverModel            bool
	failoverThinkingLevel    bool
	failoverServiceTier      bool
}

func (h *Handler) applyCreateAgentLanes(
	w http.ResponseWriter,
	r *http.Request,
	req CreateAgentRequest,
	runtime db.AgentRuntime,
	params *db.CreateAgentParams,
) bool {
	if !h.agentExecutionLanesEnabled(r.Context()) {
		return true
	}
	if err := h.validateLaneThinking(r, runtime, req.LightweightThinkingLevel); err != nil {
		writeLaneError(w, err)
		return false
	}
	failoverRuntime := runtime
	if req.FailoverRuntimeID != "" {
		resolved, ok := h.loadLaneRuntime(w, r, runtime.WorkspaceID, req.FailoverRuntimeID, "failover_runtime_id")
		if !ok {
			return false
		}
		failoverRuntime = resolved
		if resolved.ID != runtime.ID {
			params.FailoverRuntimeID = resolved.ID
		}
	}
	if err := h.validateLaneThinking(r, failoverRuntime, req.FailoverThinkingLevel); err != nil {
		writeLaneError(w, err)
		return false
	}
	if !agent.IsKnownServiceTier(failoverRuntime.Provider, req.FailoverServiceTier) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf(
			"failover_service_tier %q is not a recognised value for runtime %q",
			req.FailoverServiceTier, failoverRuntime.Provider,
		))
		return false
	}
	params.LightweightModel = pgtype.Text{String: req.LightweightModel, Valid: req.LightweightModel != ""}
	params.LightweightThinkingLevel = pgtype.Text{String: req.LightweightThinkingLevel, Valid: req.LightweightThinkingLevel != ""}
	if req.StartLightweight != nil {
		params.StartLightweight = *req.StartLightweight
	}
	params.FailoverModel = pgtype.Text{String: req.FailoverModel, Valid: req.FailoverModel != ""}
	params.FailoverThinkingLevel = pgtype.Text{String: req.FailoverThinkingLevel, Valid: req.FailoverThinkingLevel != ""}
	params.FailoverServiceTier = pgtype.Text{String: req.FailoverServiceTier, Valid: req.FailoverServiceTier != ""}
	return true
}

func (h *Handler) applyUpdateAgentLanes(
	w http.ResponseWriter,
	r *http.Request,
	req UpdateAgentRequest,
	existing db.Agent,
	targetRuntimeID pgtype.UUID,
	targetProvider string,
	params *db.UpdateAgentParams,
) (laneClearFlags, bool) {
	var clears laneClearFlags
	if !h.agentExecutionLanesEnabled(r.Context()) {
		return clears, true
	}

	if req.RuntimeID != nil {
		if req.LightweightModel == nil {
			clears.lightweightModel = existing.LightweightModel.Valid
		}
		if req.LightweightThinkingLevel == nil {
			clears.lightweightThinkingLevel = existing.LightweightThinkingLevel.Valid
		}
		inheritsFailover := !existing.FailoverRuntimeID.Valid && req.FailoverRuntimeID == nil
		if inheritsFailover {
			if req.FailoverModel == nil {
				clears.failoverModel = existing.FailoverModel.Valid
			}
			if req.FailoverThinkingLevel == nil {
				clears.failoverThinkingLevel = existing.FailoverThinkingLevel.Valid
			}
			if req.FailoverServiceTier == nil {
				clears.failoverServiceTier = existing.FailoverServiceTier.Valid
			}
		}
	}

	if req.StartLightweight != nil {
		params.StartLightweight = pgtype.Bool{Bool: *req.StartLightweight, Valid: true}
	}

	if req.LightweightModel != nil {
		if *req.LightweightModel == "" {
			clears.lightweightModel = true
		} else {
			params.LightweightModel = pgtype.Text{String: *req.LightweightModel, Valid: true}
		}
	}
	if req.LightweightThinkingLevel != nil {
		if *req.LightweightThinkingLevel == "" {
			clears.lightweightThinkingLevel = true
		} else {
			provider := targetProvider
			runtimeID := targetRuntimeID
			if provider == "" {
				var ok bool
				provider, ok = h.resolveAgentProvider(r, existing.WorkspaceID, runtimeID)
				if !ok {
					writeError(w, http.StatusInternalServerError, "failed to resolve runtime for lightweight_thinking_level validation")
					return clears, false
				}
			}
			if err := h.validateLaneThinkingForProvider(r, provider, runtimeID, *req.LightweightThinkingLevel); err != nil {
				writeLaneError(w, err)
				return clears, false
			}
			params.LightweightThinkingLevel = pgtype.Text{String: *req.LightweightThinkingLevel, Valid: true}
		}
	}

	failoverRuntimeID := existing.FailoverRuntimeID
	failoverProvider := ""
	if req.FailoverRuntimeID != nil {
		if *req.FailoverRuntimeID == "" {
			clears.failoverRuntimeID = true
			failoverRuntimeID = pgtype.UUID{}
		} else {
			resolved, ok := h.loadLaneRuntime(w, r, existing.WorkspaceID, *req.FailoverRuntimeID, "failover_runtime_id")
			if !ok {
				return clears, false
			}
			if resolved.ID == targetRuntimeID {
				clears.failoverRuntimeID = true
				failoverRuntimeID = pgtype.UUID{}
				failoverProvider = resolved.Provider
			} else {
				params.FailoverRuntimeID = resolved.ID
				failoverRuntimeID = resolved.ID
				failoverProvider = resolved.Provider
			}
		}
	}

	if req.FailoverModel != nil {
		if *req.FailoverModel == "" {
			clears.failoverModel = true
		} else {
			params.FailoverModel = pgtype.Text{String: *req.FailoverModel, Valid: true}
		}
	}
	if req.FailoverThinkingLevel != nil {
		if *req.FailoverThinkingLevel == "" {
			clears.failoverThinkingLevel = true
		} else {
			provider, runtimeID, err := h.failoverProviderForUpdate(r, existing, targetRuntimeID, targetProvider, failoverRuntimeID, failoverProvider)
			if err != nil {
				writeLaneError(w, err)
				return clears, false
			}
			if vErr := h.validateLaneThinkingForProvider(r, provider, runtimeID, *req.FailoverThinkingLevel); vErr != nil {
				writeLaneError(w, vErr)
				return clears, false
			}
			params.FailoverThinkingLevel = pgtype.Text{String: *req.FailoverThinkingLevel, Valid: true}
		}
	}
	if req.FailoverServiceTier != nil {
		if *req.FailoverServiceTier == "" {
			clears.failoverServiceTier = true
		} else {
			provider, _, err := h.failoverProviderForUpdate(r, existing, targetRuntimeID, targetProvider, failoverRuntimeID, failoverProvider)
			if err != nil {
				writeLaneError(w, err)
				return clears, false
			}
			if !agent.IsKnownServiceTier(provider, *req.FailoverServiceTier) {
				writeError(w, http.StatusBadRequest, fmt.Sprintf(
					"failover_service_tier %q is not a recognised value for runtime %q",
					*req.FailoverServiceTier, provider,
				))
				return clears, false
			}
			params.FailoverServiceTier = pgtype.Text{String: *req.FailoverServiceTier, Valid: true}
		}
	}
	return clears, true
}

func (h *Handler) failoverProviderForUpdate(
	r *http.Request,
	existing db.Agent,
	targetRuntimeID pgtype.UUID,
	targetProvider string,
	failoverRuntimeID pgtype.UUID,
	knownProvider string,
) (provider string, runtimeID pgtype.UUID, err *laneWriteError) {
	if knownProvider != "" {
		if failoverRuntimeID.Valid {
			return knownProvider, failoverRuntimeID, nil
		}
		return knownProvider, targetRuntimeID, nil
	}
	if failoverRuntimeID.Valid {
		provider, ok := h.resolveAgentProvider(r, existing.WorkspaceID, failoverRuntimeID)
		if !ok {
			return "", pgtype.UUID{}, &laneWriteError{status: http.StatusInternalServerError, message: "failed to resolve failover runtime"}
		}
		return provider, failoverRuntimeID, nil
	}
	provider = targetProvider
	runtimeID = targetRuntimeID
	if provider == "" {
		var ok bool
		provider, ok = h.resolveAgentProvider(r, existing.WorkspaceID, runtimeID)
		if !ok {
			return "", pgtype.UUID{}, &laneWriteError{status: http.StatusInternalServerError, message: "failed to resolve runtime for failover validation"}
		}
	}
	return provider, runtimeID, nil
}

func (h *Handler) loadLaneRuntime(
	w http.ResponseWriter,
	r *http.Request,
	workspaceID pgtype.UUID,
	runtimeID, field string,
) (db.AgentRuntime, bool) {
	runtimeUUID, ok := parseUUIDOrBadRequest(w, runtimeID, field)
	if !ok {
		return db.AgentRuntime{}, false
	}
	runtime, err := h.Queries.GetAgentRuntimeForWorkspace(r.Context(), db.GetAgentRuntimeForWorkspaceParams{
		ID:          runtimeUUID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+field)
		return db.AgentRuntime{}, false
	}
	member, ok := h.workspaceMember(w, r, uuidToString(workspaceID))
	if !ok {
		return db.AgentRuntime{}, false
	}
	if !canUseRuntimeForAgent(member, runtime) {
		writeError(w, http.StatusForbidden, "this failover runtime is private; only its owner can bind it")
		return db.AgentRuntime{}, false
	}
	return runtime, true
}

func (h *Handler) validateLaneThinking(r *http.Request, runtime db.AgentRuntime, value string) *laneWriteError {
	return h.validateLaneThinkingForProvider(r, runtime.Provider, runtime.ID, value)
}

func (h *Handler) validateLaneThinkingForProvider(r *http.Request, provider string, runtimeID pgtype.UUID, value string) *laneWriteError {
	if !agent.IsKnownThinkingValue(provider, value) {
		return &laneWriteError{status: http.StatusBadRequest, message: thinkingLevelRejection(provider, value)}
	}
	if value == "" {
		return nil
	}
	switch h.acpThinkingDecision(r.Context(), provider, runtimeID) {
	case acpEffortAbsent:
		return &laneWriteError{status: http.StatusBadRequest, message: thinkingCapabilityRejection(provider)}
	case acpEffortUnknown:
		return &laneWriteError{status: http.StatusBadRequest, message: thinkingCapabilityUnknownRejection(provider)}
	}
	return nil
}

func (h *Handler) applyLaneClears(r *http.Request, updated db.Agent, clears laneClearFlags) (db.Agent, error) {
	var err error
	if clears.lightweightModel {
		updated, err = h.Queries.ClearAgentLightweightModel(r.Context(), updated.ID)
		if err != nil {
			return updated, fmt.Errorf("clear lightweight_model: %w", err)
		}
	}
	if clears.lightweightThinkingLevel {
		updated, err = h.Queries.ClearAgentLightweightThinkingLevel(r.Context(), updated.ID)
		if err != nil {
			return updated, fmt.Errorf("clear lightweight_thinking_level: %w", err)
		}
	}
	if clears.failoverRuntimeID {
		updated, err = h.Queries.ClearAgentFailoverRuntimeID(r.Context(), updated.ID)
		if err != nil {
			return updated, fmt.Errorf("clear failover_runtime_id: %w", err)
		}
	}
	if clears.failoverModel {
		updated, err = h.Queries.ClearAgentFailoverModel(r.Context(), updated.ID)
		if err != nil {
			return updated, fmt.Errorf("clear failover_model: %w", err)
		}
	}
	if clears.failoverThinkingLevel {
		updated, err = h.Queries.ClearAgentFailoverThinkingLevel(r.Context(), updated.ID)
		if err != nil {
			return updated, fmt.Errorf("clear failover_thinking_level: %w", err)
		}
	}
	if clears.failoverServiceTier {
		updated, err = h.Queries.ClearAgentFailoverServiceTier(r.Context(), updated.ID)
		if err != nil {
			return updated, fmt.Errorf("clear failover_service_tier: %w", err)
		}
	}
	return updated, nil
}
