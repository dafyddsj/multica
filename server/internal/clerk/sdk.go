package clerk

import (
	"context"
	"strings"

	clerksdk "github.com/clerk/clerk-sdk-go/v2"
	clerkjwt "github.com/clerk/clerk-sdk-go/v2/jwt"
	clerkuser "github.com/clerk/clerk-sdk-go/v2/user"
)

func configureSDK(secretKey string) {
	clerksdk.SetKey(secretKey)
}

type sdkVerifier struct{}

func (sdkVerifier) Verify(ctx context.Context, token string) (string, error) {
	claims, err := clerkjwt.Verify(ctx, &clerkjwt.VerifyParams{Token: token})
	if err != nil {
		return "", err
	}
	if claims == nil || strings.TrimSpace(claims.Subject) == "" {
		return "", ErrInvalidSession
	}
	return claims.Subject, nil
}

type sdkProfiles struct{}

func (sdkProfiles) Get(ctx context.Context, clerkUserID string) (Profile, error) {
	u, err := clerkuser.Get(ctx, clerkUserID)
	if err != nil {
		return Profile{}, err
	}
	if u == nil {
		return Profile{}, ErrInvalidSession
	}
	return Profile{
		ID:        u.ID,
		Email:     primaryEmail(u),
		Name:      displayName(u),
		AvatarURL: stringOrEmpty(u.ImageURL),
	}, nil
}

func primaryEmail(u *clerksdk.User) string {
	if u.PrimaryEmailAddressID != nil {
		for _, addr := range u.EmailAddresses {
			if addr != nil && addr.ID == *u.PrimaryEmailAddressID {
				return addr.EmailAddress
			}
		}
	}
	for _, addr := range u.EmailAddresses {
		if addr != nil && addr.EmailAddress != "" {
			return addr.EmailAddress
		}
	}
	return ""
}

func displayName(u *clerksdk.User) string {
	first := strings.TrimSpace(stringOrEmpty(u.FirstName))
	last := strings.TrimSpace(stringOrEmpty(u.LastName))
	switch {
	case first != "" && last != "":
		return first + " " + last
	case first != "":
		return first
	case last != "":
		return last
	}
	return strings.TrimSpace(stringOrEmpty(u.Username))
}

func stringOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
