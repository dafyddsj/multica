package memory

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// NativeEngine is the v1 retrieval backend. It searches the Multica
// memory_entry table with ILIKE. A later Hindsight adapter implements
// MemoryEngine the same way.
type NativeEngine struct {
	Queries *db.Queries
}

func (e NativeEngine) Search(ctx context.Context, q SearchQuery) ([]Hit, error) {
	if e.Queries == nil {
		return nil, nil
	}
	limit := q.Limit
	if limit <= 0 {
		limit = MaxRecallHits
	}
	rows, err := e.Queries.ListMemoryEntriesForRecall(ctx, db.ListMemoryEntriesForRecallParams{
		Limit:        limit,
		WorkspaceID:  parseOptionalUUID(q.WorkspaceID),
		InitiativeID: parseOptionalUUID(q.InitiativeID),
		ProjectID:    parseOptionalUUID(q.ProjectID),
		IssueID:      parseOptionalUUID(q.IssueID),
		SquadID:      parseOptionalUUID(q.SquadID),
		AgentID:      parseOptionalUUID(q.AgentID),
		UserID:       parseOptionalUUID(q.UserID),
		Query:        optionalText(q.Query),
	})
	if err != nil {
		return nil, err
	}
	hits := make([]Hit, 0, len(rows))
	for _, row := range rows {
		hits = append(hits, Hit{
			ID:      util.UUIDToString(row.ID),
			Scope:   row.Scope,
			OwnerID: util.UUIDToString(row.OwnerID),
			Body:    truncateRunes(row.Body, MaxRecallRunes),
			Kind:    row.Kind,
		})
	}
	return hits, nil
}

func optionalText(s string) pgtype.Text {
	s = strings.TrimSpace(s)
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}

func parseOptionalUUID(s string) pgtype.UUID {
	s = strings.TrimSpace(s)
	if s == "" {
		return pgtype.UUID{}
	}
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}
	}
	return id
}
