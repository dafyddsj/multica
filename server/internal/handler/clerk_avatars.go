package handler

import (
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"path"
	"strings"

	"github.com/multica-ai/multica/server/internal/clerk"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// clerkAvatarMaxBytes is the largest image we will read to re-upload to Clerk.
// Matches Clerk's documented 10 MB profile-image cap.
const clerkAvatarMaxBytes = 10 << 20

// pushClerkUserAvatar copies a Multica user avatar onto the bound Clerk
// profile. Overlay-off, unbound users, emoji markers, and unreadable values
// are no-ops. Clerk failures are logged; the Multica write already committed.
func (h *Handler) pushClerkUserAvatar(ctx context.Context, user db.User, avatarURL string) {
	if h == nil || h.Clerk == nil || h.Clerk.Images == nil {
		return
	}
	if !user.ClerkUserID.Valid || user.ClerkUserID.String == "" {
		return
	}

	avatarURL = strings.TrimSpace(avatarURL)
	if avatarURL == "" {
		if err := h.Clerk.Images.DeleteProfileImage(ctx, user.ClerkUserID.String); err != nil {
			slog.Warn("clerk profile image delete failed", "error", err, "user_id", uuidToString(user.ID))
		}
		return
	}
	if strings.HasPrefix(avatarURL, "emoji:") {
		return
	}

	file, ok := h.clerkImageFromAvatar(ctx, avatarURL)
	if !ok {
		return
	}
	if err := h.Clerk.Images.UpdateProfileImage(ctx, user.ClerkUserID.String, file); err != nil {
		slog.Warn("clerk profile image update failed", "error", err, "user_id", uuidToString(user.ID))
	}
}

// pushClerkOrgLogo copies a Multica workspace avatar onto the bound Clerk org.
// Same skip / soft-fail rules as pushClerkUserAvatar.
func (h *Handler) pushClerkOrgLogo(ctx context.Context, workspace db.Workspace, uploaderUserID, avatarURL string) {
	if h == nil || h.Clerk == nil || h.Clerk.Orgs == nil {
		return
	}
	if !workspace.ClerkOrgID.Valid || workspace.ClerkOrgID.String == "" {
		return
	}

	clerkUserID := ""
	if uploaderUserID != "" {
		if user, err := h.Queries.GetUser(ctx, parseUUID(uploaderUserID)); err == nil && user.ClerkUserID.Valid {
			clerkUserID = user.ClerkUserID.String
		}
	}

	avatarURL = strings.TrimSpace(avatarURL)
	if avatarURL == "" {
		if err := h.Clerk.Orgs.DeleteLogo(ctx, workspace.ClerkOrgID.String); err != nil {
			slog.Warn("clerk org logo delete failed", "error", err, "workspace_id", uuidToString(workspace.ID))
		}
		return
	}
	if strings.HasPrefix(avatarURL, "emoji:") {
		return
	}

	file, ok := h.clerkImageFromAvatar(ctx, avatarURL)
	if !ok {
		return
	}
	if err := h.Clerk.Orgs.UpdateLogo(ctx, workspace.ClerkOrgID.String, clerkUserID, file); err != nil {
		slog.Warn("clerk org logo update failed", "error", err, "workspace_id", uuidToString(workspace.ID))
	}
}

// clerkImageFromAvatar loads image bytes for a stored avatar_url. Storage
// objects and data:image URIs work. Emoji markers, third-party https URLs,
// and missing objects return ok=false — we do not fetch arbitrary URLs.
func (h *Handler) clerkImageFromAvatar(ctx context.Context, raw string) (clerk.ImageFile, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "emoji:") {
		return clerk.ImageFile{}, false
	}
	if data, filename, ok := decodeDataURIImage(raw); ok {
		return clerk.ImageFile{Filename: filename, Bytes: data}, true
	}
	if h == nil || h.Storage == nil {
		return clerk.ImageFile{}, false
	}

	key := ""
	if k, served := avatarKeyFromServedURL(raw); served {
		key = k
	} else {
		key = h.ownedStorageKey(raw)
	}
	if key == "" || avatarContentType(key) == "" {
		return clerk.ImageFile{}, false
	}

	reader, err := h.Storage.GetReader(ctx, key)
	if err != nil {
		slog.Warn("clerk avatar storage read failed", "key", key, "error", err)
		return clerk.ImageFile{}, false
	}
	defer reader.Close()

	data, err := io.ReadAll(io.LimitReader(reader, clerkAvatarMaxBytes+1))
	if err != nil {
		slog.Warn("clerk avatar storage read failed", "key", key, "error", err)
		return clerk.ImageFile{}, false
	}
	if len(data) == 0 || len(data) > clerkAvatarMaxBytes {
		slog.Warn("clerk avatar skipped: empty or over size cap", "key", key, "bytes", len(data))
		return clerk.ImageFile{}, false
	}
	return clerk.ImageFile{Filename: path.Base(key), Bytes: data}, true
}

func decodeDataURIImage(raw string) ([]byte, string, bool) {
	if !strings.HasPrefix(raw, "data:image/") {
		return nil, "", false
	}
	header, payload, ok := strings.Cut(raw, ",")
	if !ok || payload == "" || !strings.Contains(header, ";base64") {
		return nil, "", false
	}
	media := strings.TrimPrefix(header, "data:")
	media, _, _ = strings.Cut(media, ";")
	ext := strings.TrimPrefix(media, "image/")
	if ext == "" || ext == media {
		return nil, "", false
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil || len(data) == 0 || len(data) > clerkAvatarMaxBytes {
		return nil, "", false
	}
	return data, "avatar." + ext, true
}
