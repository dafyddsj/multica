import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";
import { INITIATIVE_STATUS_CONFIG, INITIATIVE_STATUS_ORDER } from "../initiatives/config";
import { PROJECT_STATUS_CONFIG, PROJECT_STATUS_ORDER } from "../projects/config";
import type {
  EntityStatusCategory,
  EntityStatusEntry,
  EntityStatusResourceType,
} from "../types";

export const ENTITY_STATUS_CATEGORIES: EntityStatusCategory[] = [
  "planned",
  "in_progress",
  "paused",
  "completed",
  "cancelled",
];

export const entityStatusKeys = {
  all: (wsId: string) => ["entity-statuses", wsId] as const,
  list: (wsId: string, resourceType: EntityStatusResourceType) =>
    [...entityStatusKeys.all(wsId), resourceType] as const,
};

export function entityStatusListOptions(
  wsId: string,
  resourceType: EntityStatusResourceType,
) {
  return queryOptions({
    queryKey: entityStatusKeys.list(wsId, resourceType),
    queryFn: () => api.listEntityStatuses(resourceType, true),
    select: (data) => data.statuses,
    staleTime: 5 * 60_000,
  });
}

export interface EntityStatusCatalog {
  statuses: EntityStatusEntry[];
  activeStatuses: EntityStatusEntry[];
  categoryOf: (statusKey: string) => EntityStatusCategory;
  labelOf: (statusKey: string) => string;
  entryOf: (statusKey: string) => EntityStatusEntry | undefined;
  colorOf: (statusKey: string) => string | null;
  inCategory: (category: EntityStatusCategory) => EntityStatusEntry[];
  isLoaded: boolean;
  isPending: boolean;
  isError: boolean;
  retry: () => void;
  hasCustomStatuses: boolean;
}

const CATEGORY_RANK = new Map<string, number>(
  ENTITY_STATUS_CATEGORIES.map((c, i) => [c, i]),
);

export function compareEntityStatusEntries(
  a: EntityStatusEntry,
  b: EntityStatusEntry,
): number {
  const rank =
    (CATEGORY_RANK.get(a.category) ?? ENTITY_STATUS_CATEGORIES.length) -
    (CATEGORY_RANK.get(b.category) ?? ENTITY_STATUS_CATEGORIES.length);
  if (rank !== 0) return rank;
  if (a.position !== b.position) return a.position - b.position;
  return a.key.localeCompare(b.key);
}

export function isEntityStatusCategory(value: string): value is EntityStatusCategory {
  return CATEGORY_RANK.has(value);
}

export function isClosedEntityStatus(statusKey: string, catalog?: Pick<EntityStatusCatalog, "categoryOf">): boolean {
  const category = catalog?.categoryOf(statusKey) ?? (isEntityStatusCategory(statusKey) ? statusKey : statusKey);
  return category === "completed" || category === "cancelled";
}

export function entityStatusColor(entry: EntityStatusEntry | undefined): string | null {
  if (!entry || entry.is_system === true) return null;
  return entry.color;
}

function fallbackLabel(resourceType: EntityStatusResourceType, statusKey: string): string {
  if (resourceType === "initiative") {
    return INITIATIVE_STATUS_CONFIG[statusKey as keyof typeof INITIATIVE_STATUS_CONFIG]?.label
      ?? (INITIATIVE_STATUS_ORDER.includes(statusKey as (typeof INITIATIVE_STATUS_ORDER)[number])
        ? statusKey
        : statusKey);
  }
  return PROJECT_STATUS_CONFIG[statusKey as keyof typeof PROJECT_STATUS_CONFIG]?.label ?? statusKey;
}

export function buildEntityStatusCatalog(
  entries: EntityStatusEntry[] | undefined,
  resourceType: EntityStatusResourceType,
  status: { isPending?: boolean; isError?: boolean; retry?: () => void } = {},
): EntityStatusCatalog {
  const list = entries ?? [];
  const byKey = new Map(list.map((e) => [e.key, e]));

  const categoryOf = (statusKey: string): EntityStatusCategory => {
    const category = byKey.get(statusKey)?.category;
    if (category && isEntityStatusCategory(category)) return category;
    if (isEntityStatusCategory(statusKey)) return statusKey;
    return "planned";
  };

  return {
    statuses: list,
    activeStatuses: list.filter((e) => !e.archived_at),
    categoryOf,
    entryOf: (statusKey) => byKey.get(statusKey),
    colorOf: (statusKey) => entityStatusColor(byKey.get(statusKey)),
    labelOf: (statusKey) => {
      const entry = byKey.get(statusKey);
      if (entry) return entry.name;
      return fallbackLabel(resourceType, statusKey);
    },
    inCategory: (category) => list.filter((e) => e.category === category && !e.archived_at),
    isLoaded: entries !== undefined,
    isPending: status.isPending ?? entries === undefined,
    isError: (status.isError ?? false) && entries === undefined,
    retry: status.retry ?? (() => {}),
    hasCustomStatuses: entries !== undefined && list.some((e) => e.is_system !== true),
  };
}
