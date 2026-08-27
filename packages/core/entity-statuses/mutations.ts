import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";
import type {
  CreateEntityStatusRequest,
  EntityStatusCategory,
  EntityStatusEntry,
  EntityStatusResourceType,
  ListEntityStatusesResponse,
  UpdateEntityStatusRequest,
} from "../types";
import { compareEntityStatusEntries, entityStatusKeys } from "./queries";

function useCatalogCache(resourceType: EntityStatusResourceType) {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  const listKey = entityStatusKeys.list(wsId, resourceType);

  const writeStatuses = (
    update: (statuses: EntityStatusEntry[]) => EntityStatusEntry[] | null,
    totalDelta = 0,
  ) => {
    qc.setQueryData<ListEntityStatusesResponse>(listKey, (old) => {
      if (!old) return old;
      const statuses = update(old.statuses);
      if (!statuses) return old;
      return {
        ...old,
        statuses: statuses.sort(compareEntityStatusEntries),
        total: old.total + totalDelta,
      };
    });
  };

  return {
    qc,
    listKey,
    insertEntry: (entry: EntityStatusEntry) => {
      if (!entry.id) return;
      writeStatuses((statuses) => {
        if (statuses.some((s) => s.id === entry.id)) return null;
        return [...statuses, entry];
      }, 1);
    },
    patchEntry: (id: string, patch: Partial<EntityStatusEntry>) => {
      writeStatuses((statuses) =>
        statuses.map((s) => (s.id === id ? { ...s, ...patch } : s)),
      );
    },
    archiveEntry: (id: string) => {
      writeStatuses((statuses) =>
        statuses.map((s) =>
          s.id === id ? { ...s, archived_at: s.archived_at ?? new Date().toISOString() } : s,
        ),
      );
    },
    invalidate: () => {
      void qc.invalidateQueries({ queryKey: listKey });
    },
  };
}

export function useCreateEntityStatus(resourceType: EntityStatusResourceType) {
  const cache = useCatalogCache(resourceType);
  return useMutation({
    mutationFn: (data: Omit<CreateEntityStatusRequest, "resource_type">) =>
      api.createEntityStatus({ ...data, resource_type: resourceType }),
    onSuccess: (entry) => cache.insertEntry(entry),
    onError: () => cache.invalidate(),
  });
}

export function useUpdateEntityStatus(resourceType: EntityStatusResourceType) {
  const cache = useCatalogCache(resourceType);
  return useMutation({
    mutationFn: ({ id, ...data }: { id: string } & UpdateEntityStatusRequest) =>
      api.updateEntityStatus(id, data),
    onMutate: ({ id, ...data }) => {
      cache.patchEntry(id, data);
    },
    onError: () => cache.invalidate(),
  });
}

export function useArchiveEntityStatus(resourceType: EntityStatusResourceType) {
  const cache = useCatalogCache(resourceType);
  return useMutation({
    mutationFn: (id: string) => api.archiveEntityStatus(id),
    onMutate: (id) => cache.archiveEntry(id),
    onError: () => cache.invalidate(),
  });
}

export function useReorderEntityStatuses(resourceType: EntityStatusResourceType) {
  const cache = useCatalogCache(resourceType);
  return useMutation({
    mutationFn: ({
      category,
      ordered,
    }: {
      category: EntityStatusCategory;
      ordered: EntityStatusEntry[];
    }) => api.reorderEntityStatuses(resourceType, category, ordered.map((s) => s.id)),
    onMutate: ({ ordered }) => {
      const byId = new Map(ordered.map((s, i) => [s.id, i + 1]));
      cache.qc.setQueryData<ListEntityStatusesResponse>(cache.listKey, (old) => {
        if (!old) return old;
        return {
          ...old,
          statuses: old.statuses
            .map((s) => (byId.has(s.id) ? { ...s, position: byId.get(s.id)! } : s))
            .sort(compareEntityStatusEntries),
        };
      });
    },
    onError: () => cache.invalidate(),
  });
}
