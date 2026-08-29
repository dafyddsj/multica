package clerk

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type stubVerifier struct {
	id  string
	err error
}

func (s stubVerifier) Verify(context.Context, string) (string, error) {
	return s.id, s.err
}

type stubProfiles struct {
	profile Profile
	err     error
}

func (s stubProfiles) Get(context.Context, string) (Profile, error) {
	return s.profile, s.err
}

type memoryStore struct {
	byClerk map[string]db.User
	byEmail map[string]db.User
	byID    map[string]db.User
}

func newMemoryStore(users ...db.User) *memoryStore {
	s := &memoryStore{
		byClerk: map[string]db.User{},
		byEmail: map[string]db.User{},
		byID:    map[string]db.User{},
	}
	for _, u := range users {
		s.put(u)
	}
	return s
}

func (s *memoryStore) put(u db.User) {
	id := util.UUIDToString(u.ID)
	s.byID[id] = u
	s.byEmail[u.Email] = u
	if u.ClerkUserID.Valid {
		s.byClerk[u.ClerkUserID.String] = u
	}
}

func (s *memoryStore) GetUserByClerkID(_ context.Context, clerkUserID pgtype.Text) (db.User, error) {
	if !clerkUserID.Valid {
		return db.User{}, pgx.ErrNoRows
	}
	u, ok := s.byClerk[clerkUserID.String]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (s *memoryStore) GetUserByEmail(_ context.Context, email string) (db.User, error) {
	u, ok := s.byEmail[email]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (s *memoryStore) CreateUser(_ context.Context, arg db.CreateUserParams) (db.User, error) {
	u := db.User{
		ID:        util.MustParseUUID("11111111-1111-1111-1111-111111111111"),
		Name:      arg.Name,
		Email:     arg.Email,
		AvatarUrl: arg.AvatarUrl,
	}
	s.put(u)
	return u, nil
}

func (s *memoryStore) BindUserClerkID(_ context.Context, arg db.BindUserClerkIDParams) (db.User, error) {
	id := util.UUIDToString(arg.ID)
	u, ok := s.byID[id]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	if u.ClerkUserID.Valid && u.ClerkUserID.String != arg.ClerkUserID.String {
		return db.User{}, pgx.ErrNoRows
	}
	u.ClerkUserID = arg.ClerkUserID
	s.put(u)
	return u, nil
}

func testUser(id, email, clerkID string) db.User {
	u := db.User{
		ID:    util.MustParseUUID(id),
		Email: email,
		Name:  "Ada",
	}
	if clerkID != "" {
		u.ClerkUserID = util.StrToText(clerkID)
	}
	return u
}

func TestResolveMappedClerkUser(t *testing.T) {
	existing := testUser("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "ada@example.com", "user_clerk")
	c := &Client{Verifier: stubVerifier{id: "user_clerk"}}
	got, err := c.Resolve(context.Background(), "tok", newMemoryStore(existing))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.UserID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("user id: got %q", got.UserID)
	}
	if got.Email != "ada@example.com" {
		t.Fatalf("email: got %q", got.Email)
	}
}

func TestResolveBindsExistingEmail(t *testing.T) {
	existing := testUser("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "ada@example.com", "")
	store := newMemoryStore(existing)
	c := &Client{
		Verifier: stubVerifier{id: "user_new"},
		Profiles: stubProfiles{profile: Profile{Email: "Ada@example.com", Name: "Ada Lovelace"}},
	}
	got, err := c.Resolve(context.Background(), "tok", store)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.UserID != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("user id: got %q", got.UserID)
	}
	bound, err := store.GetUserByClerkID(context.Background(), util.StrToText("user_new"))
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	if bound.ClerkUserID.String != "user_new" {
		t.Fatalf("clerk id: got %q", bound.ClerkUserID.String)
	}
}

func TestResolveCreatesUser(t *testing.T) {
	store := newMemoryStore()
	c := &Client{
		Verifier: stubVerifier{id: "user_fresh"},
		Profiles: stubProfiles{profile: Profile{Email: "new@example.com", Name: "New User", AvatarURL: "https://img"}},
	}
	got, err := c.Resolve(context.Background(), "tok", store)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.UserID == "" || got.Email != "new@example.com" {
		t.Fatalf("created identity: %+v", got)
	}
	bound, err := store.GetUserByClerkID(context.Background(), util.StrToText("user_fresh"))
	if err != nil {
		t.Fatalf("created bind: %v", err)
	}
	if bound.Name != "New User" {
		t.Fatalf("name: got %q", bound.Name)
	}
}

func TestResolveRejectsMissingEmail(t *testing.T) {
	c := &Client{
		Verifier: stubVerifier{id: "user_no_email"},
		Profiles: stubProfiles{profile: Profile{Name: "No Email"}},
	}
	_, err := c.Resolve(context.Background(), "tok", newMemoryStore())
	if !errors.Is(err, ErrNoEmail) {
		t.Fatalf("want ErrNoEmail, got %v", err)
	}
}

func TestResolveRejectsInvalidSession(t *testing.T) {
	c := &Client{Verifier: stubVerifier{err: ErrInvalidSession}}
	_, err := c.Resolve(context.Background(), "tok", newMemoryStore())
	if !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("want ErrInvalidSession, got %v", err)
	}
}

func TestResolveNilClient(t *testing.T) {
	var c *Client
	_, err := c.Resolve(context.Background(), "tok", newMemoryStore())
	if !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("want ErrInvalidSession, got %v", err)
	}
}
