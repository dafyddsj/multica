package clerk

import (
	"context"
	"strings"

	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// EmailStore writes the Clerk primary email onto a bound Multica user.
// *db.Queries implements it; tests inject a fake.
type EmailStore interface {
	UpdateUserEmail(ctx context.Context, arg db.UpdateUserEmailParams) (db.User, error)
}

// SyncProfile copies the current Clerk primary email onto a bound user.
// It does not run on every Resolve: that is the per-request auth path
// and must not call Clerk Users.Get. GetMe is the refresh point, same
// as SyncOrgs.
//
// Name and avatar are left alone inbound. After first bind those are
// Multica profile fields. Avatar writes from UpdateMe / UpdateWorkspace
// push a real image to Clerk separately (see handler/clerk_avatars.go).
func (c *Client) SyncProfile(ctx context.Context, user db.User, store EmailStore) (db.User, error) {
	if c == nil || c.Profiles == nil {
		return user, nil
	}
	if store == nil {
		return user, ErrStoreRequired
	}
	if !user.ClerkUserID.Valid || user.ClerkUserID.String == "" {
		return user, nil
	}

	profile, err := c.Profiles.Get(ctx, user.ClerkUserID.String)
	if err != nil {
		return user, err
	}
	email := strings.ToLower(strings.TrimSpace(profile.Email))
	if email == "" {
		return user, nil
	}
	if email == strings.ToLower(strings.TrimSpace(user.Email)) {
		return user, nil
	}

	updated, err := store.UpdateUserEmail(ctx, db.UpdateUserEmailParams{
		ID:    user.ID,
		Email: email,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return user, ErrBindConflict
		}
		return user, err
	}
	return updated, nil
}
