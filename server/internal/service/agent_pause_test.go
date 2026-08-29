package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

func loadAgent(t *testing.T, q *db.Queries, agentID string) db.Agent {
	t.Helper()
	agent, err := q.GetAgent(context.Background(), util.MustParseUUID(agentID))
	if err != nil {
		t.Fatalf("load agent: %v", err)
	}
	return agent
}

func TestPauseActorConstructors(t *testing.T) {
	user := PausedByUser(util.MustParseUUID("00000000-0000-0000-0000-000000000001"))
	if !user.UserID.Valid || user.BudgetID.Valid || !user.valid() {
		t.Fatalf("PausedByUser = %+v", user)
	}
	budget := PausedByBudget(util.MustParseUUID("00000000-0000-0000-0000-000000000002"))
	if budget.UserID.Valid || !budget.BudgetID.Valid || !budget.valid() {
		t.Fatalf("PausedByBudget = %+v", budget)
	}
	if (PauseActor{}).valid() {
		t.Fatal("zero PauseActor must be invalid")
	}
}

func TestPauseKeepsFirstActor(t *testing.T) {
	pool := newResolveOriginatorPool(t)
	ctx := context.Background()
	_, userID, agentID, _ := seedAttributionFixture(t, pool)
	q := db.New(pool)
	bus := events.New()
	var pausedEvents int
	bus.Subscribe(protocol.EventAgentPaused, func(events.Event) { pausedEvents++ })
	svc := NewAgentPauseService(q, bus)

	agent := loadAgent(t, q, agentID)
	first, err := svc.Pause(ctx, agent, PausedByUser(util.MustParseUUID(userID)), "member", userID)
	if err != nil {
		t.Fatalf("first pause: %v", err)
	}
	if !first.PausedAt.Valid || !first.PausedBy.Valid || first.PausedByBudgetID.Valid {
		t.Fatalf("first pause actor = paused_by=%v budget=%v", first.PausedBy, first.PausedByBudgetID)
	}

	budgetID := util.MustParseUUID("00000000-0000-4000-8000-000000000099")
	second, err := svc.Pause(ctx, first, PausedByBudget(budgetID), "system", "")
	if err != nil {
		t.Fatalf("second pause: %v", err)
	}
	if second.PausedAt != first.PausedAt {
		t.Fatalf("idempotent pause reset paused_at")
	}
	if second.PausedBy != first.PausedBy || second.PausedByBudgetID.Valid {
		t.Fatalf("second pause overwrote actor: %+v", second)
	}
	if pausedEvents != 1 {
		t.Fatalf("paused events = %d, want 1", pausedEvents)
	}
}

func TestPauseRefusesSystemAgent(t *testing.T) {
	pool := newResolveOriginatorPool(t)
	ctx := context.Background()
	_, userID, agentID, _ := seedAttributionFixture(t, pool)
	if _, err := pool.Exec(ctx, `UPDATE agent SET system_key = 'mika' WHERE id = $1`, agentID); err != nil {
		t.Fatalf("stamp system agent: %v", err)
	}
	q := db.New(pool)
	svc := NewAgentPauseService(q, events.New())
	_, err := svc.Pause(ctx, loadAgent(t, q, agentID), PausedByUser(util.MustParseUUID(userID)), "member", userID)
	if !errors.Is(err, ErrSystemAgent) {
		t.Fatalf("pause system agent: %v", err)
	}
	if loadAgent(t, q, agentID).PausedAt.Valid {
		t.Fatal("system agent was paused")
	}
}

func TestPauseDoesNotCancelTasks(t *testing.T) {
	pool := newResolveOriginatorPool(t)
	ctx := context.Background()
	_, userID, agentID, _ := seedAttributionFixture(t, pool)
	q := db.New(pool)
	svc := NewAgentPauseService(q, events.New())

	var runtimeID string
	if err := pool.QueryRow(ctx, `SELECT runtime_id::text FROM agent WHERE id = $1`, agentID).Scan(&runtimeID); err != nil {
		t.Fatalf("load runtime: %v", err)
	}
	var queuedID, runningID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, status, priority, attempt, max_attempts)
		VALUES ($1, $2, 'queued', 0, 0, 1)
		RETURNING id`, agentID, runtimeID).Scan(&queuedID); err != nil {
		t.Fatalf("seed queued: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (agent_id, runtime_id, status, priority, attempt, max_attempts, started_at)
		VALUES ($1, $2, 'running', 0, 1, 1, now())
		RETURNING id`, agentID, runtimeID).Scan(&runningID); err != nil {
		t.Fatalf("seed running: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM agent_task_queue WHERE id = ANY($1::uuid[])`,
			[]string{queuedID, runningID})
	})

	if _, err := svc.Pause(ctx, loadAgent(t, q, agentID), PausedByUser(util.MustParseUUID(userID)), "member", userID); err != nil {
		t.Fatalf("pause: %v", err)
	}
	for _, taskID := range []string{queuedID, runningID} {
		var status string
		if err := pool.QueryRow(ctx, `SELECT status FROM agent_task_queue WHERE id = $1`, taskID).Scan(&status); err != nil {
			t.Fatalf("read task %s: %v", taskID, err)
		}
		want := "queued"
		if taskID == runningID {
			want = "running"
		}
		if status != want {
			t.Errorf("task %s after pause = %q, want %q", taskID, status, want)
		}
	}
}

func TestResumeBudgetPausedLeavesHumanPause(t *testing.T) {
	pool := newResolveOriginatorPool(t)
	ctx := context.Background()
	_, userID, agentID, _ := seedAttributionFixture(t, pool)
	q := db.New(pool)
	svc := NewAgentPauseService(q, events.New())
	budgetID := util.MustParseUUID("00000000-0000-4000-8000-000000000088")

	if _, err := svc.Pause(ctx, loadAgent(t, q, agentID), PausedByUser(util.MustParseUUID(userID)), "member", userID); err != nil {
		t.Fatalf("human pause: %v", err)
	}
	if err := svc.ResumeBudgetPaused(ctx, budgetID); err != nil {
		t.Fatalf("resume budget: %v", err)
	}
	got := loadAgent(t, q, agentID)
	if !got.PausedAt.Valid || !got.PausedBy.Valid || got.PausedByBudgetID.Valid {
		t.Fatalf("human pause was cleared: %+v", got)
	}
}

func TestResumeBudgetPausedClearsMatchingRows(t *testing.T) {
	pool := newResolveOriginatorPool(t)
	ctx := context.Background()
	_, _, agentID, _ := seedAttributionFixture(t, pool)
	q := db.New(pool)
	bus := events.New()
	var resumed int
	bus.Subscribe(protocol.EventAgentResumed, func(events.Event) { resumed++ })
	svc := NewAgentPauseService(q, bus)
	budgetID := util.MustParseUUID("00000000-0000-4000-8000-000000000077")

	if _, err := svc.Pause(ctx, loadAgent(t, q, agentID), PausedByBudget(budgetID), "system", ""); err != nil {
		t.Fatalf("budget pause: %v", err)
	}
	other := util.MustParseUUID("00000000-0000-4000-8000-000000000066")
	if err := svc.ResumeBudgetPaused(ctx, other); err != nil {
		t.Fatalf("resume other budget: %v", err)
	}
	if !loadAgent(t, q, agentID).PausedAt.Valid {
		t.Fatal("other budget resume cleared this agent")
	}
	if err := svc.ResumeBudgetPaused(ctx, budgetID); err != nil {
		t.Fatalf("resume matching budget: %v", err)
	}
	got := loadAgent(t, q, agentID)
	if got.PausedAt.Valid || got.PausedBy.Valid || got.PausedByBudgetID.Valid {
		t.Fatalf("budget pause not cleared: %+v", got)
	}
	if resumed != 1 {
		t.Fatalf("resumed events = %d, want 1", resumed)
	}
}

func TestResumeClearsAnyPause(t *testing.T) {
	pool := newResolveOriginatorPool(t)
	ctx := context.Background()
	_, _, agentID, _ := seedAttributionFixture(t, pool)
	q := db.New(pool)
	svc := NewAgentPauseService(q, events.New())
	budgetID := util.MustParseUUID("00000000-0000-4000-8000-000000000055")
	paused, err := svc.Pause(ctx, loadAgent(t, q, agentID), PausedByBudget(budgetID), "system", "")
	if err != nil {
		t.Fatalf("budget pause: %v", err)
	}
	resumed, err := svc.Resume(ctx, paused, "member", "user")
	if err != nil {
		t.Fatalf("http resume: %v", err)
	}
	if resumed.PausedAt.Valid || resumed.PausedByBudgetID.Valid {
		t.Fatalf("http resume left pause set: %+v", resumed)
	}
}

func TestPauseRejectsEmptyActor(t *testing.T) {
	svc := NewAgentPauseService(nil, nil)
	_, err := svc.Pause(context.Background(), db.Agent{}, PauseActor{}, "member", "")
	if !errors.Is(err, ErrInvalidPauseActor) {
		t.Fatalf("empty actor: %v", err)
	}
}

func TestResumeRefusesSystemAgent(t *testing.T) {
	svc := NewAgentPauseService(nil, nil)
	_, err := svc.Resume(context.Background(), db.Agent{
		SystemKey: pgtype.Text{String: "mika", Valid: true},
	}, "member", "")
	if !errors.Is(err, ErrSystemAgent) {
		t.Fatalf("resume system agent: %v", err)
	}
}
