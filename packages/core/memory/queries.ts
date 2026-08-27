import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";
import type { MemoryScope } from "../types/memory";

export const memoryKeys = {
  all: (wsId: string) => ["memory", wsId] as const,
  list: (wsId: string, scope: MemoryScope, ownerId: string, q?: string) =>
    [...memoryKeys.all(wsId), "list", scope, ownerId, q ?? ""] as const,
};

export function memoryListOptions(
  wsId: string,
  scope: MemoryScope,
  ownerId: string,
  q?: string,
) {
  return queryOptions({
    queryKey: memoryKeys.list(wsId, scope, ownerId, q),
    queryFn: () => api.listMemory({ scope, owner_id: ownerId, q, limit: 100 }),
    enabled: Boolean(wsId && ownerId),
  });
}
