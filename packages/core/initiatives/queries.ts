import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const initiativeKeys = {
  all: (wsId: string) => ["initiatives", wsId] as const,
  list: (wsId: string) => [...initiativeKeys.all(wsId), "list"] as const,
  detail: (wsId: string, id: string) =>
    [...initiativeKeys.all(wsId), "detail", id] as const,
};

export function initiativeListOptions(wsId: string) {
  return queryOptions({
    queryKey: initiativeKeys.list(wsId),
    queryFn: () => api.listInitiatives(),
    select: (data) => data.initiatives,
  });
}

export function initiativeDetailOptions(wsId: string, id: string) {
  return queryOptions({
    queryKey: initiativeKeys.detail(wsId, id),
    queryFn: () => api.getInitiative(id),
  });
}
