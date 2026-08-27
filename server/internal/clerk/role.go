package clerk

import "strings"

// clerkRoleToMember is the only role map. Clerk org roles become Multica
// member roles here, not in handlers.
var clerkRoleToMember = map[string]string{
	"org:admin":  "owner",
	"org:owner":  "owner",
	"admin":      "owner",
	"owner":      "owner",
	"org:member": "member",
	"member":     "member",
}

// MapClerkRole converts a Clerk organization role into a Multica member role.
// Unknown keys become member so a custom Clerk role cannot mint owner.
func MapClerkRole(role string) string {
	key := strings.ToLower(strings.TrimSpace(role))
	if mapped, ok := clerkRoleToMember[key]; ok {
		return mapped
	}
	return "member"
}

// ConvergeMemberRole applies a Clerk org role onto an existing Multica
// membership. Clerk only has org:admin / org:member, so a local admin
// must stay admin when Clerk reports org:admin. Otherwise the next
// GetMe would silently promote them to owner. A Clerk demotion to
// org:member still wins.
func ConvergeMemberRole(local, clerkRole string) string {
	mapped := MapClerkRole(clerkRole)
	if strings.EqualFold(strings.TrimSpace(local), "admin") && mapped == "owner" {
		return "admin"
	}
	return mapped
}

// ClerkRoleFromMember is the inverse used when Multica writes back to Clerk.
// Multica admin is closer to Clerk org:admin than org:member.
func ClerkRoleFromMember(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "owner", "admin":
		return "org:admin"
	default:
		return "org:member"
	}
}
