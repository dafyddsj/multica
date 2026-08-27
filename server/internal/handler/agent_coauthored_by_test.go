package handler

import (
	"net/http"
	"testing"

	"github.com/multica-ai/multica/server/internal/testutil"
)

func TestUpdateAgent_CoAuthoredByEmailTriState(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	agentID := dbfx.Agent(t, "coauthor-email-agent", handlerTestRuntimeID(t))

	var set AgentResponse
	testutil.Call(t, testHandler.UpdateAgent, withURLParam(
		newRequest(http.MethodPut, "/api/agents/"+agentID, map[string]any{
			"co_authored_by_email": "Review@Example.com",
		}),
		"id",
		agentID,
	)).Want(http.StatusOK).JSON(&set)
	if set.CoAuthoredByEmail != "review@example.com" {
		t.Fatalf("set co_authored_by_email = %q, want review@example.com", set.CoAuthoredByEmail)
	}

	var preserved AgentResponse
	testutil.Call(t, testHandler.UpdateAgent, withURLParam(
		newRequest(http.MethodPut, "/api/agents/"+agentID, map[string]any{
			"description": "leave email alone",
		}),
		"id",
		agentID,
	)).Want(http.StatusOK).JSON(&preserved)
	if preserved.CoAuthoredByEmail != "review@example.com" {
		t.Fatalf("omitted update co_authored_by_email = %q, want review@example.com", preserved.CoAuthoredByEmail)
	}

	testutil.Call(t, testHandler.UpdateAgent, withURLParam(
		newRequest(http.MethodPut, "/api/agents/"+agentID, map[string]any{
			"co_authored_by_email": "not-an-email",
		}),
		"id",
		agentID,
	)).Want(http.StatusBadRequest)

	var stillSet AgentResponse
	testutil.Call(t, testHandler.GetAgent, withURLParam(
		newRequest(http.MethodGet, "/api/agents/"+agentID, nil),
		"id",
		agentID,
	)).Want(http.StatusOK).JSON(&stillSet)
	if stillSet.CoAuthoredByEmail != "review@example.com" {
		t.Fatalf("invalid update mutated co_authored_by_email = %q", stillSet.CoAuthoredByEmail)
	}

	var cleared AgentResponse
	testutil.Call(t, testHandler.UpdateAgent, withURLParam(
		newRequest(http.MethodPut, "/api/agents/"+agentID, map[string]any{
			"co_authored_by_email": "",
		}),
		"id",
		agentID,
	)).Want(http.StatusOK).JSON(&cleared)
	if cleared.CoAuthoredByEmail != "" {
		t.Fatalf("cleared co_authored_by_email = %q, want empty", cleared.CoAuthoredByEmail)
	}
}
