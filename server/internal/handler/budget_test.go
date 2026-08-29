package handler

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/budgetpolicy"
	"github.com/multica-ai/multica/server/internal/testutil"
)

func budgetAuth(userID string) []string {
	return []string{"X-User-ID", userID, "X-Workspace-ID", testWorkspaceID}
}

func budgetReq(method, path string, body any, userID string) *http.Request {
	return testutil.WithHeaders(testutil.JSONRequest(method, path, body), budgetAuth(userID)...)
}

func cleanupBudget(t *testing.T, id string) {
	t.Helper()
	t.Cleanup(func() {
		req := testutil.WithHeaders(
			testutil.WithURLParams(
				testutil.JSONRequest("DELETE", "/api/budgets/"+id, nil),
				"id", id,
			),
			budgetAuth(testUserID)...,
		)
		testutil.Call(t, testHandler.DeleteBudget, req)
	})
}

func cleanupWaiver(t *testing.T, id string) {
	t.Helper()
	t.Cleanup(func() {
		req := testutil.WithHeaders(
			testutil.WithURLParams(
				testutil.JSONRequest("DELETE", "/api/budgets/waivers/"+id, nil),
				"id", id,
			),
			budgetAuth(testUserID)...,
		)
		testutil.Call(t, testHandler.DeleteBudgetWaiver, req)
	})
}

func TestCreateAndListProjectBudget(t *testing.T) {
	projectID := dbfx.Project(t, "budget project")
	create := budgetReq("POST", "/api/budgets", map[string]any{
		"scope":             "project",
		"owner_id":          projectID,
		"limit_usd_ticks":   3_000_000_000_000,
		"soften_at_percent": 80,
		"over_limit":        "pause",
	}, testUserID)
	var created BudgetResponse
	testutil.Call(t, testHandler.CreateBudget, create).Want(http.StatusCreated).JSON(&created)
	cleanupBudget(t, created.ID)
	if created.Scope != "project" || created.OwnerID != projectID || created.LimitUsdTicks != 3_000_000_000_000 {
		t.Fatalf("create = %+v", created)
	}
	if created.CurrentPeriod == nil {
		t.Fatal("expected current_period after create")
	}
	if created.CurrentPeriod.State != "ok" || created.CurrentPeriod.SpentUsdTicks != 0 {
		t.Fatalf("period = %+v", created.CurrentPeriod)
	}

	listed := testutil.Call(t, testHandler.ListBudgets, budgetReq("GET", "/api/budgets", nil, testUserID)).
		Want(http.StatusOK).Map()
	rows, _ := listed["budgets"].([]any)
	found := false
	for _, raw := range rows {
		row, _ := raw.(map[string]any)
		if row["id"] == created.ID {
			found = true
			if row["current_period"] == nil {
				t.Fatal("list omitted current_period")
			}
		}
	}
	if !found {
		t.Fatalf("created budget missing from list: %#v", listed)
	}
}

func TestCreateBudgetUpsertsSameOwner(t *testing.T) {
	projectID := dbfx.Project(t, "budget upsert")
	body := map[string]any{
		"scope":           "project",
		"owner_id":        projectID,
		"limit_usd_ticks": 1_000_000_000_000,
		"over_limit":      "allow",
	}
	var first BudgetResponse
	testutil.Call(t, testHandler.CreateBudget, budgetReq("POST", "/api/budgets", body, testUserID)).
		Want(http.StatusCreated).JSON(&first)
	cleanupBudget(t, first.ID)

	body["limit_usd_ticks"] = 5_000_000_000_000
	var second BudgetResponse
	testutil.Call(t, testHandler.CreateBudget, budgetReq("POST", "/api/budgets", body, testUserID)).
		Want(http.StatusOK).JSON(&second)
	if second.ID != first.ID {
		t.Fatalf("second POST inserted %s instead of updating %s", second.ID, first.ID)
	}
	if second.LimitUsdTicks != 5_000_000_000_000 || second.OverLimit != "allow" {
		t.Fatalf("upsert = %+v", second)
	}

	listed := testutil.Call(t, testHandler.ListBudgets, budgetReq("GET", "/api/budgets", nil, testUserID)).
		Want(http.StatusOK).Map()
	count := 0
	for _, raw := range listed["budgets"].([]any) {
		row, _ := raw.(map[string]any)
		if row["owner_id"] == projectID && row["scope"] == "project" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one project budget after upsert, got %d", count)
	}
}

func TestCreateBudgetBackfillsUsage(t *testing.T) {
	runtimeID := dbfx.Runtime(t, "budget runtime")
	agentID := dbfx.Agent(t, "budget agent", runtimeID)
	projectID := dbfx.Project(t, "budget spend")
	taskID := dbfx.Task(t, agentID, testutil.Cols{
		"runtime_id":        runtimeID,
		"budget_project_id": projectID,
	})
	const ticks int64 = 2_000_000_000_000
	dbfx.Insert(t, "task_usage", testutil.Cols{
		"task_id":            taskID,
		"provider":           "anthropic",
		"model":              "budget-test-model",
		"input_tokens":       0,
		"output_tokens":      0,
		"cache_read_tokens":  0,
		"cache_write_tokens": 0,
		"cost_usd_ticks":     ticks,
	})

	var created BudgetResponse
	testutil.Call(t, testHandler.CreateBudget, budgetReq("POST", "/api/budgets", map[string]any{
		"scope":           "project",
		"owner_id":        projectID,
		"limit_usd_ticks": 3_000_000_000_000,
		"over_limit":      "pause",
	}, testUserID)).Want(http.StatusCreated).JSON(&created)
	cleanupBudget(t, created.ID)
	if created.CurrentPeriod == nil || created.CurrentPeriod.SpentUsdTicks != ticks {
		t.Fatalf("backfill period = %+v", created.CurrentPeriod)
	}
	if created.CurrentPeriod.State != "ok" {
		t.Fatalf("state = %q, want ok", created.CurrentPeriod.State)
	}
}

func TestSquadBudgetWithoutStampIsUnattributed(t *testing.T) {
	runtimeID := dbfx.Runtime(t, "squad budget runtime")
	leaderID := dbfx.Agent(t, "squad budget leader", runtimeID)
	squadID := dbfx.Squad(t, "budget squad", leaderID)
	var created BudgetResponse
	testutil.Call(t, testHandler.CreateBudget, budgetReq("POST", "/api/budgets", map[string]any{
		"scope":           "squad",
		"owner_id":        squadID,
		"limit_usd_ticks": 1_000_000_000_000,
		"over_limit":      "pause",
	}, testUserID)).Want(http.StatusCreated).JSON(&created)
	cleanupBudget(t, created.ID)
	if created.CurrentPeriod == nil || created.CurrentPeriod.State != "unattributed" {
		t.Fatalf("squad period = %+v", created.CurrentPeriod)
	}
	if created.CurrentPeriod.SpentUsdTicks != 0 {
		t.Fatalf("unattributed squad spent %d", created.CurrentPeriod.SpentUsdTicks)
	}
}

func TestPatchBudgetClearsSoften(t *testing.T) {
	projectID := dbfx.Project(t, "budget patch")
	var created BudgetResponse
	testutil.Call(t, testHandler.CreateBudget, budgetReq("POST", "/api/budgets", map[string]any{
		"scope":             "project",
		"owner_id":          projectID,
		"limit_usd_ticks":   1_000_000_000_000,
		"soften_at_percent": 80,
		"over_limit":        "pause",
	}, testUserID)).Want(http.StatusCreated).JSON(&created)
	cleanupBudget(t, created.ID)
	if created.SoftenAtPercent == nil || *created.SoftenAtPercent != 80 {
		t.Fatalf("create soften = %v", created.SoftenAtPercent)
	}

	req := testutil.WithHeaders(
		testutil.WithURLParams(
			testutil.JSONRequest("PATCH", "/api/budgets/"+created.ID, map[string]any{
				"soften_at_percent": nil,
			}),
			"id", created.ID,
		),
		budgetAuth(testUserID)...,
	)
	var patched BudgetResponse
	testutil.Call(t, testHandler.PatchBudget, req).Want(http.StatusOK).JSON(&patched)
	if patched.SoftenAtPercent != nil {
		t.Fatalf("expected null soften, got %v", *patched.SoftenAtPercent)
	}
}

func TestCreateBudgetRejectsBadInput(t *testing.T) {
	projectID := dbfx.Project(t, "budget bad input")
	testutil.Call(t, testHandler.CreateBudget, budgetReq("POST", "/api/budgets", map[string]any{
		"scope":           "project",
		"owner_id":        projectID,
		"limit_usd_ticks": 0,
		"over_limit":      "pause",
	}, testUserID)).Want(http.StatusBadRequest)

	testutil.Call(t, testHandler.CreateBudget, budgetReq("POST", "/api/budgets", map[string]any{
		"scope":           "project",
		"owner_id":        projectID,
		"limit_usd_ticks": 100,
		"over_limit":      "stop",
	}, testUserID)).Want(http.StatusBadRequest)
}

func TestMemberCanWriteProjectBudgetNotAgentOrWaiver(t *testing.T) {
	memberID := dbfx.User(t, "Budget Member", "budget-member@multica.ai")
	dbfx.Member(t, testWorkspaceID, memberID, "member")
	projectID := dbfx.Project(t, "member budget project")
	runtimeID := dbfx.Runtime(t, "member budget runtime")
	agentID := dbfx.Agent(t, "owner agent", runtimeID)

	var created BudgetResponse
	testutil.Call(t, testHandler.CreateBudget, budgetReq("POST", "/api/budgets", map[string]any{
		"scope":           "project",
		"owner_id":        projectID,
		"limit_usd_ticks": 1_000_000_000_000,
		"over_limit":      "allow",
	}, memberID)).Want(http.StatusCreated).JSON(&created)
	cleanupBudget(t, created.ID)

	testutil.Call(t, testHandler.CreateBudget, budgetReq("POST", "/api/budgets", map[string]any{
		"scope":           "agent",
		"owner_id":        agentID,
		"limit_usd_ticks": 1_000_000_000_000,
		"over_limit":      "pause",
	}, memberID)).Want(http.StatusForbidden)

	testutil.Call(t, testHandler.CreateBudgetWaiver, budgetReq("POST", "/api/budgets/waivers", map[string]any{
		"scope":    "project",
		"owner_id": projectID,
	}, memberID)).Want(http.StatusForbidden)
}

func TestOwnerCanCreateProjectWaiverAndMemberCannot(t *testing.T) {
	projectID := dbfx.Project(t, "waiver project")
	memberID := dbfx.User(t, "Waiver Member", "budget-waiver-member@multica.ai")
	dbfx.Member(t, testWorkspaceID, memberID, "member")

	var created BudgetWaiverResponse
	testutil.Call(t, testHandler.CreateBudgetWaiver, budgetReq("POST", "/api/budgets/waivers", map[string]any{
		"scope":    "project",
		"owner_id": projectID,
		"reason":   "launch week",
	}, testUserID)).Want(http.StatusCreated).JSON(&created)
	cleanupWaiver(t, created.ID)
	if created.Scope != "project" || created.OwnerID != projectID || created.Reason == nil || *created.Reason != "launch week" {
		t.Fatalf("waiver = %+v", created)
	}
	starts, err := time.Parse(time.RFC3339, created.StartsAt)
	if err != nil {
		t.Fatalf("starts_at: %v", err)
	}
	ends, err := time.Parse(time.RFC3339, created.EndsAt)
	if err != nil {
		t.Fatalf("ends_at: %v", err)
	}
	_, monthEnd := budgetpolicy.MonthWindow(time.Now())
	if !ends.Equal(monthEnd) {
		t.Fatalf("ends_at = %s, want month end %s", ends, monthEnd)
	}
	if !ends.After(starts) {
		t.Fatalf("window inverted: %s -> %s", starts, ends)
	}

	listed := testutil.Call(t, testHandler.ListBudgetWaivers, budgetReq("GET", "/api/budgets/waivers", nil, memberID)).
		Want(http.StatusOK).Map()
	found := false
	for _, raw := range listed["waivers"].([]any) {
		row, _ := raw.(map[string]any)
		if row["id"] == created.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("member could not list the waiver")
	}
}

func TestCreateWaiverRejectsAgentAndInvertedWindow(t *testing.T) {
	runtimeID := dbfx.Runtime(t, "waiver agent runtime")
	agentID := dbfx.Agent(t, "waiver agent", runtimeID)
	projectID := dbfx.Project(t, "waiver inverted")

	testutil.Call(t, testHandler.CreateBudgetWaiver, budgetReq("POST", "/api/budgets/waivers", map[string]any{
		"scope":    "agent",
		"owner_id": agentID,
	}, testUserID)).Want(http.StatusBadRequest)

	testutil.Call(t, testHandler.CreateBudgetWaiver, budgetReq("POST", "/api/budgets/waivers", map[string]any{
		"scope":     "project",
		"owner_id":  projectID,
		"starts_at": "2026-08-20T00:00:00Z",
		"ends_at":   "2026-08-10T00:00:00Z",
	}, testUserID)).Want(http.StatusBadRequest)
}

func TestCreateWaiverRejectsOverlap(t *testing.T) {
	projectID := dbfx.Project(t, "waiver overlap")
	var first BudgetWaiverResponse
	testutil.Call(t, testHandler.CreateBudgetWaiver, budgetReq("POST", "/api/budgets/waivers", map[string]any{
		"scope":    "project",
		"owner_id": projectID,
	}, testUserID)).Want(http.StatusCreated).JSON(&first)
	cleanupWaiver(t, first.ID)

	testutil.Call(t, testHandler.CreateBudgetWaiver, budgetReq("POST", "/api/budgets/waivers", map[string]any{
		"scope":    "project",
		"owner_id": projectID,
	}, testUserID)).Want(http.StatusConflict)
}

func TestProjectBudgetWaivedWhenWaiverActive(t *testing.T) {
	projectID := dbfx.Project(t, "waived bar")
	var budget BudgetResponse
	testutil.Call(t, testHandler.CreateBudget, budgetReq("POST", "/api/budgets", map[string]any{
		"scope":           "project",
		"owner_id":        projectID,
		"limit_usd_ticks": 1_000_000_000_000,
		"over_limit":      "pause",
	}, testUserID)).Want(http.StatusCreated).JSON(&budget)
	cleanupBudget(t, budget.ID)

	var waiver BudgetWaiverResponse
	testutil.Call(t, testHandler.CreateBudgetWaiver, budgetReq("POST", "/api/budgets/waivers", map[string]any{
		"scope":    "project",
		"owner_id": projectID,
	}, testUserID)).Want(http.StatusCreated).JSON(&waiver)
	cleanupWaiver(t, waiver.ID)

	req := testutil.WithHeaders(
		testutil.WithURLParams(
			testutil.JSONRequest("PATCH", "/api/budgets/"+budget.ID, map[string]any{
				"limit_usd_ticks": 1_000_000_000_001,
			}),
			"id", budget.ID,
		),
		budgetAuth(testUserID)...,
	)
	var patched BudgetResponse
	testutil.Call(t, testHandler.PatchBudget, req).Want(http.StatusOK).JSON(&patched)
	if patched.CurrentPeriod == nil || patched.CurrentPeriod.State != "waived" {
		t.Fatalf("expected waived state, got %+v", patched.CurrentPeriod)
	}
}

func TestDeleteBudget(t *testing.T) {
	projectID := dbfx.Project(t, "budget delete")
	var created BudgetResponse
	testutil.Call(t, testHandler.CreateBudget, budgetReq("POST", "/api/budgets", map[string]any{
		"scope":           "project",
		"owner_id":        projectID,
		"limit_usd_ticks": 1_000_000_000_000,
		"over_limit":      "pause",
	}, testUserID)).Want(http.StatusCreated).JSON(&created)

	req := testutil.WithHeaders(
		testutil.WithURLParams(
			testutil.JSONRequest("DELETE", "/api/budgets/"+created.ID, nil),
			"id", created.ID,
		),
		budgetAuth(testUserID)...,
	)
	testutil.Call(t, testHandler.DeleteBudget, req).Want(http.StatusNoContent)

	listed := testutil.Call(t, testHandler.ListBudgets, budgetReq("GET", "/api/budgets", nil, testUserID)).
		Want(http.StatusOK).Map()
	for _, raw := range listed["budgets"].([]any) {
		row, _ := raw.(map[string]any)
		if row["id"] == created.ID {
			t.Fatal("deleted budget still listed")
		}
	}
}

func TestCreateBudgetMissingOwner(t *testing.T) {
	testutil.Call(t, testHandler.CreateBudget, budgetReq("POST", "/api/budgets", map[string]any{
		"scope":           "project",
		"owner_id":        "00000000-0000-0000-0000-000000000001",
		"limit_usd_ticks": 1_000_000_000_000,
		"over_limit":      "pause",
	}, testUserID)).Want(http.StatusNotFound)
}

func TestSquadBudgetWriteRequiresCreatorOrAdmin(t *testing.T) {
	memberID := dbfx.User(t, "Squad Budget Member", "budget-squad-member@multica.ai")
	dbfx.Member(t, testWorkspaceID, memberID, "member")
	runtimeID := dbfx.Runtime(t, "foreign squad runtime")
	leaderID := dbfx.Agent(t, "foreign squad leader", runtimeID)
	squadID := dbfx.Squad(t, "foreign budget squad", leaderID)
	body := budgetReq("POST", "/api/budgets", map[string]any{
		"scope":           "squad",
		"owner_id":        squadID,
		"limit_usd_ticks": 1_000_000_000_000,
		"over_limit":      "pause",
	}, memberID)
	w := testutil.Call(t, testHandler.CreateBudget, body).Want(http.StatusForbidden)
	if !strings.Contains(w.Text(), "squad") {
		t.Fatalf("expected squad ACL message, got %s", w.Text())
	}
}
