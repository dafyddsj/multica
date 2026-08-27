package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/testutil"
)

func TestCreateInitiativeRequiresTitle(t *testing.T) {
	req := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/initiatives", map[string]any{}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	testutil.Call(t, testHandler.CreateInitiative, req).Want(http.StatusBadRequest)
}

func TestCreateInitiativeInvalidStatusReturns400(t *testing.T) {
	req := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/initiatives", map[string]any{
			"title":  "bad status",
			"status": "active",
		}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	w := testutil.Call(t, testHandler.CreateInitiative, req).Want(http.StatusBadRequest)
	if body := w.Text(); !strings.Contains(body, "planned") {
		t.Errorf("expected error to list valid statuses, got: %s", body)
	}
}

func TestInitiativeCRUDAndProjectAttach(t *testing.T) {
	createReq := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/initiatives", map[string]any{
			"title":       "Mobile app",
			"description": "The iOS product",
			"status":      "in_progress",
			"priority":    "high",
			"icon":        "📱",
		}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	var created InitiativeResponse
	testutil.Call(t, testHandler.CreateInitiative, createReq).Want(http.StatusCreated).JSON(&created)
	if created.Title != "Mobile app" || created.Status != "in_progress" || created.Priority != "high" {
		t.Fatalf("create response = %+v", created)
	}
	if created.ProjectCount != 0 || created.IssueCount != 0 {
		t.Errorf("empty initiative should have zero counts, got projects=%d issues=%d", created.ProjectCount, created.IssueCount)
	}

	getReq := testutil.WithHeaders(
		testutil.WithURLParams(
			testutil.JSONRequest("GET", "/api/initiatives/"+created.ID, nil),
			"id", created.ID,
		),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	var got InitiativeResponse
	testutil.Call(t, testHandler.GetInitiative, getReq).Want(http.StatusOK).JSON(&got)
	if got.ID != created.ID || got.Title != "Mobile app" {
		t.Fatalf("get = %+v", got)
	}

	projectID := dbfx.Project(t, "Q3 launch", testutil.Cols{"initiative_id": created.ID})
	issueID := dbfx.Issue(t, "Ship onboarding", testutil.Cols{"project_id": projectID, "status": "done"})
	_ = issueID

	testutil.Call(t, testHandler.GetInitiative, getReq).Want(http.StatusOK).JSON(&got)
	if got.ProjectCount != 1 {
		t.Errorf("project_count = %d, want 1", got.ProjectCount)
	}
	if got.IssueCount != 1 || got.DoneCount != 1 {
		t.Errorf("issue_count=%d done_count=%d, want 1/1", got.IssueCount, got.DoneCount)
	}

	listReq := testutil.WithHeaders(
		testutil.JSONRequest("GET", "/api/projects?initiative_id="+created.ID, nil),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	listed := testutil.Call(t, testHandler.ListProjects, listReq).Want(http.StatusOK).Map()
	projects, _ := listed["projects"].([]any)
	if len(projects) != 1 {
		t.Fatalf("list projects by initiative: got %d, body=%v", len(projects), listed)
	}

	updateReq := testutil.WithHeaders(
		testutil.WithURLParams(
			testutil.JSONRequest("PUT", "/api/initiatives/"+created.ID, map[string]any{
				"title":  "Mobile",
				"status": "paused",
			}),
			"id", created.ID,
		),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	var updated InitiativeResponse
	testutil.Call(t, testHandler.UpdateInitiative, updateReq).Want(http.StatusOK).JSON(&updated)
	if updated.Title != "Mobile" || updated.Status != "paused" {
		t.Fatalf("update = %+v", updated)
	}

	deleteReq := testutil.WithHeaders(
		testutil.WithURLParams(
			testutil.JSONRequest("DELETE", "/api/initiatives/"+created.ID, nil),
			"id", created.ID,
		),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	testutil.Call(t, testHandler.DeleteInitiative, deleteReq).Want(http.StatusNoContent)

	testutil.Call(t, testHandler.GetInitiative, getReq).Want(http.StatusNotFound)

	if dbfx.Count(t, `SELECT count(*) FROM project WHERE id = $1 AND initiative_id IS NULL`, projectID) != 1 {
		t.Error("delete should detach projects and leave the project row")
	}
}

func TestCreateProjectWithInitiative(t *testing.T) {
	initiativeID := dbfx.Initiative(t, "Platform")
	req := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/projects", map[string]any{
			"title":         "Auth rewrite",
			"initiative_id": initiativeID,
		}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	var project ProjectResponse
	testutil.Call(t, testHandler.CreateProject, req).Want(http.StatusCreated).JSON(&project)
	if project.InitiativeID == nil || *project.InitiativeID != initiativeID {
		t.Fatalf("initiative_id = %v, want %s", project.InitiativeID, initiativeID)
	}

	clearReq := testutil.WithHeaders(
		testutil.WithURLParams(
			testutil.JSONRequest("PUT", "/api/projects/"+project.ID, map[string]any{
				"initiative_id": nil,
			}),
			"id", project.ID,
		),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	var cleared ProjectResponse
	testutil.Call(t, testHandler.UpdateProject, clearReq).Want(http.StatusOK).JSON(&cleared)
	if cleared.InitiativeID != nil {
		t.Errorf("clear initiative_id left %v", cleared.InitiativeID)
	}

	missing := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/projects", map[string]any{
			"title":         "orphan",
			"initiative_id": "00000000-0000-0000-0000-000000000001",
		}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	testutil.Call(t, testHandler.CreateProject, missing).Want(http.StatusBadRequest)
}

func TestSearchInitiatives(t *testing.T) {
	_ = dbfx.Initiative(t, "zzsearchwidget")
	req := testutil.WithHeaders(
		testutil.JSONRequest("GET", "/api/initiatives/search?q=zzsearchwidget", nil),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	body := testutil.Call(t, testHandler.SearchInitiatives, req).Want(http.StatusOK).Map()
	rows, _ := body["initiatives"].([]any)
	if len(rows) == 0 {
		t.Fatalf("expected search hit, got %v", body)
	}
}

func TestMemberCannotDeleteInitiative(t *testing.T) {
	initiativeID := dbfx.Initiative(t, "locked")
	memberID := dbfx.User(t, "Member", "initiative-member@multica.ai")
	dbfx.Member(t, testWorkspaceID, memberID, "member")
	req := testutil.WithHeaders(
		testutil.WithURLParams(
			testutil.JSONRequest("DELETE", "/api/initiatives/"+initiativeID, nil),
			"id", initiativeID,
		),
		"X-User-ID", memberID, "X-Workspace-ID", testWorkspaceID,
	)
	testutil.Call(t, testHandler.DeleteInitiative, req).Want(http.StatusForbidden)
}

func TestInitiativePinOptIn(t *testing.T) {
	initiativeID := dbfx.Initiative(t, "Pinned product")
	createPin := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/pins", map[string]any{
			"item_type": "initiative",
			"item_id":   initiativeID,
		}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	testutil.Call(t, testHandler.CreatePin, createPin).Want(http.StatusCreated)

	hidden := testutil.Call(t, testHandler.ListPins, testutil.WithHeaders(
		testutil.JSONRequest("GET", "/api/pins?include=view", nil),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)).Want(http.StatusOK)
	var hiddenPins []PinnedItemResponse
	if err := json.Unmarshal(hidden.Body.Bytes(), &hiddenPins); err != nil {
		t.Fatal(err)
	}
	for _, p := range hiddenPins {
		if p.ItemType == "initiative" && p.ItemID == initiativeID {
			t.Fatal("old include=view clients must not receive initiative pins")
		}
	}

	shown := testutil.Call(t, testHandler.ListPins, testutil.WithHeaders(
		testutil.JSONRequest("GET", "/api/pins?include=view,initiative", nil),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)).Want(http.StatusOK)
	var shownPins []PinnedItemResponse
	if err := json.Unmarshal(shown.Body.Bytes(), &shownPins); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range shownPins {
		if p.ItemType == "initiative" && p.ItemID == initiativeID {
			found = true
		}
	}
	if !found {
		t.Fatal("include=initiative should return the pin")
	}
}

func TestInitiativeIssuePrefixOverridesWorkspace(t *testing.T) {
	createReq := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/initiatives", map[string]any{
			"title":        "Prefix override",
			"issue_prefix": "mob",
		}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	var initiative InitiativeResponse
	testutil.Call(t, testHandler.CreateInitiative, createReq).Want(http.StatusCreated).JSON(&initiative)
	if initiative.IssuePrefix == nil || *initiative.IssuePrefix != "MOB" {
		t.Fatalf("create issue_prefix = %v, want MOB", initiative.IssuePrefix)
	}

	badReq := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/initiatives", map[string]any{
			"title":        "bad prefix",
			"issue_prefix": "not-valid!",
		}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	testutil.Call(t, testHandler.CreateInitiative, badReq).Want(http.StatusBadRequest)

	projectID := dbfx.Project(t, "prefix project", testutil.Cols{"initiative_id": initiative.ID})
	issueReq := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/issues", map[string]any{
			"title":      "Prefixed issue",
			"project_id": projectID,
		}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	var created IssueResponse
	testutil.Call(t, testHandler.CreateIssue, issueReq).Want(http.StatusCreated).JSON(&created)
	if !strings.HasPrefix(created.Identifier, "MOB-") {
		t.Fatalf("identifier = %q, want MOB-*", created.Identifier)
	}

	getReq := testutil.WithHeaders(
		testutil.WithURLParams(
			testutil.JSONRequest("GET", "/api/issues/"+created.Identifier, nil),
			"id", created.Identifier,
		),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	var got IssueResponse
	testutil.Call(t, testHandler.GetIssue, getReq).Want(http.StatusOK).JSON(&got)
	if got.ID != created.ID {
		t.Fatalf("GetIssue(%q) = %s, want %s", created.Identifier, got.ID, created.ID)
	}

	foreign := "ZZZ-" + strings.TrimPrefix(created.Identifier, "MOB-")
	foreignReq := testutil.WithHeaders(
		testutil.WithURLParams(
			testutil.JSONRequest("GET", "/api/issues/"+foreign, nil),
			"id", foreign,
		),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	testutil.Call(t, testHandler.GetIssue, foreignReq).Want(http.StatusNotFound)

	var workspacePrefix string
	dbfx.QueryRow(t, `SELECT issue_prefix FROM workspace WHERE id = $1`, testWorkspaceID).Scan(&workspacePrefix)
	if workspacePrefix != "" && workspacePrefix != "MOB" {
		workspaceIdent := workspacePrefix + "-" + strings.TrimPrefix(created.Identifier, "MOB-")
		wsReq := testutil.WithHeaders(
			testutil.WithURLParams(
				testutil.JSONRequest("GET", "/api/issues/"+workspaceIdent, nil),
				"id", workspaceIdent,
			),
			"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
		)
		var viaWorkspace IssueResponse
		testutil.Call(t, testHandler.GetIssue, wsReq).Want(http.StatusOK).JSON(&viaWorkspace)
		if viaWorkspace.ID != created.ID {
			t.Fatalf("GetIssue(%q) = %s, want %s", workspaceIdent, viaWorkspace.ID, created.ID)
		}
	}
}
