import type { Project } from "@multica/core/types";
import type { ProjectListFilters, ProjectSortField } from "@multica/core/projects";

export const NO_INITIATIVE_FILTER = "none";

export const PRIORITY_ORDER: Record<Project["priority"], number> = {
  urgent: 4,
  high: 3,
  medium: 2,
  low: 1,
  none: 0,
};

export const STATUS_ORDER: Record<Project["status"], number> = {
  planned: 0,
  in_progress: 1,
  paused: 2,
  completed: 3,
  cancelled: 4,
};

export function progressOf(project: Project): number {
  return project.issue_count > 0 ? project.done_count / project.issue_count : -1;
}

export function leadFilterValue(project: Project): string | null {
  return project.lead_type && project.lead_id
    ? `${project.lead_type}:${project.lead_id}`
    : null;
}

export function countActiveFilters(filters: ProjectListFilters): number {
  let count = 0;
  if (filters.statuses.length) count += 1;
  if (filters.priorities.length) count += 1;
  if (filters.leads.length) count += 1;
  if (filters.initiatives.length) count += 1;
  return count;
}

export function matchesProjectSearch(
  project: Project,
  query: string,
  matchesPinyin: (text: string, q: string) => boolean,
): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return true;
  return project.title.toLowerCase().includes(q) || matchesPinyin(project.title, q);
}

export function projectPassesFilters(project: Project, filters: ProjectListFilters): boolean {
  if (filters.statuses.length && !filters.statuses.includes(project.status)) return false;
  if (filters.priorities.length && !filters.priorities.includes(project.priority)) {
    return false;
  }
  if (filters.leads.length) {
    const lead = leadFilterValue(project);
    if (!lead || !filters.leads.includes(lead)) return false;
  }
  if (filters.initiatives.length) {
    const value = project.initiative_id ?? NO_INITIATIVE_FILTER;
    if (!filters.initiatives.includes(value)) return false;
  }
  return true;
}

export function compareProjects(
  a: Project,
  b: Project,
  sortField: ProjectSortField,
  direction: "asc" | "desc",
  initiativeTitle: (id: string | null) => string,
): number {
  const dir = direction === "asc" ? 1 : -1;
  if (sortField === "name") return a.title.localeCompare(b.title) * dir;
  if (sortField === "priority") {
    return (
      (PRIORITY_ORDER[a.priority] - PRIORITY_ORDER[b.priority]) * dir ||
      a.title.localeCompare(b.title)
    );
  }
  if (sortField === "status") {
    return (
      (STATUS_ORDER[a.status] - STATUS_ORDER[b.status]) * dir ||
      a.title.localeCompare(b.title)
    );
  }
  if (sortField === "progress") {
    return (progressOf(a) - progressOf(b)) * dir || a.title.localeCompare(b.title);
  }
  if (sortField === "initiative") {
    return (
      initiativeTitle(a.initiative_id).localeCompare(initiativeTitle(b.initiative_id)) * dir ||
      a.title.localeCompare(b.title)
    );
  }
  return (Date.parse(a.created_at) - Date.parse(b.created_at)) * dir;
}

export interface ProjectListGroup {
  key: string;
  initiativeId: string | null;
  projects: Project[];
}

export function groupProjectsByInitiative(projects: Project[]): ProjectListGroup[] {
  const groups = new Map<string, ProjectListGroup>();
  const order: string[] = [];
  for (const project of projects) {
    const key = project.initiative_id ?? NO_INITIATIVE_FILTER;
    const existing = groups.get(key);
    if (existing) {
      existing.projects.push(project);
      continue;
    }
    order.push(key);
    groups.set(key, {
      key,
      initiativeId: project.initiative_id,
      projects: [project],
    });
  }
  return order.map((key) => groups.get(key)!);
}

export function sortProjectGroups(
  groups: ProjectListGroup[],
  initiativeTitle: (id: string | null) => string,
): ProjectListGroup[] {
  return [...groups].sort((a, b) => {
    if (a.initiativeId === null) return 1;
    if (b.initiativeId === null) return -1;
    return initiativeTitle(a.initiativeId).localeCompare(initiativeTitle(b.initiativeId));
  });
}
