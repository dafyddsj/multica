package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/integrations/agentmail"
	"github.com/multica-ai/multica/server/internal/testutil"
	"github.com/multica-ai/multica/server/internal/util/secretbox"
)

func withAgentMail(t *testing.T) *agentmail.Service {
	t.Helper()
	if testHandler == nil || testPool == nil {
		t.Skip("database not available")
	}
	ensureAgentMailTables(t)

	box, err := secretbox.New(make([]byte, secretbox.KeySize))
	if err != nil {
		t.Fatalf("secretbox.New: %v", err)
	}
	svc, err := agentmail.NewMemory(agentmail.Config{
		Box:                 box,
		HostedOrgKey:        "am_hosted_org_test_secret",
		WorkspaceInboxLimit: 5,
	}, testHandler.Queries)
	if err != nil {
		t.Fatalf("NewMemory: %v", err)
	}
	prev := testHandler.AgentMail
	testHandler.AgentMail = svc
	t.Cleanup(func() {
		testHandler.AgentMail = prev
		_, _ = testPool.Exec(context.Background(), `DELETE FROM agentmail_purge WHERE workspace_id = $1`, testWorkspaceID)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM agentmail_inbox WHERE workspace_id = $1`, testWorkspaceID)
		_, _ = testPool.Exec(context.Background(), `DELETE FROM agentmail_connection WHERE workspace_id = $1`, testWorkspaceID)
	})
	return svc
}

func ensureAgentMailTables(t *testing.T) {
	t.Helper()
	var exists bool
	if err := testPool.QueryRow(context.Background(), `
SELECT EXISTS (
  SELECT 1 FROM information_schema.tables
  WHERE table_schema = 'public' AND table_name = 'agentmail_connection'
)`).Scan(&exists); err != nil {
		t.Fatalf("check agentmail tables: %v", err)
	}
	if !exists {
		t.Skip("agentmail migrations are not applied")
	}
}

func agentMailWorkspaceReq(method string, body any) *http.Request {
	req := newRequest(method, "/api/workspaces/"+testWorkspaceID+"/agentmail", body)
	return withURLParam(req, "id", testWorkspaceID)
}

func agentMailAgentReq(method, agentID string, body any) *http.Request {
	req := newRequest(method, "/api/agents/"+agentID+"/agentmail", body)
	return withURLParam(req, "id", agentID)
}

func TestGetAgentMailUnconfigured(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	prev := testHandler.AgentMail
	testHandler.AgentMail = nil
	t.Cleanup(func() { testHandler.AgentMail = prev })

	var resp AgentMailWorkspaceResponse
	testutil.Call(t, testHandler.GetAgentMail, agentMailWorkspaceReq(http.MethodGet, nil)).
		Want(http.StatusOK).JSON(&resp)
	if resp.Available || resp.Connected || resp.CanManage {
		t.Fatalf("unconfigured GET = %+v", resp)
	}
	if resp.Inboxes == nil {
		t.Fatal("inboxes must be an empty list, not null")
	}
}

func TestAgentMailConnectGrantClaimAndRevoke(t *testing.T) {
	withAgentMail(t)

	var connected AgentMailWorkspaceResponse
	testutil.Call(t, testHandler.ConnectAgentMail, agentMailWorkspaceReq(http.MethodPost, map[string]string{
		"mode": "hosted",
	})).Want(http.StatusOK).JSON(&connected)
	if !connected.Connected || connected.Source != "hosted" || connected.State != "active" {
		t.Fatalf("connect = %+v", connected)
	}

	var listed AgentMailWorkspaceResponse
	testutil.Call(t, testHandler.GetAgentMail, agentMailWorkspaceReq(http.MethodGet, nil)).
		Want(http.StatusOK).JSON(&listed)
	if !listed.Available || !listed.HostedAvailable || !listed.Connected || !listed.CanManage {
		t.Fatalf("GET workspace = %+v", listed)
	}

	runtimeID := dbfx.Runtime(t, "agentmail grant runtime")
	agentID := dbfx.Agent(t, "Ada Mail", runtimeID)

	testutil.Call(t, testHandler.GrantAgentMailInbox, agentMailAgentReq(http.MethodPut, agentID, map[string]any{})).
		Want(http.StatusBadRequest)

	var inbox AgentMailInboxResponse
	testutil.Call(t, testHandler.GrantAgentMailInbox, agentMailAgentReq(http.MethodPut, agentID, map[string]any{
		"username": "ada",
		"domain":   "agentmail.to",
	})).
		Want(http.StatusOK).JSON(&inbox)
	if !inbox.Enabled || inbox.Address != "ada@agentmail.to" || inbox.AgentID != agentID {
		t.Fatalf("grant = %+v", inbox)
	}

	var got AgentMailInboxResponse
	testutil.Call(t, testHandler.GetAgentMailInbox, agentMailAgentReq(http.MethodGet, agentID, nil)).
		Want(http.StatusOK).JSON(&got)
	if got.Address != inbox.Address || !got.Enabled {
		t.Fatalf("GET inbox = %+v", got)
	}

	testutil.Call(t, testHandler.GetAgentMail, agentMailWorkspaceReq(http.MethodGet, nil)).
		Want(http.StatusOK).JSON(&listed)
	if len(listed.Inboxes) != 1 || listed.Inboxes[0].Address != inbox.Address {
		t.Fatalf("roster = %+v", listed.Inboxes)
	}

	var domains AgentMailDomainListResponse
	testutil.Call(t, testHandler.ListAgentMailDomains, agentMailWorkspaceReq(http.MethodGet, nil)).
		Want(http.StatusOK).JSON(&domains)
	if len(domains.Domains) == 0 || domains.Domains[0] != "agentmail.to" {
		t.Fatalf("domains = %+v", domains)
	}

	var mailbox AgentMailMailboxResponse
	mailReq := newRequest(http.MethodGet, "/api/agents/"+agentID+"/agentmail/mailbox?section=inbox", nil)
	mailReq = withURLParam(mailReq, "id", agentID)
	testutil.Call(t, testHandler.ListAgentMailMailbox, mailReq).Want(http.StatusOK).JSON(&mailbox)
	if mailbox.Items == nil {
		t.Fatal("mailbox items must be an empty list, not null")
	}

	var folders AgentMailFolderListResponse
	folderReq := newRequest(http.MethodGet, "/api/agents/"+agentID+"/agentmail/folders", nil)
	folderReq = withURLParam(folderReq, "id", agentID)
	testutil.Call(t, testHandler.ListAgentMailFolders, folderReq).Want(http.StatusOK).JSON(&folders)
	if folders.Folders == nil {
		t.Fatal("folders must be an empty list, not null")
	}

	testutil.Call(t, testHandler.RevokeAgentMailInbox, agentMailAgentReq(http.MethodDelete, agentID, nil)).
		Want(http.StatusNoContent)
	var revoked AgentMailInboxResponse
	testutil.Call(t, testHandler.GetAgentMailInbox, agentMailAgentReq(http.MethodGet, agentID, nil)).
		Want(http.StatusOK).JSON(&revoked)
	if revoked.Enabled || revoked.Address != "" || revoked.State != "disabled" {
		t.Fatalf("revoked inbox still enabled: %+v", revoked)
	}
}

func TestConnectAgentMailModeConflict(t *testing.T) {
	withAgentMail(t)

	testutil.Call(t, testHandler.ConnectAgentMail, agentMailWorkspaceReq(http.MethodPost, map[string]string{
		"mode":    "bring_your_own",
		"org_key": "am_byo_org_key",
	})).Want(http.StatusOK)

	w := testutil.Call(t, testHandler.ConnectAgentMail, agentMailWorkspaceReq(http.MethodPost, map[string]string{
		"mode": "hosted",
	})).Want(http.StatusConflict)
	if !strings.Contains(w.Body.String(), "disconnect before switching") {
		t.Fatalf("conflict body = %s", w.Body.String())
	}
}

func TestAgentMailDeniesAgentActorAndPlainMemberInboxRead(t *testing.T) {
	withAgentMail(t)

	testutil.Call(t, testHandler.ConnectAgentMail, agentMailWorkspaceReq(http.MethodPost, map[string]string{
		"mode": "hosted",
	})).Want(http.StatusOK)

	runtimeID := dbfx.Runtime(t, "agentmail deny runtime")
	agentID := dbfx.Agent(t, "Denied Mail", runtimeID)
	testutil.Call(t, testHandler.GrantAgentMailInbox, agentMailAgentReq(http.MethodPut, agentID, map[string]any{
		"username": "denied",
	})).
		Want(http.StatusOK)

	req := agentMailAgentReq(http.MethodPut, agentID, map[string]any{})
	req.Header.Set("X-Actor-Source", "task_token")
	req.Header.Set("X-Agent-ID", agentID)
	testutil.Call(t, testHandler.GrantAgentMailInbox, req).Want(http.StatusForbidden)

	memberID := dbfx.User(t, "AgentMail Member", "agentmail-member-"+agentID[:8]+"@example.com")
	dbfx.Member(t, testWorkspaceID, memberID, "member")
	memberReq := agentMailAgentReq(http.MethodGet, agentID, nil)
	memberReq.Header.Set("X-User-ID", memberID)
	testutil.Call(t, testHandler.GetAgentMailInbox, memberReq).Want(http.StatusForbidden)

	rosterReq := agentMailWorkspaceReq(http.MethodGet, nil)
	rosterReq.Header.Set("X-User-ID", memberID)
	var roster AgentMailWorkspaceResponse
	testutil.Call(t, testHandler.GetAgentMail, rosterReq).Want(http.StatusOK).JSON(&roster)
	if roster.CanManage {
		t.Fatal("member must not manage workspace agentmail")
	}
	if len(roster.Inboxes) != 1 || roster.Inboxes[0].Address == "" {
		t.Fatalf("member must see roster address, got %+v", roster.Inboxes)
	}
}

func TestDeleteWorkspace_SweepsAgentMailKeepsPurge(t *testing.T) {
	withAgentMail(t)

	const slug = "handler-tests-agentmail-delete"
	_, _ = testPool.Exec(context.Background(), `DELETE FROM workspace WHERE slug = $1`, slug)
	wsID := dbfx.Insert(t, "workspace", testutil.Cols{
		"name":        "AgentMail Delete",
		"slug":        slug,
		"description": "SweepWorkspace on delete",
	})
	dbfx.Exec(t, `INSERT INTO member (workspace_id, user_id, role) VALUES ($1, $2, 'owner')`, wsID, testUserID)

	req := newRequest(http.MethodPost, "/api/workspaces/"+wsID+"/agentmail", map[string]string{"mode": "hosted"})
	req = withURLParam(req, "id", wsID)
	testutil.Call(t, testHandler.ConnectAgentMail, req).Want(http.StatusOK)

	runtimeID := dbfx.Insert(t, "agent_runtime", testutil.Cols{
		"workspace_id": wsID,
		"name":         "AgentMail delete runtime",
		"runtime_mode": "cloud",
		"provider":     "delete-test",
		"status":       "offline",
		"device_info":  "",
		"metadata":     testutil.Raw("'{}'::jsonb"),
		"owner_id":     testUserID,
	})
	agentID := dbfx.Insert(t, "agent", testutil.Cols{
		"workspace_id":   wsID,
		"name":           "AgentMail delete agent",
		"runtime_mode":   "cloud",
		"runtime_config": testutil.Raw("'{}'::jsonb"),
		"runtime_id":     runtimeID,
		"owner_id":       testUserID,
	})
	grant := newRequest(http.MethodPut, "/api/agents/"+agentID+"/agentmail", map[string]any{
		"username": "ada",
	})
	grant.Header.Set("X-Workspace-ID", wsID)
	grant = withURLParam(grant, "id", agentID)
	testutil.Call(t, testHandler.GrantAgentMailInbox, grant).Want(http.StatusOK)

	del := newRequest(http.MethodDelete, "/api/workspaces/"+wsID, nil)
	del = withURLParam(del, "id", wsID)
	testutil.Call(t, testHandler.DeleteWorkspace, del).Want(http.StatusNoContent)

	var connCount, inboxCount, purgeCount int
	dbfx.QueryRow(t, `SELECT COUNT(*) FROM agentmail_connection WHERE workspace_id = $1`, wsID).Scan(&connCount)
	dbfx.QueryRow(t, `SELECT COUNT(*) FROM agentmail_inbox WHERE workspace_id = $1`, wsID).Scan(&inboxCount)
	dbfx.QueryRow(t, `SELECT COUNT(*) FROM agentmail_purge WHERE workspace_id = $1`, wsID).Scan(&purgeCount)
	if connCount != 0 || inboxCount != 0 {
		t.Fatalf("product rows survived delete: connection=%d inbox=%d", connCount, inboxCount)
	}
	if purgeCount == 0 {
		t.Fatal("purge ledger was deleted with the workspace")
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM agentmail_purge WHERE workspace_id = $1`, wsID)
	})
}

func TestClaimTaskByRuntime_CarriesAgentMailOverlay(t *testing.T) {
	withAgentMail(t)

	testutil.Call(t, testHandler.ConnectAgentMail, agentMailWorkspaceReq(http.MethodPost, map[string]string{
		"mode": "hosted",
	})).Want(http.StatusOK)

	ctx := context.Background()
	runtimeID := createClaimReclaimRuntime(t, ctx, "agentmail overlay runtime")
	agentID, issueID := createClaimReclaimAgentAndIssue(t, ctx, runtimeID, "agentmail overlay")
	testutil.Call(t, testHandler.GrantAgentMailInbox, agentMailAgentReq(http.MethodPut, agentID, map[string]any{
		"username": "overlay",
	})).
		Want(http.StatusOK)

	var taskID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority)
		VALUES ($1, $2, $3, 'queued', 0)
		RETURNING id
	`, agentID, runtimeID, issueID).Scan(&taskID); err != nil {
		t.Fatalf("create task: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, taskID)
	})

	mcp := claimAgentMcpConfigForTest(t, runtimeID)
	var parsed map[string]map[string]any
	if err := json.Unmarshal(mcp, &parsed); err != nil {
		t.Fatalf("mcp_config: %v (%s)", err, string(mcp))
	}
	server, ok := parsed["mcpServers"]["agentmail"].(map[string]any)
	if !ok {
		t.Fatalf("claim mcp missing agentmail: %s", string(mcp))
	}
	headers, _ := server["headers"].(map[string]any)
	key, _ := headers["x-api-key"].(string)
	if key == "" || strings.Contains(key, "hosted_org") {
		t.Fatalf("claim overlay key = %q", key)
	}
}

func TestAgentMailThreadsRequireActiveInboxAndSecrets(t *testing.T) {
	withAgentMail(t)

	runtimeID := dbfx.Runtime(t, "agentmail threads runtime")
	agentID := dbfx.Agent(t, "Thread Mail", runtimeID)

	listReq := newRequest(http.MethodGet, "/api/agents/"+agentID+"/agentmail/threads", nil)
	listReq = withURLParam(listReq, "id", agentID)
	testutil.Call(t, testHandler.ListAgentMailThreads, listReq).Want(http.StatusConflict)

	testutil.Call(t, testHandler.ConnectAgentMail, agentMailWorkspaceReq(http.MethodPost, map[string]string{
		"mode": "hosted",
	})).Want(http.StatusOK)
	testutil.Call(t, testHandler.GrantAgentMailInbox, agentMailAgentReq(http.MethodPut, agentID, map[string]any{
		"username": "threads",
	})).
		Want(http.StatusOK)

	var listed AgentMailThreadListResponse
	testutil.Call(t, testHandler.ListAgentMailThreads, listReq).Want(http.StatusOK).JSON(&listed)
	if listed.Threads == nil {
		t.Fatal("threads must be an empty list, not null")
	}

	missing := newRequest(http.MethodGet, "/api/agents/"+agentID+"/agentmail/threads/missing", nil)
	missing = withURLParams(missing, "id", agentID, "threadId", "missing")
	testutil.Call(t, testHandler.GetAgentMailThread, missing).Want(http.StatusNotFound)

	memberID := dbfx.User(t, "AgentMail Thread Member", "agentmail-thread-"+agentID[:8]+"@example.com")
	dbfx.Member(t, testWorkspaceID, memberID, "member")
	memberReq := newRequest(http.MethodGet, "/api/agents/"+agentID+"/agentmail/threads", nil)
	memberReq = withURLParam(memberReq, "id", agentID)
	memberReq.Header.Set("X-User-ID", memberID)
	testutil.Call(t, testHandler.ListAgentMailThreads, memberReq).Want(http.StatusForbidden)
}
