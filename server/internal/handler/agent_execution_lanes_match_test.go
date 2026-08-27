package handler

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestApplyClaimLaneSelectionOverride(t *testing.T) {
	agent := db.Agent{
		Model:            pgtype.Text{String: "opus", Valid: true},
		LightweightModel: pgtype.Text{String: "haiku", Valid: true},
		StartLightweight: true,
	}
	task := db.AgentTaskQueue{
		ExecutionLane: "lightweight",
		ModelOverride: pgtype.Text{String: "stamped-haiku", Valid: true},
	}
	got := applyClaimLaneSelection(agent, task, true)
	if got.Model != "stamped-haiku" {
		t.Fatalf("want stamped override, got %+v", got)
	}
	got = applyClaimLaneSelection(agent, task, false)
	if got.Model != "opus" {
		t.Fatalf("disabled flag must use primary model, got %+v", got)
	}
}
