package handler

import (
	"context"
	"errors"
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
func (stubHandlerOrgs) UpdateMember(context.Context, string, string, string) error {
	return nil
}

func TestGetMe_SyncsClerkOrganization(t *testing.T) {
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

	testutil.Call(t, testHandler.GetMe, newRequest("GET", "/api/me", nil)).Want(http.StatusOK)

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
}

type stubProfiles struct {
	email string
}

func (s stubProfiles) Get(context.Context, string) (clerk.Profile, error) {
	return clerk.Profile{Email: s.email, Name: "Ignored Clerk Name"}, nil
}

func TestGetMe_RefreshesClerkEmail(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	const clerkUserID = "user_handler_email_sync"
	const nextEmail = "clerk-refreshed@example.com"

	var previousClerk *string
	var previousEmail string
	_ = testPool.QueryRow(ctx, `SELECT clerk_user_id, email FROM "user" WHERE id = $1`, testUserID).Scan(&previousClerk, &previousEmail)
	_, _ = testPool.Exec(ctx, `UPDATE "user" SET clerk_user_id = $1 WHERE id = $2`, clerkUserID, testUserID)
	t.Cleanup(func() {
		if previousClerk == nil {
			_, _ = testPool.Exec(context.Background(), `UPDATE "user" SET email = $1, clerk_user_id = NULL WHERE id = $2`, previousEmail, testUserID)
		} else {
			_, _ = testPool.Exec(context.Background(), `UPDATE "user" SET email = $1, clerk_user_id = $2 WHERE id = $3`, previousEmail, *previousClerk, testUserID)
		}
	})

	orig := testHandler.Clerk
	testHandler.Clerk = &clerk.Client{
		Profiles: stubProfiles{email: "Clerk-Refreshed@example.com"},
		Orgs:     stubHandlerOrgs{},
	}
	t.Cleanup(func() { testHandler.Clerk = orig })

	var me UserResponse
	testutil.Call(t, testHandler.GetMe, newRequest("GET", "/api/me", nil)).Want(http.StatusOK).JSON(&me)
	if me.Email != nextEmail {
		t.Fatalf("GetMe email: got %q want %q", me.Email, nextEmail)
	}
	if me.Name == "Ignored Clerk Name" {
		t.Fatal("GetMe must not overwrite Multica name from Clerk")
	}
}

type explodingOrgs struct{}

func (explodingOrgs) ListMemberships(context.Context, string) ([]clerk.OrgMembership, error) {
	return nil, errors.New("clerk down")
}
func (explodingOrgs) Create(context.Context, string, string, string) (clerk.OrgRef, error) {
	return clerk.OrgRef{}, errors.New("clerk down")
}
func (explodingOrgs) Delete(context.Context, string) error { return errors.New("clerk down") }
func (explodingOrgs) RemoveMember(context.Context, string, string) error {
	return errors.New("clerk down")
}
func (explodingOrgs) AddMember(context.Context, string, string, string) error {
	return errors.New("clerk down")
}
func (explodingOrgs) UpdateMember(context.Context, string, string, string) error {
	return errors.New("clerk down")
}

func TestListWorkspaces_DoesNotSyncClerkOrganizations(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	_, _ = testPool.Exec(context.Background(), `UPDATE "user" SET clerk_user_id = $1 WHERE id = $2`, "user_list_no_sync", testUserID)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `UPDATE "user" SET clerk_user_id = NULL WHERE id = $1`, testUserID)
	})

	orig := testHandler.Clerk
	testHandler.Clerk = &clerk.Client{Orgs: explodingOrgs{}}
	t.Cleanup(func() { testHandler.Clerk = orig })

	testutil.Call(t, testHandler.ListWorkspaces, newRequest("GET", "/api/workspaces", nil)).
		Want(http.StatusOK)

	testutil.Call(t, testHandler.GetMe, newRequest("GET", "/api/me", nil)).
		Want(http.StatusBadGateway)
}

type recordingOrgs struct {
	created []string
	deleted []string
}

func (r *recordingOrgs) ListMemberships(context.Context, string) ([]clerk.OrgMembership, error) {
	return nil, nil
}
func (r *recordingOrgs) Create(_ context.Context, _, _, _ string) (clerk.OrgRef, error) {
	id := "org_compensating_delete"
	r.created = append(r.created, id)
	return clerk.OrgRef{ID: id, Name: "Compensating", Slug: "compensating"}, nil
}
func (r *recordingOrgs) Delete(_ context.Context, id string) error {
	r.deleted = append(r.deleted, id)
	return nil
}
func (*recordingOrgs) RemoveMember(context.Context, string, string) error { return nil }
func (*recordingOrgs) AddMember(context.Context, string, string, string) error {
	return nil
}
func (*recordingOrgs) UpdateMember(context.Context, string, string, string) error {
	return nil
}

func TestCreateWorkspace_DeletesClerkOrgWhenLocalCreateFails(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	const slug = "handler-clerk-compensate"
	ctx := context.Background()
	_, _ = testPool.Exec(ctx, `DELETE FROM workspace WHERE slug = $1`, slug)
	_, _ = testPool.Exec(ctx, `UPDATE "user" SET clerk_user_id = $1 WHERE id = $2`, "user_compensate", testUserID)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM workspace WHERE slug = $1`, slug)
		_, _ = testPool.Exec(context.Background(), `UPDATE "user" SET clerk_user_id = NULL WHERE id = $1`, testUserID)
	})

	testutil.Call(t, testHandler.CreateWorkspace, newRequest("POST", "/api/workspaces", map[string]any{
		"name": "First",
		"slug": slug,
	})).Want(http.StatusCreated)

	recs := &recordingOrgs{}
	orig := testHandler.Clerk
	testHandler.Clerk = &clerk.Client{Orgs: recs}
	t.Cleanup(func() { testHandler.Clerk = orig })

	testutil.Call(t, testHandler.CreateWorkspace, newRequest("POST", "/api/workspaces", map[string]any{
		"name": "Second",
		"slug": slug,
	})).Want(http.StatusConflict)

	if len(recs.created) != 1 || len(recs.deleted) != 1 || recs.deleted[0] != recs.created[0] {
		t.Fatalf("compensating delete: created=%v deleted=%v", recs.created, recs.deleted)
	}
}
