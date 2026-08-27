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
