"use client";

import { AppLink } from "../../navigation";
import { useWorkspacePaths } from "@multica/core/paths";
import { InitiativeChip } from "./initiative-chip";

export function InitiativeMentionCard({
  initiativeId,
  fallbackLabel,
}: {
  initiativeId: string;
  fallbackLabel?: string;
}) {
  const p = useWorkspacePaths();
  return (
    <AppLink
      href={p.initiativeDetail(initiativeId)}
      newTabTitle={fallbackLabel}
      className="initiative-mention inline-flex"
    >
      <InitiativeChip
        initiativeId={initiativeId}
        fallbackLabel={fallbackLabel}
        className="cursor-pointer hover:bg-accent transition-colors"
      />
    </AppLink>
  );
}
