package clerk

import (
	"context"
	"errors"
	"strings"

	clerksdk "github.com/clerk/clerk-sdk-go/v2"
	clerkjwt "github.com/clerk/clerk-sdk-go/v2/jwt"
	clerkorg "github.com/clerk/clerk-sdk-go/v2/organization"
	clerkorgmem "github.com/clerk/clerk-sdk-go/v2/organizationmembership"
	clerkuser "github.com/clerk/clerk-sdk-go/v2/user"
)

type sdkImages struct{}

func (sdkImages) UpdateProfileImage(ctx context.Context, clerkUserID string, file ImageFile) error {
	_, err := clerkuser.UpdateProfileImage(ctx, clerkUserID, &clerkuser.UpdateProfileImageParams{
		File: file.File(),
	})
	return err
}

func (sdkImages) DeleteProfileImage(ctx context.Context, clerkUserID string) error {
	_, err := clerkuser.DeleteProfileImage(ctx, clerkUserID)
	return err
}

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

func IsNotFound(err error) bool {
	var apiErr *clerksdk.APIErrorResponse
	return errors.As(err, &apiErr) && apiErr.HTTPStatusCode == 404
}

func stringOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

type sdkOrgs struct{}

func (sdkOrgs) ListMemberships(ctx context.Context, clerkUserID string) ([]OrgMembership, error) {
	var out []OrgMembership
	var offset int64
	const page int64 = 100
	for {
		limit := page
		off := offset
		list, err := clerkuser.ListOrganizationMemberships(ctx, clerkUserID, &clerkuser.ListOrganizationMembershipsParams{
			ListParams: clerksdk.ListParams{Limit: &limit, Offset: &off},
		})
		if err != nil {
			return nil, err
		}
		if list == nil {
			break
		}
		for _, m := range list.OrganizationMemberships {
			if m == nil || m.Organization == nil || strings.TrimSpace(m.Organization.ID) == "" {
				continue
			}
			out = append(out, OrgMembership{
				Org: OrgRef{
					ID:   m.Organization.ID,
					Name: m.Organization.Name,
					Slug: m.Organization.Slug,
				},
				Role: m.Role,
			})
		}
		if int64(len(list.OrganizationMemberships)) < page {
			break
		}
		offset += page
	}
	return out, nil
}

func (sdkOrgs) Create(ctx context.Context, name, slug, createdBy string) (OrgRef, error) {
	org, err := clerkorg.Create(ctx, &clerkorg.CreateParams{
		Name:      &name,
		Slug:      &slug,
		CreatedBy: &createdBy,
	})
	if err != nil {
		return OrgRef{}, err
	}
	if org == nil {
		return OrgRef{}, ErrInvalidSession
	}
	return OrgRef{ID: org.ID, Name: org.Name, Slug: org.Slug}, nil
}

func (sdkOrgs) Delete(ctx context.Context, orgID string) error {
	_, err := clerkorg.Delete(ctx, orgID)
	return err
}

func (sdkOrgs) RemoveMember(ctx context.Context, orgID, clerkUserID string) error {
	_, err := clerkorgmem.Delete(ctx, &clerkorgmem.DeleteParams{
		OrganizationID: orgID,
		UserID:         clerkUserID,
	})
	return err
}

func (sdkOrgs) AddMember(ctx context.Context, orgID, clerkUserID, role string) error {
	_, err := clerkorgmem.Create(ctx, &clerkorgmem.CreateParams{
		OrganizationID: orgID,
		UserID:         &clerkUserID,
		Role:           &role,
	})
	return err
}

func (sdkOrgs) UpdateMember(ctx context.Context, orgID, clerkUserID, role string) error {
	_, err := clerkorgmem.Update(ctx, &clerkorgmem.UpdateParams{
		OrganizationID: orgID,
		UserID:         clerkUserID,
		Role:           &role,
	})
	return err
}

func (sdkOrgs) UpdateLogo(ctx context.Context, orgID, uploaderUserID string, file ImageFile) error {
	params := &clerkorg.UpdateLogoParams{File: file.File()}
	if uploaderUserID != "" {
		params.UploaderUserID = &uploaderUserID
	}
	_, err := clerkorg.UpdateLogo(ctx, orgID, params)
	return err
}

func (sdkOrgs) DeleteLogo(ctx context.Context, orgID string) error {
	_, err := clerkorg.DeleteLogo(ctx, orgID)
	return err
}
