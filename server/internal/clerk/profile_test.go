package clerk

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type memoryEmailStore struct {
	users map[string]db.User
	fail  error
}

func (s *memoryEmailStore) UpdateUserEmail(_ context.Context, arg db.UpdateUserEmailParams) (db.User, error) {
	if s.fail != nil {
		return db.User{}, s.fail
	}
	id := util.UUIDToString(arg.ID)
	u, ok := s.users[id]
	if !ok {
		return db.User{}, errors.New("missing user")
	}
	for _, other := range s.users {
		if other.Email == arg.Email && util.UUIDToString(other.ID) != id {
			return db.User{}, &pgconn.PgError{Code: "23505"}
		}
	}
	u.Email = arg.Email
	s.users[id] = u
	return u, nil
}

func TestSyncProfileUpdatesEmail(t *testing.T) {
	user := testUser("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "old@example.com", "user_clerk")
	store := &memoryEmailStore{users: map[string]db.User{util.UUIDToString(user.ID): user}}
	c := &Client{Profiles: stubProfiles{profile: Profile{Email: "New@example.com", Name: "Ignored"}}}

	got, err := c.SyncProfile(context.Background(), user, store)
	if err != nil {
		t.Fatalf("SyncProfile: %v", err)
	}
	if got.Email != "new@example.com" {
		t.Fatalf("email: got %q", got.Email)
	}
	if got.Name != user.Name {
		t.Fatalf("name must stay Multica-owned: got %q", got.Name)
	}
}

func TestSyncProfileNoopsWhenEmailMatches(t *testing.T) {
	user := testUser("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "ada@example.com", "user_clerk")
	store := &memoryEmailStore{users: map[string]db.User{util.UUIDToString(user.ID): user}}
	c := &Client{Profiles: stubProfiles{profile: Profile{Email: "Ada@example.com"}}}

	got, err := c.SyncProfile(context.Background(), user, store)
	if err != nil {
		t.Fatalf("SyncProfile: %v", err)
	}
	if got.Email != "ada@example.com" {
		t.Fatalf("email: got %q", got.Email)
	}
}

func TestSyncProfileNoopsWithoutClerkID(t *testing.T) {
	user := testUser("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "ada@example.com", "")
	c := &Client{Profiles: stubProfiles{profile: Profile{Email: "other@example.com"}}}
	got, err := c.SyncProfile(context.Background(), user, &memoryEmailStore{})
	if err != nil {
		t.Fatalf("SyncProfile: %v", err)
	}
	if got.Email != "ada@example.com" {
		t.Fatalf("unbound user must not change: %q", got.Email)
	}
}

func TestSyncProfileConflict(t *testing.T) {
	user := testUser("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "old@example.com", "user_clerk")
	other := testUser("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "taken@example.com", "")
	store := &memoryEmailStore{users: map[string]db.User{
		util.UUIDToString(user.ID):  user,
		util.UUIDToString(other.ID): other,
	}}
	c := &Client{Profiles: stubProfiles{profile: Profile{Email: "taken@example.com"}}}

	_, err := c.SyncProfile(context.Background(), user, store)
	if !errors.Is(err, ErrBindConflict) {
		t.Fatalf("want ErrBindConflict, got %v", err)
	}
}

func TestSyncProfileNilClient(t *testing.T) {
	var c *Client
	user := testUser("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "ada@example.com", "user_clerk")
	got, err := c.SyncProfile(context.Background(), user, &memoryEmailStore{})
	if err != nil || got.Email != "ada@example.com" {
		t.Fatalf("nil client: %v %+v", err, got)
	}
}

func TestSyncProfileRequiresStore(t *testing.T) {
	c := &Client{Profiles: stubProfiles{profile: Profile{Email: "n@example.com"}}}
	user := testUser("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "ada@example.com", "user_clerk")
	_, err := c.SyncProfile(context.Background(), user, nil)
	if !errors.Is(err, ErrStoreRequired) {
		t.Fatalf("got %v", err)
	}
}
