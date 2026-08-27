import { useWorkspaceId } from "@multica/core/hooks";
import { useEntityStatuses } from "@multica/core/entity-statuses";
import { INITIATIVE_STATUS_CONFIG, INITIATIVE_STATUS_ORDER } from "@multica/core/initiatives/config";
import { PROJECT_STATUS_CONFIG, PROJECT_STATUS_ORDER } from "@multica/core/projects/config";
import type { EntityStatusCategory, EntityStatusResourceType } from "@multica/core/types";
import { useInitiativeStatusLabels } from "../initiatives/components/labels";
import { useProjectStatusLabels } from "../projects/components/labels";

export interface EntityStatusOption {
  key: string;
  label: string;
  category: EntityStatusCategory;
  hex: string | null;
  dotClass: string;
  badgeBg: string;
  badgeText: string;
}

const FALLBACK = {
  project: { order: PROJECT_STATUS_ORDER, config: PROJECT_STATUS_CONFIG },
  initiative: { order: INITIATIVE_STATUS_ORDER, config: INITIATIVE_STATUS_CONFIG },
} as const;

function appearance(resourceType: EntityStatusResourceType, category: EntityStatusCategory) {
  const cfg = FALLBACK[resourceType].config[category];
  return {
    dotClass: cfg?.dotColor ?? "bg-muted-foreground",
    badgeBg: cfg?.badgeBg ?? "bg-muted",
    badgeText: cfg?.badgeText ?? "text-muted-foreground",
  };
}

export function useEntityStatusPicker(resourceType: EntityStatusResourceType) {
  const wsId = useWorkspaceId();
  const catalog = useEntityStatuses(wsId, resourceType);
  const fallback = FALLBACK[resourceType];
  const projectLabels = useProjectStatusLabels();
  const initiativeLabels = useInitiativeStatusLabels();
  const localeLabels = resourceType === "initiative" ? initiativeLabels : projectLabels;

  const labelFor = (entry: { key: string; name: string; is_system: boolean }): string => {
    if (entry.is_system === true) {
      const seed = fallback.config[entry.key as keyof typeof fallback.config]?.label;
      const localized = localeLabels[entry.key as keyof typeof localeLabels];
      if (seed && localized && entry.name === seed) return localized;
    }
    return entry.name;
  };

  const options: EntityStatusOption[] =
    catalog.activeStatuses.length > 0
      ? catalog.activeStatuses.map((entry) => {
          const category = catalog.categoryOf(entry.key);
          return {
            key: entry.key,
            label: labelFor(entry),
            category,
            hex: catalog.colorOf(entry.key),
            ...appearance(resourceType, category),
          };
        })
      : fallback.order.map((key) => ({
          key,
          label: localeLabels[key],
          category: key,
          hex: null,
          ...appearance(resourceType, key),
        }));

  const current = (statusKey: string): EntityStatusOption => {
    const found = options.find((option) => option.key === statusKey);
    if (found) return found;
    const category = catalog.categoryOf(statusKey);
    return {
      key: statusKey,
      label: catalog.labelOf(statusKey),
      category,
      hex: catalog.colorOf(statusKey),
      ...appearance(resourceType, category),
    };
  };

  return { options, current, catalog };
}
