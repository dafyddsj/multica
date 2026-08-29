package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/executionlane"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/dbid"
)

func agentLanes(a db.Agent) executionlane.AgentLanes {
	return executionlane.AgentLanes{
		Model:                    a.Model.String,
		ThinkingLevel:            a.ThinkingLevel.String,
		ServiceTier:              a.ServiceTier.String,
		LightweightModel:         a.LightweightModel.String,
		LightweightThinkingLevel: a.LightweightThinkingLevel.String,
		StartLightweight:         a.StartLightweight,
		FailoverRuntimeID:        util.UUIDToString(a.FailoverRuntimeID),
		FailoverModel:            a.FailoverModel.String,
		FailoverThinkingLevel:    a.FailoverThinkingLevel.String,
		FailoverServiceTier:      a.FailoverServiceTier.String,
	}
}

type laneStamp struct {
	Lane              pgtype.Text
	ModelOverride     pgtype.Text
	ForceFreshSession bool
}

func (s *TaskService) initialLaneStamp(ctx context.Context, agent db.Agent) laneStamp {
	sel := executionlane.Initial(agentLanes(agent))
	stamp := laneStamp{
		Lane:              pgtype.Text{String: string(sel.Lane), Valid: true},
		ForceFreshSession: sel.ForceFreshSession,
	}
	if sel.Lane != executionlane.LanePrimary && sel.Model != "" {
		stamp.ModelOverride = pgtype.Text{String: sel.Model, Valid: true}
	}
	return stamp
}

func applyLaneStampToFresh(forceFresh pgtype.Bool, stamp laneStamp) pgtype.Bool {
	if stamp.ForceFreshSession {
		return pgtype.Bool{Bool: true, Valid: true}
	}
	return forceFresh
}

func (s *TaskService) nextLaneHop(ctx context.Context, parent db.AgentTaskQueue, agent db.Agent, reason string) (executionlane.Selection, bool) {
	if parent.AutopilotRunID.Valid {
		return executionlane.Selection{}, false
	}
	return executionlane.NextOnFailure(
		agentLanes(agent),
		executionlane.ParseLane(parent.ExecutionLane),
		reason,
	)
}

func claimRuntimeAllowed(agent db.Agent, runtimeID pgtype.UUID) bool {
	if agent.RuntimeID == runtimeID {
		return true
	}
	return agent.FailoverRuntimeID.Valid && agent.FailoverRuntimeID == runtimeID
}

func retryParamsForLaneHop(parent db.AgentTaskQueue, sel executionlane.Selection, overlay runtimeMCPOverlayData) db.CreateRetryTaskParams {
	params := db.CreateRetryTaskParams{
		NewTaskID:         dbid.NewV7(),
		ID:                parent.ID,
		ForceFreshSession: pgtype.Bool{Bool: true, Valid: true},
		// A hop is independent of the infra retry budget. Persist a ceiling
		// that matches the child's attempt so the row does not read
		// attempt=N / max_attempts=1 (MUL-4910).
		MaxAttempts:          pgtype.Int4{Int32: parent.Attempt + 1, Valid: true},
		ExecutionLane:        pgtype.Text{String: string(sel.Lane), Valid: true},
		ModelOverride:        pgtype.Text{String: sel.Model, Valid: true},
		RuntimeMcpOverlay:    overlay.Overlay,
		RuntimeConnectedApps: overlay.ConnectedApps,
	}
	if sel.RuntimeID != "" {
		params.RuntimeID = util.MustParseUUID(sel.RuntimeID)
	}
	return params
}
