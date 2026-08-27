package clerk

import (
	"os"
	"strings"
)

const (
	envSecretKey      = "CLERK_SECRET_KEY"
	envPublishableKey = "CLERK_PUBLISHABLE_KEY"
)

// FromEnv returns a Clerk client when both secret and publishable keys are
// set. Either key missing (or blank) is the Redis-style no-op: callers treat
// nil as "use native auth". A single key is not enough — the API cannot
// verify sessions without the secret, and the web app cannot start Clerk
// without the publishable key.
func FromEnv() *Client {
	secret, publishable := keysFromEnv()
	if secret == "" || publishable == "" {
		return nil
	}
	return New(secret, publishable)
}

// PublishableKeyFromEnv returns the public Clerk key only when Clerk is
// fully configured. /api/config uses this so a half-set deployment does
// not advertise a sign-in mode the API cannot honour.
func PublishableKeyFromEnv() string {
	secret, publishable := keysFromEnv()
	if secret == "" || publishable == "" {
		return ""
	}
	return publishable
}

func keysFromEnv() (secret, publishable string) {
	return strings.TrimSpace(os.Getenv(envSecretKey)), strings.TrimSpace(os.Getenv(envPublishableKey))
}
