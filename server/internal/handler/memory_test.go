package handler

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/featureflags"
	"github.com/multica-ai/multica/server/internal/memory"
	"github.com/multica-ai/multica/server/internal/testutil"
)

func enableMemoryForTest(t *testing.T) {
	t.Helper()
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}
	withFeatureFlag(t, testHandler, featureflags.MemoryV1, true)
	dbfx.Exec(t, `UPDATE workspace SET settings = COALESCE(settings, '{}'::jsonb) || '{"memory_enabled": true}'::jsonb WHERE id = $1`, testWorkspaceID)
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `UPDATE workspace SET settings = settings - 'memory_enabled' WHERE id = $1`, testWorkspaceID)
	})
}

func TestListMemory_NotEnabled(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	req := newRequest(http.MethodGet, "/api/memory?scope=workspace&owner_id="+testWorkspaceID, nil)
	testutil.Call(t, testHandler.ListMemory, req).Want(http.StatusNotFound)
}

func TestListMemory_FlagOnLabsOff(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	withFeatureFlag(t, testHandler, featureflags.MemoryV1, true)
	req := newRequest(http.MethodGet, "/api/memory?scope=workspace&owner_id="+testWorkspaceID, nil)
	testutil.Call(t, testHandler.ListMemory, req).Want(http.StatusNotFound)
}

func TestCreateListGetForgetMemory(t *testing.T) {
	enableMemoryForTest(t)

	var created MemoryEntryResponse
	testutil.Call(t, testHandler.CreateMemory, newRequest(http.MethodPost, "/api/memory", map[string]any{
		"scope":    memory.ScopeWorkspace,
		"owner_id": testWorkspaceID,
		"body":     "Deploy on Fridays is fine",
		"kind":     "preference",
	})).Want(http.StatusCreated).JSON(&created)
	if created.Body != "Deploy on Fridays is fine" || created.Kind != "preference" {
		t.Fatalf("create = %#v", created)
	}
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM memory_entry WHERE id = $1`, created.ID)
	})

	var listed MemoryListResponse
	testutil.Call(t, testHandler.ListMemory, newRequest(http.MethodGet, "/api/memory?scope=workspace&owner_id="+testWorkspaceID, nil)).
		Want(http.StatusOK).JSON(&listed)
	if listed.Total < 1 {
		t.Fatalf("list total = %d, want at least 1", listed.Total)
	}

	var got MemoryEntryResponse
	req := withURLParam(newRequest(http.MethodGet, "/api/memory/"+created.ID, nil), "id", created.ID)
	testutil.Call(t, testHandler.GetMemory, req).Want(http.StatusOK).JSON(&got)
	if got.ID != created.ID {
		t.Fatalf("get id = %s, want %s", got.ID, created.ID)
	}

	var updated MemoryEntryResponse
	testutil.Call(t, testHandler.UpdateMemory, withURLParam(newRequest(http.MethodPatch, "/api/memory/"+created.ID, map[string]any{
		"body": "Deploy on Fridays is still fine",
		"kind": "fact",
	}), "id", created.ID)).Want(http.StatusOK).JSON(&updated)
	if updated.Body != "Deploy on Fridays is still fine" || updated.Kind != "fact" {
		t.Fatalf("update = %#v", updated)
	}

	forget := withURLParam(newRequest(http.MethodDelete, "/api/memory/"+created.ID, nil), "id", created.ID)
	testutil.Call(t, testHandler.DeleteMemory, forget).Want(http.StatusNoContent)

	testutil.Call(t, testHandler.GetMemory, withURLParam(newRequest(http.MethodGet, "/api/memory/"+created.ID, nil), "id", created.ID)).
		Want(http.StatusNotFound)
}

func TestCreateMemory_WorkspaceWriteDeniedForMember(t *testing.T) {
	enableMemoryForTest(t)
	memberID := createPermissionTestMember(t, "memory-member@multica.ai")
	req := newRequest(http.MethodPost, "/api/memory", map[string]any{
		"scope":    memory.ScopeWorkspace,
		"owner_id": testWorkspaceID,
		"body":     "member should not write workspace memory",
	})
	req.Header.Set("X-User-ID", memberID)
	testutil.Call(t, testHandler.CreateMemory, req).Want(http.StatusForbidden)
}

func TestCreateMemory_IssueScopeAllowsMember(t *testing.T) {
	enableMemoryForTest(t)
	issueID := dbfx.Issue(t, "memory issue")
	memberID := createPermissionTestMember(t, "memory-issue-member@multica.ai")
	req := newRequest(http.MethodPost, "/api/memory", map[string]any{
		"scope":    memory.ScopeIssue,
		"owner_id": issueID,
		"body":     "this issue prefers rebase",
	})
	req.Header.Set("X-User-ID", memberID)
	var created MemoryEntryResponse
	testutil.Call(t, testHandler.CreateMemory, req).Want(http.StatusCreated).JSON(&created)
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM memory_entry WHERE id = $1`, created.ID)
	})
}

func TestUserMemoryIsPrivate(t *testing.T) {
	enableMemoryForTest(t)
	otherID := createPermissionTestMember(t, "memory-other@multica.ai")
	var created MemoryEntryResponse
	testutil.Call(t, testHandler.CreateMemory, newRequest(http.MethodPost, "/api/memory", map[string]any{
		"scope":    memory.ScopeUser,
		"owner_id": testUserID,
		"body":     "private preference",
	})).Want(http.StatusCreated).JSON(&created)
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM memory_entry WHERE id = $1`, created.ID)
	})

	req := withURLParam(newRequest(http.MethodGet, "/api/memory/"+created.ID, nil), "id", created.ID)
	req.Header.Set("X-User-ID", otherID)
	testutil.Call(t, testHandler.GetMemory, req).Want(http.StatusForbidden)
}

func TestCreateMemory_BankCap(t *testing.T) {
	enableMemoryForTest(t)
	issueID := dbfx.Issue(t, "memory cap")
	dbfx.Exec(t, `
		INSERT INTO memory_entry (workspace_id, scope, owner_id, body, kind)
		SELECT $1::uuid, 'issue', $2::uuid, 'seed ' || g, 'fact'
		FROM generate_series(1, 200) AS g
	`, testWorkspaceID, issueID)
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM memory_entry WHERE scope = 'issue' AND owner_id = $1`, issueID)
	})
	testutil.Call(t, testHandler.CreateMemory, newRequest(http.MethodPost, "/api/memory", map[string]any{
		"scope":    memory.ScopeIssue,
		"owner_id": issueID,
		"body":     "one more",
	})).Want(http.StatusBadRequest)
}

func TestCreateMemory_MalformedBody(t *testing.T) {
	enableMemoryForTest(t)
	testutil.Call(t, testHandler.CreateMemory, newRequest(http.MethodPost, "/api/memory", map[string]any{
		"scope":    memory.ScopeWorkspace,
		"owner_id": testWorkspaceID,
		"body":     "",
	})).Want(http.StatusBadRequest)

	testutil.Call(t, testHandler.CreateMemory, newRequest(http.MethodPost, "/api/memory", map[string]any{
		"scope":    "not-a-scope",
		"owner_id": testWorkspaceID,
		"body":     "x",
	})).Want(http.StatusBadRequest)

	tooLong := strings.Repeat("x", memory.MaxBodyRunes+1)
	testutil.Call(t, testHandler.CreateMemory, newRequest(http.MethodPost, "/api/memory", map[string]any{
		"scope":    memory.ScopeWorkspace,
		"owner_id": testWorkspaceID,
		"body":     tooLong,
	})).Want(http.StatusBadRequest)

	huge := strings.Repeat("x", memory.MaxProvenanceBytes+1)
	testutil.Call(t, testHandler.CreateMemory, newRequest(http.MethodPost, "/api/memory", map[string]any{
		"scope":      memory.ScopeWorkspace,
		"owner_id":   testWorkspaceID,
		"body":       "ok",
		"provenance": map[string]any{"blob": huge},
	})).Want(http.StatusBadRequest)
}

func TestRecallMemory_ReturnsHits(t *testing.T) {
	enableMemoryForTest(t)
	var created MemoryEntryResponse
	testutil.Call(t, testHandler.CreateMemory, newRequest(http.MethodPost, "/api/memory", map[string]any{
		"scope":    memory.ScopeWorkspace,
		"owner_id": testWorkspaceID,
		"body":     "workspace ships on Thursday",
	})).Want(http.StatusCreated).JSON(&created)
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM memory_entry WHERE id = $1`, created.ID)
	})

	var recalled MemoryRecallResponse
	testutil.Call(t, testHandler.RecallMemory, newRequest(http.MethodGet, "/api/memory/recall?q=Thursday", nil)).
		Want(http.StatusOK).JSON(&recalled)
	found := false
	for _, hit := range recalled.Hits {
		if hit.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("recall missed created entry: %#v", recalled.Hits)
	}
}
