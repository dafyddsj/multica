package handler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/testutil"
)

func TestEntityStatusCatalogAndProjectWrite(t *testing.T) {
	listReq := testutil.WithHeaders(
		testutil.JSONRequest("GET", "/api/entity-statuses?resource_type=project", nil),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	listed := testutil.Call(t, testHandler.ListEntityStatuses, listReq).Want(http.StatusOK).Map()
	statuses, _ := listed["statuses"].([]any)
	if len(statuses) < 5 {
		t.Fatalf("expected at least the 5 built-in project statuses, got %d: %#v", len(statuses), listed)
	}

	createReq := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/entity-statuses", map[string]any{
			"resource_type": "project",
			"name":          "Shipping",
			"category":      "in_progress",
			"color":         "#22c55e",
		}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	var created EntityStatusResponse
	testutil.Call(t, testHandler.CreateEntityStatus, createReq).Want(http.StatusCreated).JSON(&created)
	t.Cleanup(func() {
		req := testutil.WithHeaders(
			testutil.WithURLParams(
				testutil.JSONRequest("DELETE", "/api/entity-statuses/"+created.ID, nil),
				"id", created.ID,
			),
			"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
		)
		testutil.Call(t, testHandler.ArchiveEntityStatus, req)
	})
	if created.Key != "shipping" || created.Category != "in_progress" || created.IsSystem {
		t.Fatalf("create response = %+v", created)
	}

	projectReq := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/projects", map[string]any{
			"title":  "Catalog status project",
			"status": "shipping",
		}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	var project ProjectResponse
	testutil.Call(t, testHandler.CreateProject, projectReq).Want(http.StatusCreated).JSON(&project)
	t.Cleanup(func() {
		req := testutil.WithHeaders(
			testutil.WithURLParams(
				testutil.JSONRequest("DELETE", "/api/projects/"+project.ID, nil),
				"id", project.ID,
			),
			"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
		)
		testutil.Call(t, testHandler.DeleteProject, req)
	})
	if project.Status != "shipping" {
		t.Fatalf("project status = %q, want shipping", project.Status)
	}

	unknown := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/projects", map[string]any{
			"title":  "bad status",
			"status": "does_not_exist",
		}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	w := testutil.Call(t, testHandler.CreateProject, unknown).Want(http.StatusBadRequest)
	if body := w.Text(); !strings.Contains(body, "shipping") {
		t.Errorf("expected error to list catalog keys including shipping, got: %s", body)
	}
}

func TestEntityStatusInitiativeCatalog(t *testing.T) {
	createReq := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/entity-statuses", map[string]any{
			"resource_type": "initiative",
			"name":          "Exploring",
			"category":      "planned",
			"color":         "#6b7280",
		}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	var created EntityStatusResponse
	testutil.Call(t, testHandler.CreateEntityStatus, createReq).Want(http.StatusCreated).JSON(&created)
	t.Cleanup(func() {
		req := testutil.WithHeaders(
			testutil.WithURLParams(
				testutil.JSONRequest("DELETE", "/api/entity-statuses/"+created.ID, nil),
				"id", created.ID,
			),
			"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
		)
		testutil.Call(t, testHandler.ArchiveEntityStatus, req)
	})

	initReq := testutil.WithHeaders(
		testutil.JSONRequest("POST", "/api/initiatives", map[string]any{
			"title":  "Catalog status initiative",
			"status": "exploring",
		}),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	var initiative InitiativeResponse
	testutil.Call(t, testHandler.CreateInitiative, initReq).Want(http.StatusCreated).JSON(&initiative)
	t.Cleanup(func() {
		req := testutil.WithHeaders(
			testutil.WithURLParams(
				testutil.JSONRequest("DELETE", "/api/initiatives/"+initiative.ID, nil),
				"id", initiative.ID,
			),
			"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
		)
		testutil.Call(t, testHandler.DeleteInitiative, req)
	})
	if initiative.Status != "exploring" {
		t.Fatalf("initiative status = %q, want exploring", initiative.Status)
	}
}

func TestEntityStatusBuiltInRenameAndArchiveGuard(t *testing.T) {
	listReq := testutil.WithHeaders(
		testutil.JSONRequest("GET", "/api/entity-statuses?resource_type=project&include_archived=true", nil),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	listed := testutil.Call(t, testHandler.ListEntityStatuses, listReq).Want(http.StatusOK).Map()
	var plannedID string
	for _, raw := range listed["statuses"].([]any) {
		row, _ := raw.(map[string]any)
		if row["key"] == "planned" {
			plannedID, _ = row["id"].(string)
			break
		}
	}
	if plannedID == "" {
		t.Fatal("planned built-in missing from catalog")
	}

	rename := testutil.WithHeaders(
		testutil.WithURLParams(
			testutil.JSONRequest("PATCH", "/api/entity-statuses/"+plannedID, map[string]any{
				"name": "Not started",
			}),
			"id", plannedID,
		),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	var updated EntityStatusResponse
	testutil.Call(t, testHandler.UpdateEntityStatus, rename).Want(http.StatusOK).JSON(&updated)
	t.Cleanup(func() {
		req := testutil.WithHeaders(
			testutil.WithURLParams(
				testutil.JSONRequest("PATCH", "/api/entity-statuses/"+plannedID, map[string]any{
					"name": "Planned",
				}),
				"id", plannedID,
			),
			"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
		)
		testutil.Call(t, testHandler.UpdateEntityStatus, req)
	})
	if updated.Name != "Not started" || !updated.IsSystem {
		t.Fatalf("rename built-in = %+v", updated)
	}

	archive := testutil.WithHeaders(
		testutil.WithURLParams(
			testutil.JSONRequest("DELETE", "/api/entity-statuses/"+plannedID, nil),
			"id", plannedID,
		),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	testutil.Call(t, testHandler.ArchiveEntityStatus, archive).Want(http.StatusForbidden)
}

func TestEntityStatusRequiresResourceType(t *testing.T) {
	req := testutil.WithHeaders(
		testutil.JSONRequest("GET", "/api/entity-statuses", nil),
		"X-User-ID", testUserID, "X-Workspace-ID", testWorkspaceID,
	)
	testutil.Call(t, testHandler.ListEntityStatuses, req).Want(http.StatusBadRequest)
}
