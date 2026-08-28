package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/testutil"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/dbid"
)

func TestBudgetTaskContextWriteOnce(t *testing.T) {
	project := util.MustParseUUID("11111111-1111-1111-1111-111111111111")
	other := util.MustParseUUID("22222222-2222-2222-2222-222222222222")
	var c budgetTaskContext
	if err := c.write(budgetTaskContext{ProjectID: project}); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := c.write(budgetTaskContext{ProjectID: project}); err != nil {
		t.Fatalf("identical second write: %v", err)
	}
	if err := c.write(budgetTaskContext{ProjectID: other}); !errors.Is(err, ErrBudgetContextConflict) {
		t.Fatalf("conflicting write err = %v, want ErrBudgetContextConflict", err)
	}
	if !c.ProjectID.Valid || c.ProjectID.Bytes != project.Bytes {
		t.Fatalf("context mutated after conflict: %v", c.ProjectID)
	}
}

type budgetCoverageGraph struct {
	pool         *testutil.Fixture
	workspaceID  string
	userID       string
	runtimeID    string
	agentID      string
	workerID     string
	squadID      string
	initiativeID string
	projectID    string
	otherProject string
	issueID      string
}

func seedBudgetCoverageGraph(t *testing.T) (*budgetCoverageGraph, *TaskService) {
	t.Helper()
	pool := newResolveOriginatorPool(t)
	base := testutil.New(pool, "", "")
	suffix := time.Now().UnixNano()
	userID := base.User(t, "Budget User", fmt.Sprintf("budget-%d@multica.test", suffix))
	workspaceID := base.Workspace(t, "budget ws", fmt.Sprintf("budget-%d", suffix))
	base.Member(t, workspaceID, userID, "owner")
	fx := testutil.New(pool, workspaceID, userID)

	runtimeID := fx.Runtime(t, fmt.Sprintf("budget-rt-%d", suffix))
	agentID := fx.Agent(t, fmt.Sprintf("budget-agent-%d", suffix), runtimeID, testutil.Cols{
		"visibility": "workspace",
	})
	workerID := fx.Agent(t, fmt.Sprintf("budget-worker-%d", suffix), runtimeID, testutil.Cols{
		"visibility": "workspace",
	})
	squadID := fx.Squad(t, fmt.Sprintf("budget-squad-%d", suffix), agentID)
	fx.SquadMember(t, squadID, "agent", agentID)
	fx.SquadMember(t, squadID, "agent", workerID)

	initiativeID := fx.Initiative(t, "budget initiative")
	projectID := fx.Project(t, "budget project", testutil.Cols{"initiative_id": initiativeID})
	otherProject := fx.Project(t, "other project")
	issueID := fx.Issue(t, "budget issue", testutil.Cols{
		"assignee_type": "agent",
		"assignee_id":   agentID,
		"priority":      "medium",
		"project_id":    projectID,
	})
	fx.Cleanup(t, `DELETE FROM agent_task_queue WHERE agent_id = ANY($1::uuid[])`, []string{agentID, workerID})

	return &budgetCoverageGraph{
		pool:         fx,
		workspaceID:  workspaceID,
		userID:       userID,
		runtimeID:    runtimeID,
		agentID:      agentID,
		workerID:     workerID,
		squadID:      squadID,
		initiativeID: initiativeID,
		projectID:    projectID,
		otherProject: otherProject,
		issueID:      issueID,
	}, &TaskService{Queries: db.New(pool), TxStarter: pool, Bus: events.New()}
}

func (g *budgetCoverageGraph) issue(t *testing.T) db.Issue {
	t.Helper()
	var issue db.Issue
	g.pool.QueryRow(t, `SELECT id, workspace_id, title, status, priority, assignee_type, assignee_id,
		creator_type, creator_id, project_id FROM issue WHERE id = $1`, g.issueID).Scan(
		&issue.ID, &issue.WorkspaceID, &issue.Title, &issue.Status, &issue.Priority,
		&issue.AssigneeType, &issue.AssigneeID, &issue.CreatorType, &issue.CreatorID, &issue.ProjectID)
	return issue
}

func readBudgetCoverage(t *testing.T, fx *testutil.Fixture, taskID pgtype.UUID) (project, initiative, origin string) {
	t.Helper()
	var p, i, s pgtype.UUID
	fx.QueryRow(t, `
		SELECT budget_project_id, budget_initiative_id, budget_origin_squad_id
		FROM agent_task_queue WHERE id = $1`, taskID).Scan(&p, &i, &s)
	return util.UUIDToString(p), util.UUIDToString(i), util.UUIDToString(s)
}

func TestEnqueueSnapshotsBudgetCoverage(t *testing.T) {
	g, svc := seedBudgetCoverageGraph(t)
	ctx := context.Background()
	issue := g.issue(t)

	t.Run("issue", func(t *testing.T) {
		task, err := svc.EnqueueTaskForIssue(ctx, issue)
		if err != nil {
			t.Fatalf("EnqueueTaskForIssue: %v", err)
		}
		project, initiative, origin := readBudgetCoverage(t, g.pool, task.ID)
		if project != g.projectID || initiative != g.initiativeID {
			t.Fatalf("issue coverage = project %s initiative %s, want %s / %s", project, initiative, g.projectID, g.initiativeID)
		}
		if origin != "" {
			t.Fatalf("issue origin squad = %q, want empty (membership does not count)", origin)
		}
	})

	t.Run("project_move_does_not_restamp", func(t *testing.T) {
		task, err := svc.EnqueueTaskForMention(ctx, issue, util.MustParseUUID(g.workerID), pgtype.UUID{})
		if err != nil {
			t.Fatalf("EnqueueTaskForMention: %v", err)
		}
		g.pool.Exec(t, `UPDATE issue SET project_id = $1 WHERE id = $2`, g.otherProject, g.issueID)
		t.Cleanup(func() {
			g.pool.Exec(t, `UPDATE issue SET project_id = $1 WHERE id = $2`, g.projectID, g.issueID)
		})
		project, initiative, _ := readBudgetCoverage(t, g.pool, task.ID)
		if project != g.projectID || initiative != g.initiativeID {
			t.Fatalf("after move: project %s initiative %s, want frozen %s / %s", project, initiative, g.projectID, g.initiativeID)
		}
	})

	t.Run("squad_leader", func(t *testing.T) {
		g.pool.Exec(t, `DELETE FROM agent_task_queue WHERE issue_id = $1 AND agent_id = $2`, g.issueID, g.agentID)
		task, err := svc.EnqueueTaskForSquadLeader(ctx, issue, util.MustParseUUID(g.agentID), util.MustParseUUID(g.squadID), pgtype.UUID{})
		if err != nil {
			t.Fatalf("EnqueueTaskForSquadLeader: %v", err)
		}
		project, initiative, origin := readBudgetCoverage(t, g.pool, task.ID)
		if project != g.projectID || initiative != g.initiativeID || origin != g.squadID {
			t.Fatalf("leader coverage = %s / %s / %s, want %s / %s / %s", project, initiative, origin, g.projectID, g.initiativeID, g.squadID)
		}
	})

	t.Run("squad_worker_inherits_origin", func(t *testing.T) {
		g.pool.Exec(t, `DELETE FROM agent_task_queue WHERE issue_id = $1`, g.issueID)
		leader, err := svc.EnqueueTaskForSquadLeader(ctx, issue, util.MustParseUUID(g.agentID), util.MustParseUUID(g.squadID), pgtype.UUID{})
		if err != nil {
			t.Fatalf("leader: %v", err)
		}
		commentID := g.pool.Comment(t, g.issueID, "delegate to worker", testutil.Cols{
			"author_type":    "agent",
			"author_id":      g.agentID,
			"source_task_id": leader.ID,
		})
		worker, err := svc.EnqueueTaskForMention(ctx, issue, util.MustParseUUID(g.workerID), util.MustParseUUID(commentID))
		if err != nil {
			t.Fatalf("worker mention: %v", err)
		}
		_, _, leaderOrigin := readBudgetCoverage(t, g.pool, leader.ID)
		_, _, workerOrigin := readBudgetCoverage(t, g.pool, worker.ID)
		if workerOrigin != leaderOrigin || workerOrigin != g.squadID {
			t.Fatalf("worker origin %s, leader origin %s, want %s", workerOrigin, leaderOrigin, g.squadID)
		}
	})

	t.Run("deferred_fallback", func(t *testing.T) {
		g.pool.Exec(t, `DELETE FROM agent_task_queue WHERE agent_id = $1`, g.workerID)
		task, err := svc.EnqueueDeferredAssigneeFallback(
			ctx, issue, util.MustParseUUID(g.workerID), util.MustParseUUID(g.squadID),
			pgtype.UUID{}, pgtype.UUID{}, time.Now().Add(time.Hour),
		)
		if err != nil {
			t.Fatalf("EnqueueDeferredAssigneeFallback: %v", err)
		}
		project, initiative, origin := readBudgetCoverage(t, g.pool, task.ID)
		if project != g.projectID || initiative != g.initiativeID || origin != g.squadID {
			t.Fatalf("deferred coverage = %s / %s / %s, want %s / %s / %s", project, initiative, origin, g.projectID, g.initiativeID, g.squadID)
		}
	})

	t.Run("deferred_channel_issue", func(t *testing.T) {
		g.pool.Exec(t, `DELETE FROM agent_task_queue WHERE issue_id = $1 AND agent_id = $2`, g.issueID, g.agentID)
		task, err := svc.EnqueueDeferredChannelIssueTask(ctx, issue, time.Now().Add(time.Minute))
		if err != nil {
			t.Fatalf("EnqueueDeferredChannelIssueTask: %v", err)
		}
		project, initiative, origin := readBudgetCoverage(t, g.pool, task.ID)
		if project != g.projectID || initiative != g.initiativeID {
			t.Fatalf("channel deferred coverage = %s / %s, want %s / %s", project, initiative, g.projectID, g.initiativeID)
		}
		if origin != "" {
			t.Fatalf("channel deferred origin = %q, want empty", origin)
		}
	})

	t.Run("quick_create", func(t *testing.T) {
		task, err := svc.EnqueueQuickCreateTask(
			ctx,
			util.MustParseUUID(g.workspaceID),
			util.MustParseUUID(g.userID),
			util.MustParseUUID(g.agentID),
			util.MustParseUUID(g.squadID),
			"create an issue",
			"high",
			"",
			util.MustParseUUID(g.projectID),
			pgtype.UUID{},
			nil,
		)
		if err != nil {
			t.Fatalf("EnqueueQuickCreateTask: %v", err)
		}
		project, initiative, origin := readBudgetCoverage(t, g.pool, task.ID)
		if project != g.projectID || initiative != g.initiativeID || origin != g.squadID {
			t.Fatalf("quick-create coverage = %s / %s / %s, want %s / %s / %s", project, initiative, origin, g.projectID, g.initiativeID, g.squadID)
		}
	})

	t.Run("chat_stays_unattributed", func(t *testing.T) {
		chatSessionID := g.pool.ChatSession(t, g.agentID, testutil.Cols{"project_id": g.projectID})
		seedChannelTaskBinding(t, g.pool.Pool, chatSessionID)
		task, err := svc.EnqueueChatTask(ctx, db.ChatSession{
			ID:          util.MustParseUUID(chatSessionID),
			WorkspaceID: util.MustParseUUID(g.workspaceID),
			AgentID:     util.MustParseUUID(g.agentID),
		}, util.MustParseUUID(g.userID), false)
		if err != nil {
			t.Fatalf("EnqueueChatTask: %v", err)
		}
		project, initiative, origin := readBudgetCoverage(t, g.pool, task.ID)
		if project != "" || initiative != "" || origin != "" {
			t.Fatalf("chat coverage = %s / %s / %s, want all empty", project, initiative, origin)
		}
	})

	t.Run("retry_inherits", func(t *testing.T) {
		g.pool.Exec(t, `DELETE FROM agent_task_queue WHERE issue_id = $1`, g.issueID)
		parent, err := svc.EnqueueTaskForSquadLeader(ctx, issue, util.MustParseUUID(g.agentID), util.MustParseUUID(g.squadID), pgtype.UUID{})
		if err != nil {
			t.Fatalf("parent: %v", err)
		}
		g.pool.Exec(t, `UPDATE agent_task_queue SET status = 'failed', attempt = 1, max_attempts = 3 WHERE id = $1`, parent.ID)
		child, err := svc.Queries.CreateRetryTask(ctx, db.CreateRetryTaskParams{ID: parent.ID})
		if err != nil {
			t.Fatalf("CreateRetryTask: %v", err)
		}
		pProject, pInit, pOrigin := readBudgetCoverage(t, g.pool, parent.ID)
		cProject, cInit, cOrigin := readBudgetCoverage(t, g.pool, child.ID)
		if cProject != pProject || cInit != pInit || cOrigin != pOrigin {
			t.Fatalf("retry coverage = %s / %s / %s, want parent %s / %s / %s", cProject, cInit, cOrigin, pProject, pInit, pOrigin)
		}
	})

	t.Run("recovery_inherits_origin_and_snapshots_issue", func(t *testing.T) {
		f, recoverySvc := seedDelegatedFailureFixture(t)
		initiativeID := f.pool.QueryRow(context.Background(), `
			INSERT INTO initiative (workspace_id, title, status, priority)
			VALUES ($1, 'recovery initiative', 'planned', 'none') RETURNING id`, f.workspaceID)
		var initiative string
		if err := initiativeID.Scan(&initiative); err != nil {
			t.Fatalf("seed initiative: %v", err)
		}
		t.Cleanup(func() { f.pool.Exec(context.Background(), `DELETE FROM initiative WHERE id = $1`, initiative) })
		var project string
		if err := f.pool.QueryRow(context.Background(), `
			INSERT INTO project (workspace_id, title, status, priority, initiative_id)
			VALUES ($1, 'recovery project', 'planned', 'none', $2) RETURNING id`, f.workspaceID, initiative).Scan(&project); err != nil {
			t.Fatalf("seed project: %v", err)
		}
		t.Cleanup(func() { f.pool.Exec(context.Background(), `DELETE FROM project WHERE id = $1`, project) })
		if _, err := f.pool.Exec(context.Background(), `UPDATE issue SET project_id = $1 WHERE id = $2`, project, f.issueID); err != nil {
			t.Fatalf("stamp issue project: %v", err)
		}
		var squad string
		if err := f.pool.QueryRow(context.Background(), `
			INSERT INTO squad (workspace_id, name, description, leader_id, creator_id)
			VALUES ($1, 'recovery squad', '', $2, $3) RETURNING id`, f.workspaceID, f.coordinator, f.userID).Scan(&squad); err != nil {
			t.Fatalf("seed squad: %v", err)
		}
		t.Cleanup(func() { f.pool.Exec(context.Background(), `DELETE FROM squad WHERE id = $1`, squad) })
		if _, err := f.pool.Exec(context.Background(), `
			UPDATE agent_task_queue
			SET budget_project_id = $1, budget_initiative_id = $2, budget_origin_squad_id = $3,
			    is_leader_task = TRUE, squad_id = $3
			WHERE id = $4`, project, initiative, squad, f.sourceTask); err != nil {
			t.Fatalf("stamp source coverage: %v", err)
		}

		failedID := f.insertWorkerTask(t, "running", "comment", 1, 2)
		if _, err := recoverySvc.FailTask(ctx, failedID, "delegated worker died", "", "", "", "agent_error.process_failure", false, "", ""); err != nil {
			t.Fatalf("FailTask: %v", err)
		}
		var recoveryID pgtype.UUID
		if err := f.pool.QueryRow(ctx, `
			SELECT id FROM agent_task_queue
			WHERE trigger_evidence_kind = 'delegated_failure' AND trigger_evidence_ref_id = $1`, failedID).Scan(&recoveryID); err != nil {
			t.Fatalf("load recovery task: %v", err)
		}
		var recProject, recInit, recOrigin pgtype.UUID
		if err := f.pool.QueryRow(ctx, `
			SELECT budget_project_id, budget_initiative_id, budget_origin_squad_id
			FROM agent_task_queue WHERE id = $1`, recoveryID).Scan(&recProject, &recInit, &recOrigin); err != nil {
			t.Fatalf("read recovery coverage: %v", err)
		}
		if util.UUIDToString(recProject) != project || util.UUIDToString(recInit) != initiative || util.UUIDToString(recOrigin) != squad {
			t.Fatalf("recovery coverage = %s / %s / %s, want %s / %s / %s",
				util.UUIDToString(recProject), util.UUIDToString(recInit), util.UUIDToString(recOrigin),
				project, initiative, squad)
		}
	})

	t.Run("autopilot_run_only", func(t *testing.T) {
		var autopilotID, runID string
		g.pool.QueryRow(t, `
			INSERT INTO autopilot (
				workspace_id, title, assignee_type, assignee_id, status, execution_mode,
				created_by_type, created_by_id
			) VALUES ($1, 'budget autopilot', 'squad', $2, 'active', 'run_only', 'member', $3)
			RETURNING id`, g.workspaceID, g.squadID, g.userID).Scan(&autopilotID)
		g.pool.Cleanup(t, `DELETE FROM autopilot WHERE id = $1`, autopilotID)
		g.pool.QueryRow(t, `
			INSERT INTO autopilot_run (autopilot_id, source, status) VALUES ($1, 'manual', 'running')
			RETURNING id`, autopilotID).Scan(&runID)
		g.pool.Cleanup(t, `DELETE FROM autopilot_run WHERE id = $1`, runID)

		params := db.CreateAutopilotTaskParams{
			ID:                   dbid.NewV7(),
			AgentID:              util.MustParseUUID(g.agentID),
			RuntimeID:            util.MustParseUUID(g.runtimeID),
			AutopilotRunID:       util.MustParseUUID(runID),
			OriginatorUserID:     util.MustParseUUID(g.userID),
			AccountableUserID:    util.MustParseUUID(g.userID),
			OriginatorSource:     pgtype.Text{String: "direct_human", Valid: true},
			TriggerEvidenceKind:  pgtype.Text{String: "autopilot_run", Valid: true},
			TriggerEvidenceRefID: util.MustParseUUID(runID),
		}
		svc.resolveBudgetTaskContext(ctx, util.MustParseUUID(g.workspaceID), pgtype.UUID{}, util.MustParseUUID(g.squadID)).applyAutopilot(&params)
		task, err := svc.Queries.CreateAutopilotTask(ctx, params)
		if err != nil {
			t.Fatalf("CreateAutopilotTask: %v", err)
		}
		project, initiative, origin := readBudgetCoverage(t, g.pool, task.ID)
		if project != "" || initiative != "" {
			t.Fatalf("autopilot project/initiative = %s / %s, want empty (no issue)", project, initiative)
		}
		if origin != g.squadID {
			t.Fatalf("autopilot origin = %s, want squad %s", origin, g.squadID)
		}
	})
}
