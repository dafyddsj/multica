package clerk

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// UserStore is the persistence surface Resolve needs. *db.Queries
// implements it; tests inject a fake.
type UserStore interface {
	GetUserByClerkID(ctx context.Context, clerkUserID pgtype.Text) (db.User, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	BindUserClerkID(ctx context.Context, arg db.BindUserClerkIDParams) (db.User, error)
}

// Identity is the Multica user a verified Clerk session maps to.
type Identity struct {
	UserID string
	Email  string
}

// Resolve verifies the Clerk JWT and upserts the local user. Clerk is the
// identity control plane: signup allowlists are not re-checked here.
// users.id stays a Multica UUID; the Clerk id is stored on clerk_user_id.
func (c *Client) Resolve(ctx context.Context, token string, store UserStore) (Identity, error) {
	if c == nil || c.Verifier == nil {
		return Identity{}, ErrInvalidSession
	}
	if store == nil {
		return Identity{}, ErrStoreRequired
	}

	clerkUserID, err := c.Verifier.Verify(ctx, token)
	if err != nil {
		return Identity{}, err
	}
	clerkUserID = strings.TrimSpace(clerkUserID)
	if clerkUserID == "" {
		return Identity{}, ErrInvalidSession
	}

	if user, err := store.GetUserByClerkID(ctx, util.StrToText(clerkUserID)); err == nil {
		return identityOf(user), nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Identity{}, err
	}

	if c.Profiles == nil {
		return Identity{}, ErrInvalidSession
	}
	profile, err := c.Profiles.Get(ctx, clerkUserID)
	if err != nil {
		return Identity{}, err
	}
	email := strings.ToLower(strings.TrimSpace(profile.Email))
	if email == "" {
		return Identity{}, ErrNoEmail
	}
	name := strings.TrimSpace(profile.Name)
	if name == "" {
		name = nameFromEmail(email)
	}

	existing, err := store.GetUserByEmail(ctx, email)
	if err == nil {
		bound, bindErr := store.BindUserClerkID(ctx, db.BindUserClerkIDParams{
			ID:          existing.ID,
			ClerkUserID: util.StrToText(clerkUserID),
		})
		if bindErr != nil {
			if errors.Is(bindErr, pgx.ErrNoRows) {
				return Identity{}, ErrBindConflict
			}
			return Identity{}, bindErr
		}
		return identityOf(bound), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Identity{}, err
	}

	created, err := store.CreateUser(ctx, db.CreateUserParams{
		Name:      name,
		Email:     email,
		AvatarUrl: util.StrToText(profile.AvatarURL),
	})
	if err != nil {
		if rebound, ok := lookupByClerkID(ctx, store, clerkUserID); ok {
			return rebound, nil
		}
		return Identity{}, err
	}

	bound, err := store.BindUserClerkID(ctx, db.BindUserClerkIDParams{
		ID:          created.ID,
		ClerkUserID: util.StrToText(clerkUserID),
	})
	if err != nil {
		if rebound, ok := lookupByClerkID(ctx, store, clerkUserID); ok {
			return rebound, nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return Identity{}, ErrBindConflict
		}
		return Identity{}, err
	}
	return identityOf(bound), nil
}

func lookupByClerkID(ctx context.Context, store UserStore, clerkUserID string) (Identity, bool) {
	user, err := store.GetUserByClerkID(ctx, util.StrToText(clerkUserID))
	if err != nil {
		return Identity{}, false
	}
	return identityOf(user), true
}

func identityOf(user db.User) Identity {
	return Identity{
		UserID: util.UUIDToString(user.ID),
		Email:  user.Email,
	}
}

func nameFromEmail(email string) string {
	if at := strings.Index(email, "@"); at > 0 {
		return email[:at]
	}
	return email
}
