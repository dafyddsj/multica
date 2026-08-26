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
