package handler

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/multica-ai/multica/server/internal/testutil"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func TestPauseResumeAgent(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	agentID := dbfx.Agent(t, "pause-resume-agent", handlerTestRuntimeID(t))

	var paused AgentResponse
	testutil.Call(t, testHandler.PauseAgent, testutil.WithURLParams(
		newRequest(http.MethodPost, "/api/agents/"+agentID+"/pause", nil),
		"id", agentID,
	)).Want(http.StatusOK).JSON(&paused)
	if paused.PausedAt == nil || paused.PausedBy == nil || *paused.PausedBy != testUserID {
		t.Fatalf("pause response = %+v, want paused_at and paused_by=%s", paused, testUserID)
	}

	var pausedAgain AgentResponse
	testutil.Call(t, testHandler.PauseAgent, testutil.WithURLParams(
		newRequest(http.MethodPost, "/api/agents/"+agentID+"/pause", nil),
		"id", agentID,
	)).Want(http.StatusOK).JSON(&pausedAgain)
	if pausedAgain.PausedAt == nil {
		t.Fatal("idempotent pause cleared paused_at")
	}

	var resumed AgentResponse
	testutil.Call(t, testHandler.ResumeAgent, testutil.WithURLParams(
		newRequest(http.MethodPost, "/api/agents/"+agentID+"/resume", nil),
		"id", agentID,
	)).Want(http.StatusOK).JSON(&resumed)
	if resumed.PausedAt != nil || resumed.PausedBy != nil {
		t.Fatalf("resume response = %+v, want pause fields cleared", resumed)
	}

	var resumedAgain AgentResponse
	testutil.Call(t, testHandler.ResumeAgent, testutil.WithURLParams(
		newRequest(http.MethodPost, "/api/agents/"+agentID+"/resume", nil),
		"id", agentID,
	)).Want(http.StatusOK).JSON(&resumedAgain)
	if resumedAgain.PausedAt != nil || resumedAgain.PausedBy != nil {
		t.Fatalf("idempotent resume response = %+v, want pause fields cleared", resumedAgain)
	}
}

func TestPauseAgentRejectsArchivedAgent(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	agentID := dbfx.Agent(t, "pause-archived-agent", handlerTestRuntimeID(t), testutil.Cols{
		"archived_at": testutil.Raw("now()"),
		"archived_by": testUserID,
	})
	resp := testutil.Call(t, testHandler.PauseAgent, testutil.WithURLParams(
		newRequest(http.MethodPost, "/api/agents/"+agentID+"/pause", nil),
		"id", agentID,
	)).Want(http.StatusConflict)
	if !strings.Contains(resp.Text(), "already archived") {
		t.Fatalf("pause archived response = %q", resp.Text())
	}
	if dbfx.Count(t, `SELECT count(*) FROM agent WHERE id = $1 AND paused_at IS NOT NULL`, agentID) != 0 {
		t.Fatal("archived agent was paused")
	}
}

func TestPauseResumeAgentRejectSystemAgent(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	agentID := dbfx.Agent(t, "pause-system-agent", handlerTestRuntimeID(t), testutil.Cols{
		"kind":       "system",
		"system_key": "pause-test",
	})
	for _, tc := range []struct {
		name    string
		action  string
		handler http.HandlerFunc
	}{
		{name: "pause", action: "pause", handler: testHandler.PauseAgent},
		{name: "resume", action: "resume", handler: testHandler.ResumeAgent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := testutil.Call(t, tc.handler, testutil.WithURLParams(
				newRequest(http.MethodPost, "/api/agents/"+agentID+"/"+tc.action, nil),
				"id", agentID,
			)).Want(http.StatusBadRequest)
			if !strings.Contains(resp.Text(), "built into Multica") {
				t.Fatalf("%s system agent response = %q", tc.action, resp.Text())
			}
		})
	}
}

func TestAssignIssueToPausedAgent(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	agentID := dbfx.Agent(t, "assign-paused-agent", handlerTestRuntimeID(t), testutil.Cols{
		"paused_at": testutil.Raw("now()"),
		"paused_by": testUserID,
	})
	resp := testutil.Call(t, testHandler.CreateIssue,
		newRequest(http.MethodPost, "/api/issues?workspace_id="+testWorkspaceID, map[string]any{
			"title":         "must not assign paused agent",
			"status":        "backlog",
			"assignee_type": "agent",
			"assignee_id":   agentID,
		}),
	).Want(http.StatusBadRequest)
	if !strings.Contains(resp.Text(), "cannot assign to paused agent") {
		t.Fatalf("assign paused response = %q", resp.Text())
	}
}

func TestPauseAgentDoesNotCancelTasks(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	runtimeID := handlerTestRuntimeID(t)
	agentID := dbfx.Agent(t, "pause-keeps-tasks-agent", runtimeID)
	issueID := dbfx.Issue(t, "pause keeps queued task")
	queuedID := dbfx.Task(t, agentID, testutil.Cols{
		"runtime_id": runtimeID,
		"issue_id":   issueID,
	})
	runningID := dbfx.Task(t, agentID, testutil.Cols{
		"runtime_id": runtimeID,
		"status":     "running",
		"started_at": testutil.Raw("now()"),
	})

	testutil.Call(t, testHandler.PauseAgent, testutil.WithURLParams(
		newRequest(http.MethodPost, "/api/agents/"+agentID+"/pause", nil),
		"id", agentID,
	)).Want(http.StatusOK)
	for taskID, want := range map[string]string{queuedID: "queued", runningID: "running"} {
		var got string
		dbfx.QueryRow(t, `SELECT status FROM agent_task_queue WHERE id = $1`, taskID).Scan(&got)
		if got != want {
			t.Errorf("task %s after pause = %q, want %q", taskID, got, want)
		}
	}
	if _, err := testHandler.Queries.ClaimAgentTask(t.Context(), db.ClaimAgentTaskParams{
		AgentID:          parseUUID(agentID),
		RuntimeID:        parseUUID(runtimeID),
		PrepareLeaseSecs: 30,
		RuntimeStaleSecs: 300,
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("claim paused agent task: got %v, want pgx.ErrNoRows", err)
	}

	testutil.Call(t, testHandler.ResumeAgent, testutil.WithURLParams(
		newRequest(http.MethodPost, "/api/agents/"+agentID+"/resume", nil),
		"id", agentID,
	)).Want(http.StatusOK)
	claimed, err := testHandler.Queries.ClaimAgentTask(t.Context(), db.ClaimAgentTaskParams{
		AgentID:          parseUUID(agentID),
		RuntimeID:        parseUUID(runtimeID),
		PrepareLeaseSecs: 30,
		RuntimeStaleSecs: 300,
	})
	if err != nil {
		t.Fatalf("claim resumed agent task: %v", err)
	}
	if uuidToString(claimed.ID) != queuedID {
		t.Fatalf("claimed task after resume = %s, want %s", uuidToString(claimed.ID), queuedID)
	}

	testutil.Call(t, testHandler.ArchiveAgent, testutil.WithURLParams(
		newRequest(http.MethodPost, "/api/agents/"+agentID+"/archive", nil),
		"id", agentID,
	)).Want(http.StatusOK)
	for _, taskID := range []string{queuedID, runningID} {
		var got string
		dbfx.QueryRow(t, `SELECT status FROM agent_task_queue WHERE id = $1`, taskID).Scan(&got)
		if got != "cancelled" {
			t.Errorf("task %s after archive = %q, want cancelled", taskID, got)
		}
	}
}
