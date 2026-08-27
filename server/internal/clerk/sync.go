package clerk

import (
	"context"
	"errors"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// OrgStore is the persistence surface SyncOrgs needs. *db.Queries implements it.
type OrgStore interface {
	GetWorkspaceByClerkOrgID(ctx context.Context, clerkOrgID pgtype.Text) (db.Workspace, error)
	BindWorkspaceClerkOrgID(ctx context.Context, arg db.BindWorkspaceClerkOrgIDParams) (db.Workspace, error)
	CreateWorkspace(ctx context.Context, arg db.CreateWorkspaceParams) (db.Workspace, error)
	CreateMember(ctx context.Context, arg db.CreateMemberParams) (db.Member, error)
	GetMemberByUserAndWorkspace(ctx context.Context, arg db.GetMemberByUserAndWorkspaceParams) (db.Member, error)
	UpdateMemberRole(ctx context.Context, arg db.UpdateMemberRoleParams) (db.Member, error)
	DeleteMember(ctx context.Context, id pgtype.UUID) error
	ListClerkMappedWorkspacesForUser(ctx context.Context, userID pgtype.UUID) ([]db.ListClerkMappedWorkspacesForUserRow, error)
	SeedIssueStatusEntries(ctx context.Context, workspaceID pgtype.UUID) error
}

// SyncOrgs reconciles the user's Clerk organization memberships into Multica
// workspaces. It is idempotent. Workspaces with a null clerk_org_id are left
// alone. Mapped workspaces the user left in Clerk lose this user's membership.
func (c *Client) SyncOrgs(ctx context.Context, clerkUserID string, userID pgtype.UUID, store OrgStore, reserved func(string) bool) error {
	if c == nil || c.Orgs == nil {
		return nil
	}
	if store == nil {
		return ErrOrgStoreRequired
	}
	clerkUserID = strings.TrimSpace(clerkUserID)
	if clerkUserID == "" {
		return nil
	}

	memberships, err := c.Orgs.ListMemberships(ctx, clerkUserID)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(memberships))
	for _, membership := range memberships {
		orgID := strings.TrimSpace(membership.Org.ID)
		if orgID == "" {
			continue
		}
		seen[orgID] = struct{}{}
		if err := upsertOrgWorkspace(ctx, store, userID, membership.Org, MapClerkRole(membership.Role), reserved); err != nil {
			return err
		}
	}

	mapped, err := store.ListClerkMappedWorkspacesForUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, row := range mapped {
		if !row.ClerkOrgID.Valid {
			continue
		}
		if _, ok := seen[row.ClerkOrgID.String]; ok {
			continue
		}
		if err := store.DeleteMember(ctx, row.MemberID); err != nil {
			return err
		}
	}
	return nil
}

func upsertOrgWorkspace(ctx context.Context, store OrgStore, userID pgtype.UUID, org OrgRef, role string, reserved func(string) bool) error {
	if ws, err := store.GetWorkspaceByClerkOrgID(ctx, util.StrToText(org.ID)); err == nil {
		return ensureMember(ctx, store, ws.ID, userID, role)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	base := WorkspaceSlug(org.Name, org.Slug, org.ID, reserved)
	for attempt := 0; attempt < 8; attempt++ {
		slug := nextSlugAttempt(base, attempt, org.ID)
		if reserved != nil && reserved(slug) {
			continue
		}
		created, err := store.CreateWorkspace(ctx, db.CreateWorkspaceParams{
			Name:        orgName(org),
			Slug:        slug,
			IssuePrefix: issuePrefixFromSlug(slug),
		})
		if err != nil {
			if !isUniqueViolation(err) {
				return err
			}
			if existing, ok := lookupWorkspaceByOrg(ctx, store, org.ID); ok {
				return ensureMember(ctx, store, existing.ID, userID, role)
			}
			continue
		}

		bound, err := store.BindWorkspaceClerkOrgID(ctx, db.BindWorkspaceClerkOrgIDParams{
			ID:         created.ID,
			ClerkOrgID: util.StrToText(org.ID),
		})
		if err != nil {
			if existing, ok := lookupWorkspaceByOrg(ctx, store, org.ID); ok {
				return ensureMember(ctx, store, existing.ID, userID, role)
			}
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrBindConflict
			}
			return err
		}
		if err := store.SeedIssueStatusEntries(ctx, bound.ID); err != nil {
			return err
		}
		return ensureMember(ctx, store, bound.ID, userID, role)
	}
	return ErrSlugExhausted
}

func ensureMember(ctx context.Context, store OrgStore, workspaceID, userID pgtype.UUID, role string) error {
	member, err := store.GetMemberByUserAndWorkspace(ctx, db.GetMemberByUserAndWorkspaceParams{
		UserID:      userID,
		WorkspaceID: workspaceID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = store.CreateMember(ctx, db.CreateMemberParams{
			WorkspaceID: workspaceID,
			UserID:      userID,
			Role:        role,
		})
		return err
	}
	if err != nil {
		return err
	}
	if member.Role == role {
		return nil
	}
	_, err = store.UpdateMemberRole(ctx, db.UpdateMemberRoleParams{
		ID:   member.ID,
		Role: role,
	})
	return err
}

func lookupWorkspaceByOrg(ctx context.Context, store OrgStore, orgID string) (db.Workspace, bool) {
	ws, err := store.GetWorkspaceByClerkOrgID(ctx, util.StrToText(orgID))
	if err != nil {
		return db.Workspace{}, false
	}
	return ws, true
}

func orgName(org OrgRef) string {
	name := strings.TrimSpace(org.Name)
	if name != "" {
		return name
	}
	if slug := strings.TrimSpace(org.Slug); slug != "" {
		return slug
	}
	return "Organization"
}

func issuePrefixFromSlug(slug string) string {
	var b strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	head := b.String()
	if head == "" {
		return "WS"
	}
	if len(head) > 4 {
		return head[:4]
	}
	return head
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
