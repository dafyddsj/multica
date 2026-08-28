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
  agentmailThreadOptions,
  agentmailThreadsOptions,
  agentmailWorkspaceOptions,
} from "@multica/core/agentmail";
import { Switch } from "@multica/ui/components/ui/switch";
import { Label } from "@multica/ui/components/ui/label";
import { Button } from "@multica/ui/components/ui/button";
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
      {enabled ? <MailViewer agentId={agent.id} /> : null}
    </div>
  );
}

function MailViewer({ agentId }: { agentId: string }) {
  const { t } = useT("agents");
  const wsId = useWorkspaceId();
  const [threadId, setThreadId] = useState("");
  const listQuery = useQuery(agentmailThreadsOptions(wsId, agentId, true));
  const detailQuery = useQuery(agentmailThreadOptions(wsId, agentId, threadId));
  const threads = listQuery.data?.threads ?? [];

  if (threadId) {
    const subject = detailQuery.data?.subject || t(($) => $.email.untitled);
    const messages = detailQuery.data?.messages ?? [];
    return (
      <div className="space-y-4">
        <Button type="button" variant="ghost" size="sm" onClick={() => setThreadId("")}>
          {t(($) => $.email.back)}
        </Button>
        <h3 className="text-title font-medium">{subject}</h3>
        {detailQuery.isError ? (
          <p className="text-body text-muted-foreground">{t(($) => $.email.viewer_load_failed)}</p>
        ) : (
          <div className="space-y-4">
            {messages.map((message) => (
              <article key={message.message_id} className="space-y-2">
                <p className="text-caption text-muted-foreground">{message.from}</p>
                <pre className="text-body whitespace-pre-wrap break-words font-sans">
                  {message.text}
                </pre>
              </article>
            ))}
          </div>
        )}
      </div>
    );
  }

  if (listQuery.isError) {
    return <p className="text-body text-muted-foreground">{t(($) => $.email.viewer_load_failed)}</p>;
  }
  if (threads.length === 0) {
    return <p className="text-body text-muted-foreground">{t(($) => $.email.viewer_empty)}</p>;
  }

  return (
    <div className="space-y-1">
      {threads.map((thread) => (
        <button
          key={thread.thread_id}
          type="button"
          className="flex w-full flex-col items-start gap-1 rounded-md px-2 py-2 text-left hover:bg-muted"
          onClick={() => setThreadId(thread.thread_id)}
        >
          <span className="text-body font-medium">
            {thread.subject || t(($) => $.email.untitled)}
          </span>
          {thread.preview ? (
            <span className="text-caption text-muted-foreground line-clamp-2">{thread.preview}</span>
          ) : null}
        </button>
      ))}
    </div>
  );
}
