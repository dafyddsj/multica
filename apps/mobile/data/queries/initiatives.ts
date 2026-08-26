/**
 * Workspace initiative queries. Two query shapes:
 *
 *   - List    (initiativeKeys.list)    — `Initiative[]`
 *   - Detail  (initiativeKeys.detail)  — `Initiative`
 *
 * Key shape matches `packages/core/initiatives/queries.ts`. Child projects
 * are not an initiative query: they are the project list filtered by
 * `Project.initiative_id` (same membership as `GET /api/projects?initiative_id=`).
 */
import { queryOptions } from "@tanstack/react-query";
import type { Initiative, Project } from "@multica/core/types";
import { api } from "@/data/api";

export const initiativeKeys = {
  all: (wsId: string | null) => ["initiatives", wsId] as const,
  list: (wsId: string | null) => [...initiativeKeys.all(wsId), "list"] as const,
  detail: (wsId: string | null, id: string) =>
    [...initiativeKeys.all(wsId), "detail", id] as const,
};

export const initiativeListOptions = (wsId: string | null) =>
  queryOptions({
    queryKey: initiativeKeys.list(wsId),
    queryFn: async ({ signal }) => {
      const res = await api.listInitiatives({ signal });
      return res.initiatives;
    },
    enabled: !!wsId,
  });

export const initiativeDetailOptions = (wsId: string | null, id: string) =>
  queryOptions({
    queryKey: initiativeKeys.detail(wsId, id),
    queryFn: ({ signal }) => api.getInitiative(id, { signal }),
    enabled: !!wsId && !!id,
  });

export function findInitiative(
  initiatives: Initiative[],
  id: string | null,
): Initiative | undefined {
  if (!id) return undefined;
  return initiatives.find((item) => item.id === id);
}

/** Child projects of an initiative. Same filter the server applies for
 *  `listProjects({ initiative_id })`. */
export function projectsForInitiative(
  projects: Project[],
  initiativeId: string,
): Project[] {
  return projects.filter((p) => p.initiative_id === initiativeId);
}
