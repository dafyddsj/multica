// Package entitystatus owns the per-workspace Initiative and Project status
// catalogs.
//
// MODEL. There are 5 categories and they map one-to-one onto the 5 built-in
// statuses: a category's value IS its canonical status key. A custom status
// declares a category and inherits that canonical's meaning in full (active vs
// done, picker grouping).
//
// Effective is the identity function on built-in keys, so every existing
// `project.Status == "completed"` comparison keeps its meaning. Custom keys
// resolve to their category. The catalog EXTENDS the built-ins; it does not
// redefine them.
package entitystatus

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

const (
	Initiative = "initiative"
	Project    = "project"

	Planned    = "planned"
	InProgress = "in_progress"
	Paused     = "paused"
	Completed  = "completed"
	Cancelled  = "cancelled"
)

var canonicalOrder = []string{
	Planned,
	InProgress,
	Paused,
	Completed,
	Cancelled,
}

var canonicalRank = func() map[string]int {
	m := make(map[string]int, len(canonicalOrder))
	for i, key := range canonicalOrder {
		m[key] = i
	}
	return m
}()

var ErrUnknownStatus = errors.New("unknown status")
var ErrUnknownResource = errors.New("unknown resource type")

var keyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_]{0,31}$`)

// Querier is the slice of the generated query set this package needs.
type Querier interface {
	GetEntityStatusEntryByKey(ctx context.Context, arg db.GetEntityStatusEntryByKeyParams) (db.EntityStatus, error)
	ListEntityStatusEntries(ctx context.Context, arg db.ListEntityStatusEntriesParams) ([]db.EntityStatus, error)
	SeedEntityStatusEntries(ctx context.Context, workspaceID pgtype.UUID) error
}

func Canonical() []string {
	out := make([]string, len(canonicalOrder))
	copy(out, canonicalOrder)
	return out
}

func IsResourceType(value string) bool {
	return value == Initiative || value == Project
}

func IsBuiltIn(key string) bool {
	_, ok := canonicalRank[key]
	return ok
}

func IsCategory(value string) bool { return IsBuiltIn(value) }

func IsClosed(category string) bool {
	return category == Completed || category == Cancelled
}

func ValidateKey(key string) (string, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return "", errors.New("status key is required")
	}
	if !keyPattern.MatchString(key) {
		return "", errors.New("status key must be 1-32 characters of lowercase letters, digits or underscore, starting with a letter or digit")
	}
	if IsBuiltIn(key) {
		return "", fmt.Errorf("%q is a built-in status key and cannot be reused", key)
	}
	return key, nil
}

func SlugifyKey(name string) (string, error) {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastUnderscore = false
		case !lastUnderscore && b.Len() > 0:
			b.WriteRune('_')
			lastUnderscore = true
		}
	}
	slug := strings.Trim(b.String(), "_")
	if len(slug) > 32 {
		slug = strings.Trim(slug[:32], "_")
	}
	if slug == "" {
		return "", errors.New("cannot derive a status key from that name; provide one explicitly")
	}
	return ValidateKey(slug)
}

func Ensure(ctx context.Context, q Querier, workspaceID pgtype.UUID) error {
	return q.SeedEntityStatusEntries(ctx, workspaceID)
}

func Effective(ctx context.Context, q Querier, workspaceID pgtype.UUID, resourceType, status string) string {
	if IsBuiltIn(status) {
		return status
	}
	entry, err := q.GetEntityStatusEntryByKey(ctx, db.GetEntityStatusEntryByKeyParams{
		WorkspaceID:  workspaceID,
		ResourceType: resourceType,
		Key:          status,
	})
	if err != nil {
		return status
	}
	if !IsCategory(entry.Category) {
		return status
	}
	return entry.Category
}

func Resolve(ctx context.Context, q Querier, workspaceID pgtype.UUID, resourceType, status string) (db.EntityStatus, error) {
	if !IsResourceType(resourceType) {
		return db.EntityStatus{}, ErrUnknownResource
	}
	key := strings.ToLower(strings.TrimSpace(status))
	if key == "" {
		return db.EntityStatus{}, ErrUnknownStatus
	}
	entry, err := q.GetEntityStatusEntryByKey(ctx, db.GetEntityStatusEntryByKeyParams{
		WorkspaceID:  workspaceID,
		ResourceType: resourceType,
		Key:          key,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if IsBuiltIn(key) {
				return builtInEntry(workspaceID, resourceType, key), nil
			}
			return db.EntityStatus{}, ErrUnknownStatus
		}
		return db.EntityStatus{}, err
	}
	if entry.ArchivedAt.Valid {
		return db.EntityStatus{}, ErrUnknownStatus
	}
	return entry, nil
}

func builtInEntry(workspaceID pgtype.UUID, resourceType, key string) db.EntityStatus {
	return db.EntityStatus{
		WorkspaceID:  workspaceID,
		ResourceType: resourceType,
		Key:          key,
		Category:     key,
		IsSystem:     true,
	}
}

func ActiveKeys(ctx context.Context, q Querier, workspaceID pgtype.UUID, resourceType string) ([]string, error) {
	entries, err := q.ListEntityStatusEntries(ctx, db.ListEntityStatusEntriesParams{
		WorkspaceID:     workspaceID,
		ResourceType:    resourceType,
		IncludeArchived: false,
	})
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(entries)+len(canonicalOrder))
	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		keys = append(keys, e.Key)
		seen[e.Key] = true
	}
	for _, key := range canonicalOrder {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	return keys, nil
}
