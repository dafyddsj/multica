/** Sentinel for "no project filter" — kept distinct from the empty string so
 *  it survives a refactor that ever lets a project be slug-keyed. */
export const ALL_PROJECTS = "__all__";
export const ALL_INITIATIVES = "__all__";

export function resolveDashboardScopeId(
  value: string,
  allSentinel: string,
  items: { id: string }[],
): string | null {
  if (value === allSentinel) return null;
  return items.some((item) => item.id === value) ? value : null;
}

export function resolveDashboardInitiativeId(
  value: string,
  initiatives: { id: string }[],
): string | null {
  return resolveDashboardScopeId(value, ALL_INITIATIVES, initiatives);
}

export function projectsForInitiative<T extends { initiative_id: string | null }>(
  projects: T[],
  initiativeId: string | null,
): T[] {
  if (initiativeId === null) return projects;
  return projects.filter((project) => project.initiative_id === initiativeId);
}

export function resolveDashboardProjectId(
  value: string,
  projects: { id: string }[],
): string | null {
  return resolveDashboardScopeId(value, ALL_PROJECTS, projects);
}
