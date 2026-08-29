import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const budgetKeys = {
  all: (wsId: string) => ["budgets", wsId] as const,
  waivers: (wsId: string) => ["budgets", wsId, "waivers"] as const,
};

export function budgetListOptions(wsId: string) {
  return queryOptions({
    queryKey: budgetKeys.all(wsId),
    queryFn: () => api.listBudgets(),
    enabled: Boolean(wsId),
  });
}

export function budgetWaiverListOptions(wsId: string) {
  return queryOptions({
    queryKey: budgetKeys.waivers(wsId),
    queryFn: () => api.listBudgetWaivers(),
    enabled: Boolean(wsId),
  });
}
