const USERNAME_RE = /^[a-z0-9](?:[a-z0-9._-]{0,62}[a-z0-9])?$/;

export function suggestedAgentMailUsername(name: string): string {
  const slug = name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 64);
  if (USERNAME_RE.test(slug)) return slug;
  const compact = slug.replace(/[^a-z0-9]/g, "").slice(0, 64);
  return USERNAME_RE.test(compact) ? compact : "";
}

export function isAgentMailUsername(value: string): boolean {
  return USERNAME_RE.test(value.trim().toLowerCase());
}
