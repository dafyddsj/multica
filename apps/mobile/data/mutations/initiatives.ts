import { useMutation, useQueryClient } from "@tanstack/react-query";
import type {
  CreateInitiativeRequest,
  Initiative,
  UpdateInitiativeRequest,
} from "@multica/core/types";
import { api } from "@/data/api";
import { initiativeKeys } from "@/data/queries/initiatives";
import { projectKeys } from "@/data/queries/projects";
import { useWorkspaceStore } from "@/data/workspace-store";

export function useCreateInitiative() {
  const qc = useQueryClient();
  const wsId = useWorkspaceStore((s) => s.currentWorkspaceId);

  return useMutation({
    mutationFn: (body: CreateInitiativeRequest) => api.createInitiative(body),
    onSuccess: (initiative) => {
      qc.setQueryData<Initiative>(
        initiativeKeys.detail(wsId, initiative.id),
        initiative,
      );
      qc.setQueryData<Initiative[]>(initiativeKeys.list(wsId), (old) =>
        old
          ? [initiative, ...old.filter((item) => item.id !== initiative.id)]
          : [initiative],
      );
    },
  });
}

export function useUpdateInitiative(initiativeId: string) {
  const qc = useQueryClient();
  const wsId = useWorkspaceStore((s) => s.currentWorkspaceId);

  return useMutation({
    mutationKey: ["updateInitiative", initiativeId] as const,
    mutationFn: (patch: UpdateInitiativeRequest) =>
      api.updateInitiative(initiativeId, patch),
    onMutate: async (patch) => {
      const detailKey = initiativeKeys.detail(wsId, initiativeId);
      const listKey = initiativeKeys.list(wsId);
      await Promise.all([
        qc.cancelQueries({ queryKey: detailKey }),
        qc.cancelQueries({ queryKey: listKey }),
      ]);

      const prevDetail = qc.getQueryData<Initiative>(detailKey);
      const prevList = qc.getQueryData<Initiative[]>(listKey);

      if (prevDetail) {
        qc.setQueryData<Initiative>(detailKey, { ...prevDetail, ...patch });
      }
      qc.setQueryData<Initiative[]>(listKey, (old) =>
        old
          ? old.map((item) =>
              item.id === initiativeId ? { ...item, ...patch } : item,
            )
          : old,
      );

      return { prevDetail, prevList, detailKey, listKey };
    },
    onError: (_err, _vars, ctx) => {
      if (!ctx) return;
      if (ctx.prevDetail !== undefined) {
        qc.setQueryData(ctx.detailKey, ctx.prevDetail);
      }
      if (ctx.prevList !== undefined) {
        qc.setQueryData(ctx.listKey, ctx.prevList);
      }
    },
    onSuccess: (server) => {
      qc.setQueryData<Initiative>(
        initiativeKeys.detail(wsId, initiativeId),
        server,
      );
      qc.setQueryData<Initiative[]>(initiativeKeys.list(wsId), (old) =>
        old
          ? old.map((item) => (item.id === initiativeId ? server : item))
          : old,
      );
    },
  });
}

export function useDeleteInitiative(initiativeId: string) {
  const qc = useQueryClient();
  const wsId = useWorkspaceStore((s) => s.currentWorkspaceId);

  return useMutation({
    mutationKey: ["deleteInitiative", initiativeId] as const,
    mutationFn: () => api.deleteInitiative(initiativeId),
    onSuccess: () => {
      qc.removeQueries({
        queryKey: initiativeKeys.detail(wsId, initiativeId),
      });
      qc.setQueryData<Initiative[]>(initiativeKeys.list(wsId), (old) =>
        old ? old.filter((item) => item.id !== initiativeId) : old,
      );
      qc.invalidateQueries({ queryKey: initiativeKeys.all(wsId) });
      qc.invalidateQueries({ queryKey: projectKeys.all(wsId) });
    },
  });
}
