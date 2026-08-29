package clerk

import "context"

// OrgRef is a Clerk organization after we drop the SDK type.
type OrgRef struct {
	ID   string
	Name string
	Slug string
}

// OrgMembership is one Clerk org the user belongs to.
type OrgMembership struct {
	Org  OrgRef
	Role string
}

// OrgDirectory is the Clerk organization control plane. Tests inject a fake.
type OrgDirectory interface {
	ListMemberships(ctx context.Context, clerkUserID string) ([]OrgMembership, error)
	Create(ctx context.Context, name, slug, createdBy string) (OrgRef, error)
	Delete(ctx context.Context, orgID string) error
	RemoveMember(ctx context.Context, orgID, clerkUserID string) error
	AddMember(ctx context.Context, orgID, clerkUserID, role string) error
	UpdateMember(ctx context.Context, orgID, clerkUserID, role string) error
	UpdateLogo(ctx context.Context, orgID, uploaderUserID string, file ImageFile) error
	DeleteLogo(ctx context.Context, orgID string) error
}
