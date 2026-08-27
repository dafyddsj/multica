package handler

import (
	"context"
	"log/slog"

	"github.com/multica-ai/multica/server/internal/clerk"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func (h *Handler) syncClerkOrgs(ctx context.Context, user db.User) error {
	if h == nil || h.Clerk == nil {
		return nil
	}
	if !user.ClerkUserID.Valid || user.ClerkUserID.String == "" {
		return nil
	}
	return h.Clerk.SyncOrgs(ctx, user.ClerkUserID.String, user.ID, h.Queries, isReservedSlug)
}

func (h *Handler) createClerkOrg(ctx context.Context, name, slug, userID string) (string, error) {
	if h == nil || h.Clerk == nil || h.Clerk.Orgs == nil {
		return "", nil
	}
	user, err := h.Queries.GetUser(ctx, parseUUID(userID))
	if err != nil {
		return "", err
	}
	if !user.ClerkUserID.Valid || user.ClerkUserID.String == "" {
		return "", nil
	}
	org, err := h.Clerk.Orgs.Create(ctx, name, slug, user.ClerkUserID.String)
	if err != nil {
		return "", err
	}
	return org.ID, nil
}

func (h *Handler) removeClerkOrgMember(ctx context.Context, workspace db.Workspace, userID string) error {
	if h == nil || h.Clerk == nil || h.Clerk.Orgs == nil {
		return nil
	}
	if !workspace.ClerkOrgID.Valid || workspace.ClerkOrgID.String == "" {
		return nil
	}
	user, err := h.Queries.GetUser(ctx, parseUUID(userID))
	if err != nil {
		return err
	}
	if !user.ClerkUserID.Valid || user.ClerkUserID.String == "" {
		return nil
	}
	err = h.Clerk.Orgs.RemoveMember(ctx, workspace.ClerkOrgID.String, user.ClerkUserID.String)
	if clerk.IsNotFound(err) {
		return nil
	}
	return err
}

func (h *Handler) deleteClerkOrg(ctx context.Context, workspace db.Workspace) error {
	if h == nil || h.Clerk == nil || h.Clerk.Orgs == nil {
		return nil
	}
	if !workspace.ClerkOrgID.Valid || workspace.ClerkOrgID.String == "" {
		return nil
	}
	err := h.Clerk.Orgs.Delete(ctx, workspace.ClerkOrgID.String)
	if clerk.IsNotFound(err) {
		return nil
	}
	return err
}

func (h *Handler) addClerkOrgMember(ctx context.Context, workspace db.Workspace, user db.User, role string) error {
	if h == nil || h.Clerk == nil || h.Clerk.Orgs == nil {
		return nil
	}
	if !workspace.ClerkOrgID.Valid || workspace.ClerkOrgID.String == "" {
		return nil
	}
	if !user.ClerkUserID.Valid || user.ClerkUserID.String == "" {
		return nil
	}
	err := h.Clerk.Orgs.AddMember(ctx, workspace.ClerkOrgID.String, user.ClerkUserID.String, clerk.ClerkRoleFromMember(role))
	if err != nil {
		slog.Debug("clerk add member", "org", workspace.ClerkOrgID.String, "error", err)
	}
	return err
}
