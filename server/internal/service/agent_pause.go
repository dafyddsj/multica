package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

var (
	ErrSystemAgent       = errors.New("system agent")
	ErrInvalidPauseActor = errors.New("pause actor must set exactly one of user or budget")
)

// PauseActor is who first paused an agent. Constructors set exactly one side.
type PauseActor struct {
	UserID   pgtype.UUID
	BudgetID pgtype.UUID
}

func PausedByUser(userID pgtype.UUID) PauseActor {
	return PauseActor{UserID: userID}
}

func PausedByBudget(budgetID pgtype.UUID) PauseActor {
	return PauseActor{BudgetID: budgetID}
}

func (a PauseActor) valid() bool {
	return a.UserID.Valid != a.BudgetID.Valid
}

// AgentPauseService is the only writer of agent pause columns.
type AgentPauseService struct {
	Queries *db.Queries
	Bus     *events.Bus
}

func NewAgentPauseService(q *db.Queries, bus *events.Bus) *AgentPauseService {
	return &AgentPauseService{Queries: q, Bus: bus}
}

func isSystemAgent(agent db.Agent) bool {
	return agent.SystemKey.Valid && agent.SystemKey.String != ""
}

func (s *AgentPauseService) Pause(ctx context.Context, agent db.Agent, by PauseActor, actorType, actorID string) (db.Agent, error) {
	if isSystemAgent(agent) {
		return db.Agent{}, ErrSystemAgent
	}
	if !by.valid() {
		return db.Agent{}, ErrInvalidPauseActor
	}
	alreadyPaused := agent.PausedAt.Valid
	paused, err := s.Queries.PauseAgent(ctx, db.PauseAgentParams{
		ID:               agent.ID,
		PausedBy:         by.UserID,
		PausedByBudgetID: by.BudgetID,
	})
	if err != nil {
		return db.Agent{}, fmt.Errorf("pause agent: %w", err)
	}
	if !alreadyPaused {
		s.publishPauseEvent(protocol.EventAgentPaused, paused, actorType, actorID)
	}
	return paused, nil
}

func (s *AgentPauseService) Resume(ctx context.Context, agent db.Agent, actorType, actorID string) (db.Agent, error) {
	if isSystemAgent(agent) {
		return db.Agent{}, ErrSystemAgent
	}
	alreadyActive := !agent.PausedAt.Valid
	resumed, err := s.Queries.ResumeAgent(ctx, agent.ID)
	if err != nil {
		return db.Agent{}, fmt.Errorf("resume agent: %w", err)
	}
	if !alreadyActive {
		s.publishPauseEvent(protocol.EventAgentResumed, resumed, actorType, actorID)
	}
	return resumed, nil
}

func (s *AgentPauseService) ResumeBudgetPaused(ctx context.Context, budgetID pgtype.UUID) error {
	if !budgetID.Valid {
		return ErrInvalidPauseActor
	}
	resumed, err := s.Queries.ResumeBudgetPausedAgents(ctx, budgetID)
	if err != nil {
		return fmt.Errorf("resume budget-paused agents: %w", err)
	}
	for _, agent := range resumed {
		s.publishPauseEvent(protocol.EventAgentResumed, agent, "system", util.UUIDToString(budgetID))
	}
	return nil
}

func (s *AgentPauseService) publishPauseEvent(eventType string, agent db.Agent, actorType, actorID string) {
	if s == nil || s.Bus == nil {
		return
	}
	s.Bus.Publish(events.Event{
		Type:        eventType,
		WorkspaceID: util.UUIDToString(agent.WorkspaceID),
		ActorType:   actorType,
		ActorID:     actorID,
		Payload: map[string]any{
			"agent": map[string]any{
				"id":        util.UUIDToString(agent.ID),
				"paused_at": timestampOrNil(agent.PausedAt),
				"paused_by": uuidOrNil(agent.PausedBy),
			},
		},
	})
}

func timestampOrNil(ts pgtype.Timestamptz) any {
	if !ts.Valid {
		return nil
	}
	return ts.Time.UTC().Format("2006-01-02T15:04:05Z07:00")
}

func uuidOrNil(id pgtype.UUID) any {
	if !id.Valid {
		return nil
	}
	return util.UUIDToString(id)
}
