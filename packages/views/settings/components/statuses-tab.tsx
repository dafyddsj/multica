"use client";

import { useState } from "react";
import { CircleDot, FolderKanban, Target } from "lucide-react";
import { Button } from "@multica/ui/components/ui/button";
import { cn } from "@multica/ui/lib/utils";
import { useT } from "../../i18n";
import { EntityStatusesPanel } from "./entity-statuses-panel";
import { IssueStatusesTab } from "./issue-statuses-tab";
import { SettingsTab } from "./settings-layout";

type StatusScope = "initiative" | "project" | "issue";

const SCOPES: { id: StatusScope; icon: typeof Target }[] = [
  { id: "initiative", icon: Target },
  { id: "project", icon: FolderKanban },
  { id: "issue", icon: CircleDot },
];

export function StatusesTab() {
  const { t } = useT("settings");
  const [scope, setScope] = useState<StatusScope>("initiative");

  return (
    <SettingsTab
      title={t(($) => $.statuses.title)}
      description={t(($) => $.statuses.description)}
    >
      <div className="space-y-5">
        <div className="flex flex-wrap items-center gap-2 border-b border-surface-border pb-3">
          {SCOPES.map(({ id, icon: Icon }) => (
            <Button
              key={id}
              type="button"
              size="sm"
              variant={scope === id ? "secondary" : "ghost"}
              className={cn(
                "gap-2",
                scope === id && "bg-surface-selected text-surface-selected-foreground",
              )}
              onClick={() => setScope(id)}
            >
              <Icon className="size-3.5" />
              {t(($) => $.statuses.scopes[id])}
            </Button>
          ))}
        </div>

        {scope === "issue" ? (
          <IssueStatusesTab embedded />
        ) : (
          <EntityStatusesPanel resourceType={scope} />
        )}
      </div>
    </SettingsTab>
  );
}
