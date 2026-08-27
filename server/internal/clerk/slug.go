package clerk

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	notSlugChar = regexp.MustCompile(`[^a-z0-9]+`)
)

// WorkspaceSlug picks a Multica workspace slug from a Clerk org.
// Clerk's own slug wins when it already matches Multica's pattern.
// Otherwise the name is slugified. Reserved slugs and empty results
// get a suffix derived from the Clerk org id.
func WorkspaceSlug(name, clerkSlug, orgID string, reserved func(string) bool) string {
	candidate := strings.ToLower(strings.TrimSpace(clerkSlug))
	if !slugPattern.MatchString(candidate) {
		candidate = slugify(name)
	}
	if candidate == "" {
		candidate = "org"
	}
	if reserved != nil && reserved(candidate) {
		return withOrgSuffix(candidate, orgID)
	}
	return candidate
}

func slugify(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		if r == ' ' || r == '-' || r == '_' {
			b.WriteByte('-')
		}
	}
	collapsed := notSlugChar.ReplaceAllString(b.String(), "-")
	return strings.Trim(collapsed, "-")
}

func withOrgSuffix(base, orgID string) string {
	suffix := orgSuffix(orgID)
	if suffix == "" {
		return base + "-org"
	}
	out := base + "-" + suffix
	if slugPattern.MatchString(out) {
		return out
	}
	return "org-" + suffix
}

func orgSuffix(orgID string) string {
	id := strings.ToLower(strings.TrimSpace(orgID))
	id = strings.TrimPrefix(id, "org_")
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 8 {
		out = out[len(out)-8:]
	}
	return out
}

func nextSlugAttempt(base string, attempt int, orgID string) string {
	if attempt == 0 {
		return base
	}
	suffix := orgSuffix(orgID)
	if suffix == "" {
		suffix = "org"
	}
	if attempt == 1 {
		return base + "-" + suffix
	}
	return base + "-" + suffix + "-" + strconv.Itoa(attempt)
}
