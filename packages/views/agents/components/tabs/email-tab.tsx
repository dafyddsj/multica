"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import type { Agent } from "@multica/core/types";
import { api } from "@multica/core/api";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import {
  agentmailInboxOptions,
  agentmailKeys,
  agentmailWorkspaceOptions,
} from "@multica/core/agentmail";
import { Switch } from "@multica/ui/components/ui/switch";
import { Label } from "@multica/ui/components/ui/label";
import { AppLink } from "../../../navigation";
import { useT } from "../../../i18n";

export function EmailTab({ agent }: { agent: Agent }) {
  const { t } = useT("agents");
  const wsId = useWorkspaceId();
  const paths = useWorkspacePaths();
  const settingsHref = `${paths.settings()}?tab=agentmail`;
  const qc = useQueryClient();
  const workspaceQuery = useQuery(agentmailWorkspaceOptions(wsId));
  const inboxQuery = useQuery(agentmailInboxOptions(wsId, agent.id));
  const connected = workspaceQuery.data?.connected === true;
  const enabled = inboxQuery.data?.enabled === true;
  const address = inboxQuery.data?.address ?? "";
  const [pending, setPending] = useState(false);

  async function toggle(next: boolean) {
    if (pending) return;
    setPending(true);
    try {
      if (next) {
        await api.grantAgentMailInbox(agent.id);
        toast.success(t(($) => $.email.toast_granted));
      } else {
        await api.revokeAgentMailInbox(agent.id);
        toast.success(t(($) => $.email.toast_revoked));
      }
      await qc.invalidateQueries({ queryKey: agentmailKeys.all(wsId) });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t(($) => $.email.toast_failed));
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-6">
      <p className="text-body text-muted-foreground">{t(($) => $.email.description)}</p>
      {!connected ? (
        <p className="text-body text-muted-foreground">
          {t(($) => $.email.not_connected_prefix)}{" "}
          <AppLink href={settingsHref} className="underline underline-offset-2">
            {t(($) => $.email.settings_link)}
          </AppLink>
          {t(($) => $.email.not_connected_suffix)}
        </p>
      ) : (
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            <Label htmlFor="agentmail-inbox" className="text-body font-medium">
              {t(($) => $.email.switch_label)}
            </Label>
            {enabled && address ? (
              <p className="text-caption text-muted-foreground">
                <code>{address}</code>
              </p>
            ) : (
              <p className="text-caption text-muted-foreground">
                {t(($) => $.email.switch_off)}
              </p>
            )}
          </div>
          <Switch
            id="agentmail-inbox"
            checked={enabled}
            onCheckedChange={(value) => void toggle(value)}
            disabled={pending}
          />
        </div>
      )}
    </div>
  );
}
