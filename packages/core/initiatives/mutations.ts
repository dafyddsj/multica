import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { initiativeKeys } from "./queries";
import { projectKeys } from "../projects/queries";
import { useWorkspaceId } from "../hooks";
import type {
  Initiative,
  CreateInitiativeRequest,
  UpdateInitiativeRequest,
  ListInitiativesResponse,
} from "../types";

export function useCreateInitiative() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (data: CreateInitiativeRequest) => api.createInitiative(data),
    onSuccess: (created) => {
      qc.setQueryData<ListInitiativesResponse>(initiativeKeys.list(wsId), (old) =>
        old && !old.initiatives.some((item) => item.id === created.id)
          ? { ...old, initiatives: [...old.initiatives, created], total: old.total + 1 }
          : old,
      );
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: initiativeKeys.list(wsId) });
    },
  });
}

export function useUpdateInitiative() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: ({ id, ...data }: { id: string } & UpdateInitiativeRequest) =>
      api.updateInitiative(id, data),
    onMutate: ({ id, ...data }) => {
      qc.cancelQueries({ queryKey: initiativeKeys.list(wsId) });
      const prevList = qc.getQueryData<ListInitiativesResponse>(initiativeKeys.list(wsId));
      const prevDetail = qc.getQueryData<Initiative>(initiativeKeys.detail(wsId, id));
      qc.setQueryData<ListInitiativesResponse>(initiativeKeys.list(wsId), (old) =>
        old
          ? {
              ...old,
              initiatives: old.initiatives.map((item) =>
                item.id === id ? { ...item, ...data } : item,
              ),
            }
          : old,
      );
      qc.setQueryData<Initiative>(initiativeKeys.detail(wsId, id), (old) =>
        old ? { ...old, ...data } : old,
      );
      return { prevList, prevDetail, id };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prevList) qc.setQueryData(initiativeKeys.list(wsId), ctx.prevList);
      if (ctx?.prevDetail) qc.setQueryData(initiativeKeys.detail(wsId, ctx.id), ctx.prevDetail);
    },
    onSettled: (_data, _err, vars) => {
      qc.invalidateQueries({ queryKey: initiativeKeys.detail(wsId, vars.id) });
      qc.invalidateQueries({ queryKey: initiativeKeys.list(wsId) });
    },
  });
}

export function useDeleteInitiative() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (id: string) => api.deleteInitiative(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: initiativeKeys.all(wsId) });
      qc.invalidateQueries({ queryKey: projectKeys.all(wsId) });
    },
  });
}
