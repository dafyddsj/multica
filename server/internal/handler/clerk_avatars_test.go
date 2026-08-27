package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"testing"

	"github.com/multica-ai/multica/server/internal/clerk"
	"github.com/multica-ai/multica/server/internal/testutil"
)

// 1×1 PNG. Used as a real image Clerk would accept.
const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(tinyPNGBase64)
	if err != nil {
		t.Fatalf("tiny png: %v", err)
	}
	return data
}

func dataURITinyPNG() string {
	return "data:image/png;base64," + tinyPNGBase64
}

func TestDecodeDataURIImage(t *testing.T) {
	t.Parallel()

	got, filename, ok := decodeDataURIImage(dataURITinyPNG())
	if !ok || filename != "avatar.png" || !bytes.Equal(got, mustDecodeB64(tinyPNGBase64)) {
		t.Fatalf("decode: ok=%v filename=%q bytes=%d", ok, filename, len(got))
	}

	for _, raw := range []string{
		"",
		"emoji:🚀",
		"https://cdn.example.com/users/u/avatar.png",
		"data:text/plain;base64,aGVsbG8=",
		"data:image/png,not-base64",
		"data:image/png;base64,",
	} {
		if _, _, ok := decodeDataURIImage(raw); ok {
			t.Fatalf("decodeDataURIImage(%q) should fail", raw)
		}
	}
}

func mustDecodeB64(s string) []byte {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return data
}

func TestClerkImageFromAvatar(t *testing.T) {
	png := tinyPNG(t)
	store := &mockStorage{files: map[string][]byte{
		"users/u/avatar.png": png,
	}}
	h := &Handler{Storage: store}

	t.Run("data uri", func(t *testing.T) {
		file, ok := h.clerkImageFromAvatar(context.Background(), dataURITinyPNG())
		if !ok || !bytes.Equal(file.Bytes, png) {
			t.Fatalf("data uri: ok=%v bytes=%d", ok, len(file.Bytes))
		}
	})

	t.Run("owned storage url", func(t *testing.T) {
		file, ok := h.clerkImageFromAvatar(context.Background(), "https://cdn.example.com/users/u/avatar.png")
		if !ok || !bytes.Equal(file.Bytes, png) || file.Filename != "avatar.png" {
			t.Fatalf("storage: ok=%v filename=%q bytes=%d", ok, file.Filename, len(file.Bytes))
		}
	})

	t.Run("served avatar url", func(t *testing.T) {
		file, ok := h.clerkImageFromAvatar(context.Background(), avatarURLPath("users/u/avatar.png"))
		if !ok || !bytes.Equal(file.Bytes, png) {
			t.Fatalf("served: ok=%v bytes=%d", ok, len(file.Bytes))
		}
	})

	t.Run("skips emoji and third-party urls", func(t *testing.T) {
		for _, raw := range []string{"", "emoji:🚀", "https://lh3.googleusercontent.com/a/p.png"} {
			if _, ok := h.clerkImageFromAvatar(context.Background(), raw); ok {
				t.Fatalf("expected skip for %q", raw)
			}
		}
	})
}

type recordingImages struct {
	updates []clerk.ImageFile
	deletes int
	err     error
}

func (r *recordingImages) UpdateProfileImage(_ context.Context, _ string, file clerk.ImageFile) error {
	if r.err != nil {
		return r.err
	}
	r.updates = append(r.updates, file)
	return nil
}

func (r *recordingImages) DeleteProfileImage(context.Context, string) error {
	if r.err != nil {
		return r.err
	}
	r.deletes++
	return nil
}

type recordingLogoOrgs struct {
	stubHandlerOrgs
	logos   []clerk.ImageFile
	deletes int
	err     error
}

func (r *recordingLogoOrgs) UpdateLogo(_ context.Context, _, _ string, file clerk.ImageFile) error {
	if r.err != nil {
		return r.err
	}
	r.logos = append(r.logos, file)
	return nil
}

func (r *recordingLogoOrgs) DeleteLogo(context.Context, string) error {
	if r.err != nil {
		return r.err
	}
	r.deletes++
	return nil
}

func bindTestUserClerkID(t *testing.T, clerkUserID string) {
	t.Helper()
	ctx := context.Background()
	var previous *string
	var previousAvatar *string
	_ = testPool.QueryRow(ctx, `SELECT clerk_user_id, avatar_url FROM "user" WHERE id = $1`, testUserID).Scan(&previous, &previousAvatar)
	_, _ = testPool.Exec(ctx, `UPDATE "user" SET clerk_user_id = $1 WHERE id = $2`, clerkUserID, testUserID)
	t.Cleanup(func() {
		if previous == nil {
			_, _ = testPool.Exec(context.Background(), `UPDATE "user" SET clerk_user_id = NULL, avatar_url = $1 WHERE id = $2`, previousAvatar, testUserID)
		} else {
			_, _ = testPool.Exec(context.Background(), `UPDATE "user" SET clerk_user_id = $1, avatar_url = $2 WHERE id = $3`, *previous, previousAvatar, testUserID)
		}
	})
}

func TestUpdateMe_PushesClerkProfileImage(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	bindTestUserClerkID(t, "user_avatar_push")
	images := &recordingImages{}
	orig := testHandler.Clerk
	testHandler.Clerk = &clerk.Client{Images: images, Orgs: stubHandlerOrgs{}}
	t.Cleanup(func() { testHandler.Clerk = orig })

	var me UserResponse
	testutil.Call(t, testHandler.UpdateMe, newRequest("PATCH", "/api/me", map[string]any{
		"avatar_url": dataURITinyPNG(),
	})).Want(http.StatusOK).JSON(&me)
	if me.AvatarURL == nil || *me.AvatarURL != dataURITinyPNG() {
		t.Fatalf("Multica avatar: got %v", me.AvatarURL)
	}
	if len(images.updates) != 1 || !bytes.Equal(images.updates[0].Bytes, tinyPNG(t)) {
		t.Fatalf("clerk image updates: %+v", images.updates)
	}
}

func TestUpdateMe_SkipsEmojiAvatarOnClerk(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	bindTestUserClerkID(t, "user_avatar_emoji")
	images := &recordingImages{}
	orig := testHandler.Clerk
	testHandler.Clerk = &clerk.Client{Images: images, Orgs: stubHandlerOrgs{}}
	t.Cleanup(func() { testHandler.Clerk = orig })

	testutil.Call(t, testHandler.UpdateMe, newRequest("PATCH", "/api/me", map[string]any{
		"avatar_url": "emoji:🚀",
	})).Want(http.StatusOK)
	if len(images.updates) != 0 || images.deletes != 0 {
		t.Fatalf("emoji must not call Clerk: updates=%d deletes=%d", len(images.updates), images.deletes)
	}
}

func TestUpdateMe_ClearsClerkProfileImage(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	bindTestUserClerkID(t, "user_avatar_clear")
	images := &recordingImages{}
	orig := testHandler.Clerk
	testHandler.Clerk = &clerk.Client{Images: images, Orgs: stubHandlerOrgs{}}
	t.Cleanup(func() { testHandler.Clerk = orig })

	testutil.Call(t, testHandler.UpdateMe, newRequest("PATCH", "/api/me", map[string]any{
		"avatar_url": "",
	})).Want(http.StatusOK)
	if images.deletes != 1 {
		t.Fatalf("clears: deletes=%d", images.deletes)
	}
}

func TestUpdateMe_ClerkImageFailureDoesNotFailRequest(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	bindTestUserClerkID(t, "user_avatar_fail")
	orig := testHandler.Clerk
	testHandler.Clerk = &clerk.Client{
		Images: &recordingImages{err: errors.New("clerk down")},
		Orgs:   stubHandlerOrgs{},
	}
	t.Cleanup(func() { testHandler.Clerk = orig })

	testutil.Call(t, testHandler.UpdateMe, newRequest("PATCH", "/api/me", map[string]any{
		"avatar_url": dataURITinyPNG(),
	})).Want(http.StatusOK)
}

func TestUpdateWorkspace_PushesClerkOrgLogo(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	const slug = "handler-clerk-logo"
	const orgID = "org_logo_push"
	_, _ = testPool.Exec(ctx, `DELETE FROM workspace WHERE slug = $1 OR clerk_org_id = $2`, slug, orgID)
	wsID := dbfx.Insert(t, "workspace", testutil.Cols{
		"name":         "Clerk Logo",
		"slug":         slug,
		"description":  "logo push",
		"clerk_org_id": orgID,
	})
	dbfx.Exec(t, `INSERT INTO member (workspace_id, user_id, role) VALUES ($1, $2, 'owner')`, wsID, testUserID)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM workspace WHERE slug = $1 OR clerk_org_id = $2`, slug, orgID)
	})

	bindTestUserClerkID(t, "user_logo_push")
	orgs := &recordingLogoOrgs{}
	orig := testHandler.Clerk
	testHandler.Clerk = &clerk.Client{Orgs: orgs}
	t.Cleanup(func() { testHandler.Clerk = orig })

	req := newRequest("PATCH", "/api/workspaces/"+wsID, map[string]any{
		"avatar_url": dataURITinyPNG(),
	})
	req = withURLParam(req, "id", wsID)
	testutil.Call(t, testHandler.UpdateWorkspace, req).Want(http.StatusOK)
	if len(orgs.logos) != 1 || !bytes.Equal(orgs.logos[0].Bytes, tinyPNG(t)) {
		t.Fatalf("clerk logos: %+v", orgs.logos)
	}
}

func TestUpdateWorkspace_SkipsEmojiLogoOnClerk(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	const slug = "handler-clerk-logo-emoji"
	const orgID = "org_logo_emoji"
	_, _ = testPool.Exec(ctx, `DELETE FROM workspace WHERE slug = $1 OR clerk_org_id = $2`, slug, orgID)
	wsID := dbfx.Insert(t, "workspace", testutil.Cols{
		"name":         "Clerk Emoji Logo",
		"slug":         slug,
		"clerk_org_id": orgID,
	})
	dbfx.Exec(t, `INSERT INTO member (workspace_id, user_id, role) VALUES ($1, $2, 'owner')`, wsID, testUserID)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM workspace WHERE slug = $1 OR clerk_org_id = $2`, slug, orgID)
	})

	orgs := &recordingLogoOrgs{}
	orig := testHandler.Clerk
	testHandler.Clerk = &clerk.Client{Orgs: orgs}
	t.Cleanup(func() { testHandler.Clerk = orig })

	req := newRequest("PATCH", "/api/workspaces/"+wsID, map[string]any{
		"avatar_url": "emoji:🏢",
	})
	req = withURLParam(req, "id", wsID)
	testutil.Call(t, testHandler.UpdateWorkspace, req).Want(http.StatusOK)
	if len(orgs.logos) != 0 || orgs.deletes != 0 {
		t.Fatalf("emoji must not call Clerk: logos=%d deletes=%d", len(orgs.logos), orgs.deletes)
	}
}

func TestUpdateWorkspace_ClerkLogoFailureDoesNotFailRequest(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()
	const slug = "handler-clerk-logo-fail"
	const orgID = "org_logo_fail"
	_, _ = testPool.Exec(ctx, `DELETE FROM workspace WHERE slug = $1 OR clerk_org_id = $2`, slug, orgID)
	wsID := dbfx.Insert(t, "workspace", testutil.Cols{
		"name":         "Clerk Logo Fail",
		"slug":         slug,
		"clerk_org_id": orgID,
	})
	dbfx.Exec(t, `INSERT INTO member (workspace_id, user_id, role) VALUES ($1, $2, 'owner')`, wsID, testUserID)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM workspace WHERE slug = $1 OR clerk_org_id = $2`, slug, orgID)
	})

	orig := testHandler.Clerk
	testHandler.Clerk = &clerk.Client{Orgs: &recordingLogoOrgs{err: errors.New("clerk down")}}
	t.Cleanup(func() { testHandler.Clerk = orig })

	req := newRequest("PATCH", "/api/workspaces/"+wsID, map[string]any{
		"avatar_url": dataURITinyPNG(),
	})
	req = withURLParam(req, "id", wsID)
	testutil.Call(t, testHandler.UpdateWorkspace, req).Want(http.StatusOK)
}
