package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/executionlane"
	"github.com/multica-ai/multica/server/internal/featureflags"
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
	sel := executionlane.Initial(agentLanes(agent), featureflags.AgentExecutionLanesEnabled(ctx, s.FeatureFlags))
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
		featureflags.AgentExecutionLanesEnabled(ctx, s.FeatureFlags),
	)
}

func retryParamsForLaneHop(parentID pgtype.UUID, sel executionlane.Selection, overlay runtimeMCPOverlayData) db.CreateRetryTaskParams {
	params := db.CreateRetryTaskParams{
		NewTaskID:            dbid.NewV7(),
		ID:                   parentID,
		ForceFreshSession:    pgtype.Bool{Bool: true, Valid: true},
		ExecutionLane:        pgtype.Text{String: string(sel.Lane), Valid: true},
		ModelOverride:        pgtype.Text{String: sel.Model, Valid: sel.Model != ""},
		RuntimeMcpOverlay:    overlay.Overlay,
		RuntimeConnectedApps: overlay.ConnectedApps,
	}
	if sel.RuntimeID != "" {
		if id, err := util.ParseUUID(sel.RuntimeID); err == nil {
			params.RuntimeID = id
		}
	}
	return params
}
