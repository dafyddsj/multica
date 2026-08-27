package clerk

import "testing"

func TestMapClerkRole(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"org:admin", "owner"},
		{"org:owner", "owner"},
		{"ORG:ADMIN", "owner"},
		{"admin", "owner"},
		{"owner", "owner"},
		{"org:member", "member"},
		{"member", "member"},
		{"org:billing", "member"},
		{"", "member"},
		{"  org:admin  ", "owner"},
	}
	for _, tc := range cases {
		if got := MapClerkRole(tc.in); got != tc.want {
			t.Errorf("MapClerkRole(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestClerkRoleFromMember(t *testing.T) {
	if got := ClerkRoleFromMember("owner"); got != "org:admin" {
		t.Fatalf("owner: %q", got)
	}
	if got := ClerkRoleFromMember("admin"); got != "org:admin" {
		t.Fatalf("admin: %q", got)
	}
	if got := ClerkRoleFromMember("member"); got != "org:member" {
		t.Fatalf("member: %q", got)
	}
}
