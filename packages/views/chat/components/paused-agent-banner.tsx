"use client";

import { Pause } from "lucide-react";
import { cn } from "@multica/ui/lib/utils";
import { CHAT_COLUMN, CHAT_GUTTER } from "./chat-column";
import { useT } from "../../i18n";

// Sibling of ArchivedAgentBanner. Pause leaves the transcript readable and
// refuses new sends until someone resumes the agent.
export function PausedAgentBanner({ agentName }: { agentName?: string }) {
  const { t } = useT("chat");
  const name = agentName?.trim() || t(($) => $.offline_banner.fallback_name);
  return (
    <div className={cn(CHAT_GUTTER, "mb-1.5")}>
      <div className={cn(CHAT_COLUMN, "flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-caption bg-muted text-muted-foreground ring-1 ring-border")}>
        <Pause className="size-3.5 shrink-0" />
        <span className="truncate">
          {t(($) => $.paused_agent_banner, { name })}
        </span>
      </div>
    </div>
  );
}
