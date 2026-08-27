package clerk

import "testing"

func TestWorkspaceSlug(t *testing.T) {
	reserved := func(s string) bool { return s == "login" || s == "inbox" }

	cases := []struct {
		name, clerkSlug, orgID, want string
	}{
		{"Test Org", "test-org", "org_abc123xyz", "test-org"},
		{"Test Org", "Test Org", "org_abc123xyz", "test-org"},
		{"Hello World", "", "org_abc123xyz", "hello-world"},
		{"登录", "", "org_2NabcXYZ9", "org"},
		{"Login", "login", "org_2NabcXYZ9", "login-nabcxyz9"},
	}
	for _, tc := range cases {
		got := WorkspaceSlug(tc.name, tc.clerkSlug, tc.orgID, reserved)
		if got != tc.want {
			t.Errorf("WorkspaceSlug(%q,%q,%q)=%q want %q", tc.name, tc.clerkSlug, tc.orgID, got, tc.want)
		}
	}
}

func TestNextSlugAttempt(t *testing.T) {
	if got := nextSlugAttempt("acme", 0, "org_deadbeef"); got != "acme" {
		t.Fatalf("attempt 0: %q", got)
	}
	if got := nextSlugAttempt("acme", 1, "org_deadbeef"); got != "acme-deadbeef" {
		t.Fatalf("attempt 1: %q", got)
	}
	if got := nextSlugAttempt("acme", 2, "org_deadbeef"); got != "acme-deadbeef-2" {
		t.Fatalf("attempt 2: %q", got)
	}
}
