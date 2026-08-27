package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/multica-ai/multica/server/internal/clerk"
	"github.com/multica-ai/multica/server/internal/testutil"
)

type stubHandlerOrgs struct {
	memberships []clerk.OrgMembership
}

func (s stubHandlerOrgs) ListMemberships(context.Context, string) ([]clerk.OrgMembership, error) {
	return s.memberships, nil
}
func (stubHandlerOrgs) Create(context.Context, string, string, string) (clerk.OrgRef, error) {
	return clerk.OrgRef{}, nil
}
func (stubHandlerOrgs) Delete(context.Context, string) error { return nil }
func (stubHandlerOrgs) RemoveMember(context.Context, string, string) error {
	return nil
}
func (stubHandlerOrgs) AddMember(context.Context, string, string, string) error {
	return nil
}

func TestListWorkspaces_SyncsClerkOrganization(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	const orgID = "org_handler_sync_test"
	const slug = "handler-clerk-sync"
	const clerkUserID = "user_handler_sync_test"

	_, _ = testPool.Exec(ctx, `DELETE FROM workspace WHERE slug = $1 OR clerk_org_id = $2`, slug, orgID)
	var previousClerk *string
	_ = testPool.QueryRow(ctx, `SELECT clerk_user_id FROM "user" WHERE id = $1`, testUserID).Scan(&previousClerk)
	_, _ = testPool.Exec(ctx, `UPDATE "user" SET clerk_user_id = $1 WHERE id = $2`, clerkUserID, testUserID)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM workspace WHERE slug = $1 OR clerk_org_id = $2`, slug, orgID)
		if previousClerk == nil {
			_, _ = testPool.Exec(context.Background(), `UPDATE "user" SET clerk_user_id = NULL WHERE id = $1`, testUserID)
		} else {
			_, _ = testPool.Exec(context.Background(), `UPDATE "user" SET clerk_user_id = $1 WHERE id = $2`, *previousClerk, testUserID)
		}
	})

	orig := testHandler.Clerk
	testHandler.Clerk = &clerk.Client{Orgs: stubHandlerOrgs{memberships: []clerk.OrgMembership{{
		Org:  clerk.OrgRef{ID: orgID, Name: "Handler Clerk Org", Slug: slug},
		Role: "org:admin",
	}}}}
	t.Cleanup(func() { testHandler.Clerk = orig })

	var got []WorkspaceResponse
	testutil.Call(t, testHandler.ListWorkspaces, newRequest("GET", "/api/workspaces", nil)).
		Want(http.StatusOK).JSON(&got)

	found := false
	for _, ws := range got {
		if ws.Slug == slug && ws.Name == "Handler Clerk Org" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("synced workspace missing from list: %+v", got)
	}

	var bound string
	dbfx.QueryRow(t, `SELECT clerk_org_id FROM workspace WHERE slug = $1`, slug).Scan(&bound)
	if bound != orgID {
		t.Fatalf("clerk_org_id: got %q want %q", bound, orgID)
	}

	testutil.Call(t, testHandler.GetMe, newRequest("GET", "/api/me", nil)).Want(http.StatusOK)
}
