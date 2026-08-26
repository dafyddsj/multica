/**
 * Mobile-owned WS cache helpers for the initiative domain. Pure functions
 * over `QueryClient` — no React, no WS plumbing.
 *
 * Web invalidates `initiativeKeys.all` (and `projectKeys.all`) on any
 * `initiative:*` prefix in `packages/core/realtime/use-realtime-sync.ts`.
 * Mobile follows the same invalidate policy for list/detail, and refreshes
 * projects only on `initiative:deleted` (children detach).
 *
 * Keys come from `data/queries/initiatives.ts`, not packages/core.
 */
import type { QueryClient } from "@tanstack/react-query";
import type { Initiative } from "@multica/core/types";
import { initiativeKeys } from "@/data/queries/initiatives";
import { projectKeys } from "@/data/queries/projects";

export function invalidateInitiativeCaches(qc: QueryClient, wsId: string) {
  qc.invalidateQueries({ queryKey: initiativeKeys.all(wsId) });
}

export function invalidateProjectsAfterInitiativeDeleted(
  qc: QueryClient,
  wsId: string,
) {
  qc.invalidateQueries({ queryKey: projectKeys.all(wsId) });
}

export function dropInitiativeCaches(
  qc: QueryClient,
  wsId: string,
  initiativeId: string,
) {
  qc.removeQueries({ queryKey: initiativeKeys.detail(wsId, initiativeId) });
  qc.setQueryData<Initiative[]>(initiativeKeys.list(wsId), (old) =>
    old ? old.filter((item) => item.id !== initiativeId) : old,
  );
}
