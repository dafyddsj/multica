package clerk

import (
	"context"
	"errors"
)

var (
	ErrInvalidSession   = errors.New("clerk: invalid session")
	ErrNoEmail          = errors.New("clerk: user has no email")
	ErrBindConflict     = errors.New("clerk: user already bound to a different clerk id")
	ErrStoreRequired    = errors.New("clerk: user store is required")
	ErrOrgStoreRequired = errors.New("clerk: org store is required")
	ErrSlugExhausted    = errors.New("clerk: could not allocate a workspace slug")
	ErrOrgRequired      = errors.New("clerk: organization directory is required")
)

// Verifier checks a Clerk session JWT and returns the Clerk user id (sub).
// Tests inject a fake; production uses the Clerk SDK.
type Verifier interface {
	Verify(ctx context.Context, token string) (clerkUserID string, err error)
}

// Profile is the subset of a Clerk user we persist locally.
type Profile struct {
	ID        string
	Email     string
	Name      string
	AvatarURL string
}

// ProfileFetcher loads display fields from Clerk after a first-seen session.
type ProfileFetcher interface {
	Get(ctx context.Context, clerkUserID string) (Profile, error)
}

// Client is the env-gated Clerk overlay. Nil clients mean native auth.
type Client struct {
	PublishableKey string
	Verifier       Verifier
	Profiles       ProfileFetcher
	Orgs           OrgDirectory
}

// New builds a production client that talks to Clerk. Tests construct
// Client with fake Verifier/Profiles instead of calling New.
func New(secretKey, publishableKey string) *Client {
	configureSDK(secretKey)
	return &Client{
		PublishableKey: publishableKey,
		Verifier:       sdkVerifier{},
		Profiles:       sdkProfiles{},
		Orgs:           sdkOrgs{},
	}
}
