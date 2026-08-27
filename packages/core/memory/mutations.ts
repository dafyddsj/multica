import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";
import type { CreateMemoryRequest, MemoryListResponse } from "../types/memory";
import { memoryKeys } from "./queries";

export function useCreateMemory() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (data: CreateMemoryRequest) => api.createMemory(data),
    onSuccess: (entry) => {
      qc.setQueryData<MemoryListResponse>(
        memoryKeys.list(wsId, entry.scope, entry.owner_id),
        (old) =>
          old
            ? { entries: [entry, ...old.entries], total: old.total + 1 }
            : { entries: [entry], total: 1 },
      );
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: memoryKeys.all(wsId) });
    },
  });
}

export function useForgetMemory() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (id: string) => api.forgetMemory(id),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: memoryKeys.all(wsId) });
    },
  });
}
