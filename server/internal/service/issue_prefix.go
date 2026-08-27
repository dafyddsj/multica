package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// issuePrefixSet maps a project to the issue prefix its parent initiative
// overrides, falling back to the workspace prefix. Broadcast paths must use
// this so an agent-driven issue:updated cannot rewrite MOB-134 to WS-134.
type issuePrefixSet struct {
	workspace string
	byProject map[string]string
}

func (s issuePrefixSet) forProject(projectID pgtype.UUID) string {
	if projectID.Valid {
		if prefix := s.byProject[util.UUIDToString(projectID)]; prefix != "" {
			return prefix
		}
	}
	return s.workspace
}

func loadIssuePrefixSet(ctx context.Context, q *db.Queries, workspaceID pgtype.UUID) issuePrefixSet {
	set := issuePrefixSet{byProject: map[string]string{}}
	if q == nil {
		return set
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ws, err := q.GetWorkspace(ctx, workspaceID)
	if err == nil {
		set.workspace = ws.IssuePrefix
	}
	rows, err := q.ListProjectInitiativeIssuePrefixes(ctx, workspaceID)
	if err != nil {
		return set
	}
	for _, row := range rows {
		if row.IssuePrefix.Valid && row.IssuePrefix.String != "" {
			set.byProject[util.UUIDToString(row.ID)] = row.IssuePrefix.String
		}
	}
	return set
}

func issuePrefixForProject(ctx context.Context, q *db.Queries, workspaceID, projectID pgtype.UUID) string {
	return loadIssuePrefixSet(ctx, q, workspaceID).forProject(projectID)
}
