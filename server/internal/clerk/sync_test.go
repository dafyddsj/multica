package clerk

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type stubOrgs struct {
	memberships []OrgMembership
	err         error
}

func (s stubOrgs) ListMemberships(context.Context, string) ([]OrgMembership, error) {
	return s.memberships, s.err
}
func (stubOrgs) Create(context.Context, string, string, string) (OrgRef, error) {
	return OrgRef{}, nil
}
func (stubOrgs) Delete(context.Context, string) error { return nil }
func (stubOrgs) RemoveMember(context.Context, string, string) error {
	return nil
}
func (stubOrgs) AddMember(context.Context, string, string, string) error {
	return nil
}
func (stubOrgs) UpdateMember(context.Context, string, string, string) error {
	return nil
}

type memoryOrgStore struct {
	workspaces []db.Workspace
	members    []db.Member
	seeded     []string
	nextWS     int
	nextMem    int
}

func (s *memoryOrgStore) GetWorkspaceByClerkOrgID(_ context.Context, clerkOrgID pgtype.Text) (db.Workspace, error) {
	if !clerkOrgID.Valid {
		return db.Workspace{}, pgx.ErrNoRows
	}
	for _, ws := range s.workspaces {
		if ws.ClerkOrgID.Valid && ws.ClerkOrgID.String == clerkOrgID.String {
			return ws, nil
		}
	}
	return db.Workspace{}, pgx.ErrNoRows
}

func (s *memoryOrgStore) BindWorkspaceClerkOrgID(_ context.Context, arg db.BindWorkspaceClerkOrgIDParams) (db.Workspace, error) {
	for i, ws := range s.workspaces {
		if util.UUIDToString(ws.ID) != util.UUIDToString(arg.ID) {
			continue
		}
		if ws.ClerkOrgID.Valid && ws.ClerkOrgID.String != arg.ClerkOrgID.String {
			return db.Workspace{}, pgx.ErrNoRows
		}
		s.workspaces[i].ClerkOrgID = arg.ClerkOrgID
		return s.workspaces[i], nil
	}
	return db.Workspace{}, pgx.ErrNoRows
}

func (s *memoryOrgStore) CreateWorkspace(_ context.Context, arg db.CreateWorkspaceParams) (db.Workspace, error) {
	for _, ws := range s.workspaces {
		if ws.Slug == arg.Slug {
			return db.Workspace{}, &pgconn.PgError{Code: "23505"}
		}
	}
	s.nextWS++
	ws := db.Workspace{
		ID:          util.MustParseUUID(fmt.Sprintf("aaaaaaaa-aaaa-aaaa-aaaa-%012d", s.nextWS)),
		Name:        arg.Name,
		Slug:        arg.Slug,
		IssuePrefix: arg.IssuePrefix,
	}
	s.workspaces = append(s.workspaces, ws)
	return ws, nil
}

func (s *memoryOrgStore) CreateMember(_ context.Context, arg db.CreateMemberParams) (db.Member, error) {
	s.nextMem++
	m := db.Member{
		ID:          util.MustParseUUID(fmt.Sprintf("bbbbbbbb-bbbb-bbbb-bbbb-%012d", s.nextMem)),
		WorkspaceID: arg.WorkspaceID,
		UserID:      arg.UserID,
		Role:        arg.Role,
	}
	s.members = append(s.members, m)
	return m, nil
}

func (s *memoryOrgStore) GetMemberByUserAndWorkspace(_ context.Context, arg db.GetMemberByUserAndWorkspaceParams) (db.Member, error) {
	for _, m := range s.members {
		if util.UUIDToString(m.UserID) == util.UUIDToString(arg.UserID) &&
			util.UUIDToString(m.WorkspaceID) == util.UUIDToString(arg.WorkspaceID) {
			return m, nil
		}
	}
	return db.Member{}, pgx.ErrNoRows
}

func (s *memoryOrgStore) UpdateMemberRole(_ context.Context, arg db.UpdateMemberRoleParams) (db.Member, error) {
	for i, m := range s.members {
		if util.UUIDToString(m.ID) == util.UUIDToString(arg.ID) {
			s.members[i].Role = arg.Role
			return s.members[i], nil
		}
	}
	return db.Member{}, pgx.ErrNoRows
}

func (s *memoryOrgStore) DeleteMember(_ context.Context, id pgtype.UUID) error {
	kept := s.members[:0]
	for _, m := range s.members {
		if util.UUIDToString(m.ID) != util.UUIDToString(id) {
			kept = append(kept, m)
		}
	}
	s.members = kept
	return nil
}

func (s *memoryOrgStore) ListClerkMappedWorkspacesForUser(_ context.Context, userID pgtype.UUID) ([]db.ListClerkMappedWorkspacesForUserRow, error) {
	var out []db.ListClerkMappedWorkspacesForUserRow
	for _, m := range s.members {
		if util.UUIDToString(m.UserID) != util.UUIDToString(userID) {
			continue
		}
		for _, ws := range s.workspaces {
			if util.UUIDToString(ws.ID) == util.UUIDToString(m.WorkspaceID) && ws.ClerkOrgID.Valid {
				out = append(out, db.ListClerkMappedWorkspacesForUserRow{
					ID:         ws.ID,
					ClerkOrgID: ws.ClerkOrgID,
					MemberID:   m.ID,
					Role:       m.Role,
				})
			}
		}
	}
	return out, nil
}

func (s *memoryOrgStore) CountWorkspaceOwners(_ context.Context, workspaceID pgtype.UUID) (int32, error) {
	var n int32
	for _, m := range s.members {
		if util.UUIDToString(m.WorkspaceID) == util.UUIDToString(workspaceID) && m.Role == "owner" {
			n++
		}
	}
	return n, nil
}

func (s *memoryOrgStore) SeedIssueStatusEntries(_ context.Context, workspaceID pgtype.UUID) error {
	s.seeded = append(s.seeded, util.UUIDToString(workspaceID))
	return nil
}

func testUserID() pgtype.UUID {
	return util.MustParseUUID("11111111-1111-1111-1111-111111111111")
}

func TestSyncOrgsCreatesWorkspaceAndOwner(t *testing.T) {
	store := &memoryOrgStore{}
	c := &Client{Orgs: stubOrgs{memberships: []OrgMembership{{
		Org:  OrgRef{ID: "org_test", Name: "Test Org", Slug: "test-org"},
		Role: "org:admin",
	}}}}
	if err := c.SyncOrgs(context.Background(), "user_clerk", testUserID(), store, nil); err != nil {
		t.Fatalf("SyncOrgs: %v", err)
	}
	if len(store.workspaces) != 1 {
		t.Fatalf("workspaces: %d", len(store.workspaces))
	}
	ws := store.workspaces[0]
	if ws.Name != "Test Org" || ws.Slug != "test-org" || ws.ClerkOrgID.String != "org_test" {
		t.Fatalf("workspace: %+v", ws)
	}
	if len(store.members) != 1 || store.members[0].Role != "owner" {
		t.Fatalf("members: %+v", store.members)
	}
	if len(store.seeded) != 1 {
		t.Fatalf("statuses not seeded")
	}
}

func TestSyncOrgsIsIdempotent(t *testing.T) {
	store := &memoryOrgStore{}
	c := &Client{Orgs: stubOrgs{memberships: []OrgMembership{{
		Org:  OrgRef{ID: "org_test", Name: "Test Org", Slug: "test-org"},
		Role: "org:admin",
	}}}}
	ctx := context.Background()
	if err := c.SyncOrgs(ctx, "user_clerk", testUserID(), store, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.SyncOrgs(ctx, "user_clerk", testUserID(), store, nil); err != nil {
		t.Fatal(err)
	}
	if len(store.workspaces) != 1 || len(store.members) != 1 {
		t.Fatalf("duplicate rows: ws=%d mem=%d", len(store.workspaces), len(store.members))
	}
}

func TestSyncOrgsUpdatesRoleAndRemovesLeftOrg(t *testing.T) {
	userID := testUserID()
	native := db.Workspace{
		ID:   util.MustParseUUID("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		Name: "Native",
		Slug: "native",
	}
	mapped := db.Workspace{
		ID:         util.MustParseUUID("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		Name:       "Old Org",
		Slug:       "old-org",
		ClerkOrgID: util.StrToText("org_old"),
	}
	store := &memoryOrgStore{
		workspaces: []db.Workspace{native, mapped},
		members: []db.Member{
			{ID: util.MustParseUUID("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"), WorkspaceID: native.ID, UserID: userID, Role: "owner"},
			{ID: util.MustParseUUID("ffffffff-ffff-ffff-ffff-ffffffffffff"), WorkspaceID: mapped.ID, UserID: userID, Role: "owner"},
			{ID: util.MustParseUUID("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"), WorkspaceID: mapped.ID, UserID: util.MustParseUUID("22222222-2222-2222-2222-222222222222"), Role: "owner"},
		},
	}
	c := &Client{Orgs: stubOrgs{memberships: []OrgMembership{{
		Org:  OrgRef{ID: "org_old", Name: "Old Org", Slug: "old-org"},
		Role: "org:member",
	}}}}
	if err := c.SyncOrgs(context.Background(), "user_clerk", userID, store, nil); err != nil {
		t.Fatal(err)
	}
	if store.members[1].Role != "member" {
		t.Fatalf("role: %q", store.members[1].Role)
	}

	c.Orgs = stubOrgs{memberships: nil}
	if err := c.SyncOrgs(context.Background(), "user_clerk", userID, store, nil); err != nil {
		t.Fatal(err)
	}
	if len(store.members) != 2 {
		t.Fatalf("mapped membership should be gone, got %+v", store.members)
	}
	for _, m := range store.members {
		if util.UUIDToString(m.UserID) == util.UUIDToString(userID) &&
			util.UUIDToString(m.WorkspaceID) == util.UUIDToString(mapped.ID) {
			t.Fatalf("leaver still on mapped workspace: %+v", m)
		}
	}
	if len(store.workspaces) != 2 {
		t.Fatalf("workspaces should stay, got %d", len(store.workspaces))
	}
}

func TestSyncOrgsKeepsLastMappedOwner(t *testing.T) {
	userID := testUserID()
	mapped := db.Workspace{
		ID:         util.MustParseUUID("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		Name:       "Old Org",
		Slug:       "old-org",
		ClerkOrgID: util.StrToText("org_old"),
	}
	store := &memoryOrgStore{
		workspaces: []db.Workspace{mapped},
		members: []db.Member{{
			ID:          util.MustParseUUID("ffffffff-ffff-ffff-ffff-ffffffffffff"),
			WorkspaceID: mapped.ID,
			UserID:      userID,
			Role:        "owner",
		}},
	}
	c := &Client{Orgs: stubOrgs{memberships: nil}}
	if err := c.SyncOrgs(context.Background(), "user_clerk", userID, store, nil); err != nil {
		t.Fatal(err)
	}
	if len(store.members) != 1 || store.members[0].Role != "owner" {
		t.Fatalf("last owner was removed: %+v", store.members)
	}
}

func TestSyncOrgsKeepsLocalAdminUnderClerkAdmin(t *testing.T) {
	userID := testUserID()
	mapped := db.Workspace{
		ID:         util.MustParseUUID("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		Name:       "Old Org",
		Slug:       "old-org",
		ClerkOrgID: util.StrToText("org_old"),
	}
	store := &memoryOrgStore{
		workspaces: []db.Workspace{mapped},
		members: []db.Member{{
			ID:          util.MustParseUUID("ffffffff-ffff-ffff-ffff-ffffffffffff"),
			WorkspaceID: mapped.ID,
			UserID:      userID,
			Role:        "admin",
		}},
	}
	c := &Client{Orgs: stubOrgs{memberships: []OrgMembership{{
		Org:  OrgRef{ID: "org_old", Name: "Old Org", Slug: "old-org"},
		Role: "org:admin",
	}}}}
	if err := c.SyncOrgs(context.Background(), "user_clerk", userID, store, nil); err != nil {
		t.Fatal(err)
	}
	if store.members[0].Role != "admin" {
		t.Fatalf("admin collapsed to %q", store.members[0].Role)
	}
}

func TestSyncOrgsNoopsWithoutDirectory(t *testing.T) {
	store := &memoryOrgStore{}
	c := &Client{}
	if err := c.SyncOrgs(context.Background(), "user_clerk", testUserID(), store, nil); err != nil {
		t.Fatal(err)
	}
	if len(store.workspaces) != 0 {
		t.Fatalf("created workspaces without Orgs")
	}
}
