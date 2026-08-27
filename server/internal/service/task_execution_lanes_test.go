package service

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/executionlane"
	"github.com/multica-ai/multica/server/internal/featureflags"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/featureflag"
	"github.com/multica-ai/multica/server/pkg/taskfailure"
)

func executionLanesTestFlags(enabled bool) *featureflag.Service {
	provider := featureflag.NewStaticProvider()
	provider.Set(featureflags.AgentExecutionLanes, featureflag.Rule{Default: enabled})
	return featureflag.NewService(provider)
}

func TestFailTaskHopsLightweightToPrimary(t *testing.T) {
	pool := newResolveOriginatorPool(t)
	ctx := context.Background()
	_, _, agentID, _ := seedAttributionFixture(t, pool)

	var runtimeID string
	if err := pool.QueryRow(ctx, `SELECT runtime_id::text FROM agent WHERE id = $1`, agentID).Scan(&runtimeID); err != nil {
		t.Fatalf("load agent runtime: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE agent
		SET model = 'opus',
		    lightweight_model = 'haiku',
		    start_lightweight = TRUE,
		    failover_model = 'sonnet'
		WHERE id = $1`, agentID); err != nil {
		t.Fatalf("stamp agent lanes: %v", err)
	}

	var taskID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO agent_task_queue
			(agent_id, runtime_id, status, priority, attempt, max_attempts,
			 started_at, execution_lane, model_override)
		VALUES ($1, $2, 'running', 0, 1, 1, now(), 'lightweight', 'haiku')
		RETURNING id`, agentID, runtimeID).Scan(&taskID); err != nil {
		t.Fatalf("seed running task: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM agent_task_queue WHERE id = $1 OR retry_of_task_id = $1`, taskID)
	})

	svc := &TaskService{
		Queries:      db.New(pool),
		TxStarter:    pool,
		Bus:          events.New(),
		FeatureFlags: executionLanesTestFlags(true),
	}
	if _, err := svc.FailTask(ctx, util.MustParseUUID(taskID),
		"model haiku not found", "", "", "",
		string(taskfailure.ReasonAgentModelNotFoundOrUnavailable),
		false, "", ""); err != nil {
		t.Fatalf("FailTask: %v", err)
	}

	var (
		parentStatus string
		childLane    string
		childModel   string
		childFresh   bool
		childRetryOf string
	)
	if err := pool.QueryRow(ctx, `
		SELECT status FROM agent_task_queue WHERE id = $1`, taskID).Scan(&parentStatus); err != nil {
		t.Fatalf("read parent: %v", err)
	}
	if parentStatus != "failed" {
		t.Fatalf("parent status = %q, want failed", parentStatus)
	}
	if err := pool.QueryRow(ctx, `
		SELECT execution_lane, COALESCE(model_override, ''), force_fresh_session, retry_of_task_id::text
		FROM agent_task_queue
		WHERE retry_of_task_id = $1`, taskID).Scan(&childLane, &childModel, &childFresh, &childRetryOf); err != nil {
		t.Fatalf("read hop child: %v", err)
	}
	if childLane != string(executionlane.LanePrimary) {
		t.Fatalf("child lane = %q, want primary", childLane)
	}
	if childModel != "opus" {
		t.Fatalf("child model_override = %q, want opus", childModel)
	}
	if !childFresh {
		t.Fatal("hop child must force a fresh session")
	}
}

func TestFailTaskHopsPrimaryToFailover(t *testing.T) {
	pool := newResolveOriginatorPool(t)
	ctx := context.Background()
	_, _, agentID, _ := seedAttributionFixture(t, pool)

	var runtimeID string
	if err := pool.QueryRow(ctx, `SELECT runtime_id::text FROM agent WHERE id = $1`, agentID).Scan(&runtimeID); err != nil {
		t.Fatalf("load agent runtime: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE agent
		SET model = 'opus',
		    failover_model = 'sonnet'
		WHERE id = $1`, agentID); err != nil {
		t.Fatalf("stamp failover: %v", err)
	}

	var taskID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO agent_task_queue
			(agent_id, runtime_id, status, priority, attempt, max_attempts, started_at, execution_lane)
		VALUES ($1, $2, 'running', 0, 1, 1, now(), 'primary')
		RETURNING id`, agentID, runtimeID).Scan(&taskID); err != nil {
		t.Fatalf("seed running task: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM agent_task_queue WHERE id = $1 OR retry_of_task_id = $1`, taskID)
	})

	svc := &TaskService{
		Queries:      db.New(pool),
		TxStarter:    pool,
		Bus:          events.New(),
		FeatureFlags: executionLanesTestFlags(true),
	}
	if _, err := svc.FailTask(ctx, util.MustParseUUID(taskID),
		"rate limit exceeded", "", "", "",
		string(taskfailure.ReasonAgentProviderCapacityOrRateLimit),
		false, "", ""); err != nil {
		t.Fatalf("FailTask: %v", err)
	}

	var childLane, childModel string
	if err := pool.QueryRow(ctx, `
		SELECT execution_lane, COALESCE(model_override, '')
		FROM agent_task_queue
		WHERE retry_of_task_id = $1`, taskID).Scan(&childLane, &childModel); err != nil {
		t.Fatalf("read hop child: %v", err)
	}
	if childLane != string(executionlane.LaneFailover) || childModel != "sonnet" {
		t.Fatalf("want failover/sonnet, got %s/%s", childLane, childModel)
	}
}

func TestFailTaskDoesNotHopWhenFlagOff(t *testing.T) {
	pool := newResolveOriginatorPool(t)
	ctx := context.Background()
	_, _, agentID, _ := seedAttributionFixture(t, pool)

	var runtimeID string
	if err := pool.QueryRow(ctx, `SELECT runtime_id::text FROM agent WHERE id = $1`, agentID).Scan(&runtimeID); err != nil {
		t.Fatalf("load agent runtime: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE agent SET failover_model = 'sonnet' WHERE id = $1`, agentID); err != nil {
		t.Fatalf("stamp failover: %v", err)
	}

	var taskID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO agent_task_queue
			(agent_id, runtime_id, status, priority, attempt, max_attempts, started_at)
		VALUES ($1, $2, 'running', 0, 1, 1, now())
		RETURNING id`, agentID, runtimeID).Scan(&taskID); err != nil {
		t.Fatalf("seed running task: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM agent_task_queue WHERE id = $1 OR retry_of_task_id = $1`, taskID)
	})

	svc := &TaskService{
		Queries:      db.New(pool),
		TxStarter:    pool,
		Bus:          events.New(),
		FeatureFlags: executionLanesTestFlags(false),
	}
	if _, err := svc.FailTask(ctx, util.MustParseUUID(taskID),
		"rate limit exceeded", "", "", "",
		string(taskfailure.ReasonAgentProviderCapacityOrRateLimit),
		false, "", ""); err != nil {
		t.Fatalf("FailTask: %v", err)
	}

	var n int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM agent_task_queue WHERE retry_of_task_id = $1`, taskID).Scan(&n); err != nil {
		t.Fatalf("count children: %v", err)
	}
	if n != 0 {
		t.Fatalf("flag off must not hop, got %d children", n)
	}
}

func TestFailTaskDoesNotHopAutopilot(t *testing.T) {
	pool := newResolveOriginatorPool(t)
	ctx := context.Background()
	workspaceID, userID, agentID, _ := seedAttributionFixture(t, pool)
	_, runID := seedRunOnlyAutopilot(t, pool, workspaceID, agentID, userID)

	var runtimeID string
	if err := pool.QueryRow(ctx, `SELECT runtime_id::text FROM agent WHERE id = $1`, agentID).Scan(&runtimeID); err != nil {
		t.Fatalf("load agent: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE agent SET failover_model = 'sonnet' WHERE id = $1`, agentID); err != nil {
		t.Fatalf("stamp failover: %v", err)
	}

	var taskID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO agent_task_queue
			(agent_id, runtime_id, status, priority, attempt, max_attempts,
			 started_at, autopilot_run_id)
		VALUES ($1, $2, 'running', 0, 1, 1, now(), $3)
		RETURNING id`, agentID, runtimeID, runID).Scan(&taskID); err != nil {
		t.Fatalf("seed autopilot task: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM agent_task_queue WHERE id = $1 OR retry_of_task_id = $1`, taskID)
	})

	svc := &TaskService{
		Queries:      db.New(pool),
		TxStarter:    pool,
		Bus:          events.New(),
		FeatureFlags: executionLanesTestFlags(true),
	}
	if _, err := svc.FailTask(ctx, util.MustParseUUID(taskID),
		"rate limit exceeded", "", "", "",
		string(taskfailure.ReasonAgentProviderCapacityOrRateLimit),
		false, "", ""); err != nil {
		t.Fatalf("FailTask: %v", err)
	}

	var n int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM agent_task_queue WHERE retry_of_task_id = $1`, taskID).Scan(&n); err != nil {
		t.Fatalf("count children: %v", err)
	}
	if n != 0 {
		t.Fatalf("autopilot must not hop, got %d children", n)
	}
}

func TestFailTaskDoesNotHopUnknownReason(t *testing.T) {
	pool := newResolveOriginatorPool(t)
	ctx := context.Background()
	_, _, agentID, _ := seedAttributionFixture(t, pool)

	var runtimeID string
	if err := pool.QueryRow(ctx, `SELECT runtime_id::text FROM agent WHERE id = $1`, agentID).Scan(&runtimeID); err != nil {
		t.Fatalf("load agent runtime: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE agent SET failover_model = 'sonnet' WHERE id = $1`, agentID); err != nil {
		t.Fatalf("stamp failover: %v", err)
	}

	var taskID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO agent_task_queue
			(agent_id, runtime_id, status, priority, attempt, max_attempts, started_at)
		VALUES ($1, $2, 'running', 0, 1, 1, now())
		RETURNING id`, agentID, runtimeID).Scan(&taskID); err != nil {
		t.Fatalf("seed running task: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM agent_task_queue WHERE id = $1 OR retry_of_task_id = $1`, taskID)
	})

	svc := &TaskService{
		Queries:      db.New(pool),
		TxStarter:    pool,
		Bus:          events.New(),
		FeatureFlags: executionLanesTestFlags(true),
	}
	if _, err := svc.FailTask(ctx, util.MustParseUUID(taskID),
		"worker failed diagnostic details", "", "", "",
		"agent_error", false, "", ""); err != nil {
		t.Fatalf("FailTask: %v", err)
	}

	var n int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM agent_task_queue WHERE retry_of_task_id = $1`, taskID).Scan(&n); err != nil {
		t.Fatalf("count children: %v", err)
	}
	if n != 0 {
		t.Fatalf("unknown reason must not hop, got %d children", n)
	}
}

func TestInitialLaneStampUsesLightweightWhenEnabled(t *testing.T) {
	svc := &TaskService{FeatureFlags: executionLanesTestFlags(true)}
	stamp := svc.initialLaneStamp(context.Background(), db.Agent{
		Model:            pgtype.Text{String: "opus", Valid: true},
		LightweightModel: pgtype.Text{String: "haiku", Valid: true},
		StartLightweight: true,
	})
	if stamp.Lane.String != string(executionlane.LaneLightweight) {
		t.Fatalf("lane = %q, want lightweight", stamp.Lane.String)
	}
	if stamp.ModelOverride.String != "haiku" || !stamp.ForceFreshSession {
		t.Fatalf("stamp = %+v", stamp)
	}

	off := &TaskService{FeatureFlags: executionLanesTestFlags(false)}
	disabled := off.initialLaneStamp(context.Background(), db.Agent{
		Model:            pgtype.Text{String: "opus", Valid: true},
		LightweightModel: pgtype.Text{String: "haiku", Valid: true},
		StartLightweight: true,
	})
	if disabled.Lane.String != string(executionlane.LanePrimary) || disabled.ModelOverride.Valid {
		t.Fatalf("disabled stamp = %+v", disabled)
	}
}
