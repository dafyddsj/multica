import { useQuery } from "@tanstack/react-query";
import { budgetListOptions, budgetWaiverListOptions } from "./queries";

export function useBudgets(wsId: string) {
  return useQuery(budgetListOptions(wsId));
}

export function useBudgetWaivers(wsId: string) {
  return useQuery(budgetWaiverListOptions(wsId));
}
