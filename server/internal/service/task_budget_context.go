package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// ErrBudgetContextConflict is returned when a second write tries to change an
// already-stamped coverage tuple. The snapshot is immutable after enqueue.
var ErrBudgetContextConflict = errors.New("budget task context already written")

// budgetTaskContext is the coverage tuple claimed work is judged against.
// Agent is the task's agent_id. Project, initiative, and origin squad are
// optional and frozen at enqueue so a later issue move or membership change
// cannot retarget spend.
type budgetTaskContext struct {
	ProjectID     pgtype.UUID
	InitiativeID  pgtype.UUID
	OriginSquadID pgtype.UUID
}

func (c budgetTaskContext) equal(other budgetTaskContext) bool {
	return uuidBytesEqual(c.ProjectID, other.ProjectID) &&
		uuidBytesEqual(c.InitiativeID, other.InitiativeID) &&
		uuidBytesEqual(c.OriginSquadID, other.OriginSquadID)
}

func uuidBytesEqual(a, b pgtype.UUID) bool {
	if a.Valid != b.Valid {
		return false
	}
	if !a.Valid {
		return true
	}
	return a.Bytes == b.Bytes
}

func (c *budgetTaskContext) write(next budgetTaskContext) error {
	if c == nil {
		return errors.New("budget task context is nil")
	}
	empty := budgetTaskContext{}
	if c.equal(empty) {
		*c = next
		return nil
	}
	if c.equal(next) {
		return nil
	}
	return ErrBudgetContextConflict
}

func (c budgetTaskContext) applyAgent(p *db.CreateAgentTaskParams) {
	p.BudgetProjectID = c.ProjectID
	p.BudgetInitiativeID = c.InitiativeID
	p.BudgetOriginSquadID = c.OriginSquadID
}

func (c budgetTaskContext) applyDeferredChannel(p *db.CreateDeferredChannelIssueTaskParams) {
	p.BudgetProjectID = c.ProjectID
	p.BudgetInitiativeID = c.InitiativeID
	p.BudgetOriginSquadID = c.OriginSquadID
}

func (c budgetTaskContext) applyDeferred(p *db.CreateDeferredAgentTaskParams) {
	p.BudgetProjectID = c.ProjectID
	p.BudgetInitiativeID = c.InitiativeID
	p.BudgetOriginSquadID = c.OriginSquadID
}

func (c budgetTaskContext) applyQuickCreate(p *db.CreateQuickCreateTaskParams) {
	p.BudgetProjectID = c.ProjectID
	p.BudgetInitiativeID = c.InitiativeID
	p.BudgetOriginSquadID = c.OriginSquadID
}

func (c budgetTaskContext) applyChat(p *db.CreateChatTaskParams) {
	p.BudgetProjectID = c.ProjectID
	p.BudgetInitiativeID = c.InitiativeID
	p.BudgetOriginSquadID = c.OriginSquadID
}

func (c budgetTaskContext) applyAutopilot(p *db.CreateAutopilotTaskParams) {
	p.BudgetProjectID = c.ProjectID
	p.BudgetInitiativeID = c.InitiativeID
	p.BudgetOriginSquadID = c.OriginSquadID
}

// resolveBudgetTaskContext snapshots project → initiative at enqueue.
// A missing or foreign project leaves initiative unset.
func (s *TaskService) resolveBudgetTaskContext(ctx context.Context, workspaceID, projectID, originSquad pgtype.UUID) budgetTaskContext {
	out := budgetTaskContext{ProjectID: projectID, OriginSquadID: originSquad}
	if s == nil || s.Queries == nil || !projectID.Valid || !workspaceID.Valid {
		return out
	}
	project, err := s.Queries.GetProjectInWorkspace(ctx, db.GetProjectInWorkspaceParams{
		ID:          projectID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return out
	}
	out.InitiativeID = project.InitiativeID
	return out
}

func (s *TaskService) issueProjectID(ctx context.Context, issue db.Issue) pgtype.UUID {
	if s != nil && s.Queries != nil && issue.ID.Valid {
		if fresh, err := s.Queries.GetIssue(ctx, issue.ID); err == nil {
			return fresh.ProjectID
		}
	}
	return issue.ProjectID
}

// originSquadID returns the root squad invocation. An explicit squad argument
// wins. Otherwise the parent task (rerun / recovery) or the trigger comment's
// source task may donate its stamp. Membership is never consulted.
func (s *TaskService) originSquadID(ctx context.Context, workspaceID, explicitSquad, triggerCommentID, inheritFromTaskID pgtype.UUID) pgtype.UUID {
	if explicitSquad.Valid {
		return explicitSquad
	}
	if inheritFromTaskID.Valid && s != nil && s.Queries != nil {
		if parent, err := s.Queries.GetAgentTask(ctx, inheritFromTaskID); err == nil && parent.BudgetOriginSquadID.Valid {
			return parent.BudgetOriginSquadID
		}
	}
	if !triggerCommentID.Valid || s == nil || s.Queries == nil || !workspaceID.Valid {
		return pgtype.UUID{}
	}
	comment, err := s.Queries.GetCommentInWorkspace(ctx, db.GetCommentInWorkspaceParams{
		ID:          triggerCommentID,
		WorkspaceID: workspaceID,
	})
	if err != nil || !comment.SourceTaskID.Valid {
		return pgtype.UUID{}
	}
	parent, err := s.Queries.GetAgentTask(ctx, comment.SourceTaskID)
	if err != nil {
		return pgtype.UUID{}
	}
	return parent.BudgetOriginSquadID
}

func (s *TaskService) budgetContextForIssue(ctx context.Context, issue db.Issue, explicitSquad, triggerCommentID, inheritFromTaskID pgtype.UUID) budgetTaskContext {
	origin := s.originSquadID(ctx, issue.WorkspaceID, explicitSquad, triggerCommentID, inheritFromTaskID)
	return s.resolveBudgetTaskContext(ctx, issue.WorkspaceID, s.issueProjectID(ctx, issue), origin)
}

func (s *TaskService) budgetContextForProject(ctx context.Context, workspaceID, projectID, originSquad pgtype.UUID) budgetTaskContext {
	return s.resolveBudgetTaskContext(ctx, workspaceID, projectID, originSquad)
}
