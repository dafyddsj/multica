package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/executionlane"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestTaskRuntimeMatchesAgent(t *testing.T) {
	primary := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	failover := pgtype.UUID{Bytes: [16]byte{2}, Valid: true}
	agent := db.Agent{RuntimeID: primary, FailoverRuntimeID: failover}
	same := db.AgentTaskQueue{RuntimeID: primary, ExecutionLane: "primary"}
	if !taskRuntimeMatchesAgent(same, agent) {
		t.Fatal("primary runtime must match")
	}
	foreign := db.AgentTaskQueue{RuntimeID: failover, ExecutionLane: "primary"}
	if taskRuntimeMatchesAgent(foreign, agent) {
		t.Fatal("foreign runtime on primary lane must not match")
	}
	hop := db.AgentTaskQueue{
		RuntimeID:     failover,
		ExecutionLane: "failover",
		RetryOfTaskID: pgtype.UUID{Bytes: [16]byte{3}, Valid: true},
	}
	if !taskRuntimeMatchesAgent(hop, agent) {
		t.Fatal("stamped failover child on failover_runtime_id must match")
	}
	if got := applyClaimLaneSelection(db.Agent{
		Model:         pgtype.Text{String: "opus", Valid: true},
		FailoverModel: pgtype.Text{String: "sonnet", Valid: true},
	}, hop); got.Model != "sonnet" {
		t.Fatalf("unstamped failover hop must use the failover model, got %+v", got)
	}
	_ = executionlane.LaneFailover
}

func TestCreateAndUpdateAgent_ExecutionLanes(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	runtimeID := createClaudeProviderRuntime(t)
	t.Cleanup(func() {
		testPool.Exec(context.Background(),
			`DELETE FROM agent WHERE workspace_id = $1 AND name LIKE 'lanes-on-%'`,
			testWorkspaceID,
		)
	})

	w := httptest.NewRecorder()
	testHandler.CreateAgent(w, newRequest(http.MethodPost, "/api/agents", map[string]any{
		"name":                       "lanes-on-create",
		"runtime_id":                 runtimeID,
		"visibility":                 "private",
		"model":                      "opus",
		"lightweight_model":          "haiku",
		"lightweight_thinking_level": "low",
		"start_lightweight":          true,
		"failover_model":             "sonnet",
		"failover_thinking_level":    "high",
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created map[string]any
	_ = json.NewDecoder(w.Body).Decode(&created)
	if created["lightweight_model"] != "haiku" || created["failover_model"] != "sonnet" {
		t.Fatalf("create lanes: %+v", created)
	}
	if created["start_lightweight"] != true {
		t.Fatalf("start_lightweight want true, got %v", created["start_lightweight"])
	}
	id, _ := created["id"].(string)

	upd := httptest.NewRecorder()
	testHandler.UpdateAgent(upd, withURLParam(
		newRequest(http.MethodPut, "/api/agents/"+id, map[string]any{
			"lightweight_model": "",
			"failover_model":    "",
		}),
		"id", id,
	))
	if upd.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", upd.Code, upd.Body.String())
	}
	var updated map[string]any
	_ = json.NewDecoder(upd.Body).Decode(&updated)
	if updated["lightweight_model"] != "" || updated["failover_model"] != "" {
		t.Fatalf("clear lanes: %+v", updated)
	}
}

func TestCreateAgent_ExecutionLanesRejectsInvalidThinking(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	runtimeID := createClaudeProviderRuntime(t)
	w := httptest.NewRecorder()
	testHandler.CreateAgent(w, newRequest(http.MethodPost, "/api/agents", map[string]any{
		"name":                       "lanes-on-bad-thinking",
		"runtime_id":                 runtimeID,
		"visibility":                 "private",
		"lightweight_thinking_level": "none",
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
