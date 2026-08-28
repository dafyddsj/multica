import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import type { CreateBudgetRequest, CreateBudgetWaiverRequest, UpdateBudgetRequest } from "../types";
import { budgetKeys } from "./queries";

function invalidateBudgets(qc: ReturnType<typeof useQueryClient>, wsId: string) {
  void qc.invalidateQueries({ queryKey: budgetKeys.all(wsId) });
}

export function useCreateBudget(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateBudgetRequest) => api.createBudget(data),
    onSettled: () => invalidateBudgets(qc, wsId),
  });
}

export function useUpdateBudget(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...data }: { id: string } & UpdateBudgetRequest) =>
      api.updateBudget(id, data),
    onSettled: () => invalidateBudgets(qc, wsId),
  });
}

export function useDeleteBudget(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteBudget(id),
    onSettled: () => invalidateBudgets(qc, wsId),
  });
}

export function useCreateBudgetWaiver(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateBudgetWaiverRequest) => api.createBudgetWaiver(data),
    onSettled: () => invalidateBudgets(qc, wsId),
  });
}

export function useDeleteBudgetWaiver(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteBudgetWaiver(id),
    onSettled: () => invalidateBudgets(qc, wsId),
  });
}
