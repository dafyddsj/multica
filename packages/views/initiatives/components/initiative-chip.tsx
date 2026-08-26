"use client";

import { useQuery } from "@tanstack/react-query";
import { initiativeListOptions, initiativeDetailOptions } from "@multica/core/initiatives/queries";
import { useWorkspaceId } from "@multica/core/hooks";
import { InitiativeIcon } from "./initiative-icon";
import { useT } from "../../i18n";

export interface InitiativeChipProps {
  initiativeId: string;
  fallbackLabel?: string;
  className?: string;
}

const BASE_CLASS =
  "initiative-chip inline-flex min-w-0 max-w-[min(18rem,100%)] items-center gap-1.5 rounded-md border mx-0.5 px-2 py-0.5 text-caption";

export function InitiativeChip({
  initiativeId,
  fallbackLabel,
  className,
}: InitiativeChipProps) {
  const { t } = useT("initiatives");
  const wsId = useWorkspaceId();
  const { data: initiatives = [] } = useQuery(initiativeListOptions(wsId));
  const listItem = initiatives.find((item) => item.id === initiativeId);

  const { data: detail } = useQuery({
    ...initiativeDetailOptions(wsId, initiativeId),
    enabled: !listItem,
  });

  const initiative = listItem ?? detail;
  const cls = className ? `${BASE_CLASS} ${className}` : BASE_CLASS;

  if (!initiative) {
    return (
      <span className={cls}>
        <InitiativeIcon size="md" />
        <span className="min-w-0 truncate text-muted-foreground">
          {fallbackLabel ?? t(($) => $.chip.fallback_label)}
        </span>
      </span>
    );
  }

  return (
    <span className={cls}>
      <InitiativeIcon initiative={initiative} size="md" />
      <span className="min-w-0 truncate text-foreground">{initiative.title}</span>
    </span>
  );
}
