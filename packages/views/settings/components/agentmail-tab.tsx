"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Mail } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@multica/ui/components/ui/button";
import { Card, CardContent } from "@multica/ui/components/ui/card";
import { Input } from "@multica/ui/components/ui/input";
import { Label } from "@multica/ui/components/ui/label";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@multica/ui/components/ui/alert-dialog";
import { useWorkspaceId } from "@multica/core/hooks";
import { useConfigStore } from "@multica/core/config";
import { api } from "@multica/core/api";
import {
  agentmailKeys,
  agentmailWorkspaceOptions,
} from "@multica/core/agentmail";
import { useT } from "../../i18n";
import { SettingsTab } from "./settings-layout";

export function AgentMailTab() {
  const { t } = useT("settings");
  const wsId = useWorkspaceId();
  const qc = useQueryClient();
  const hostedAvailable = useConfigStore((s) => s.agentmailHostedAvailable);

  const { data } = useQuery(agentmailWorkspaceOptions(wsId));
  const available = data?.available === true;
  const canManage = data?.can_manage === true;
  const connected = data?.connected === true;
  const inboxes = (data?.inboxes ?? []).filter((inbox) => inbox.enabled === true);

  const [orgKey, setOrgKey] = useState("");
  const [connecting, setConnecting] = useState(false);
  const [disconnectOpen, setDisconnectOpen] = useState(false);
  const [disconnecting, setDisconnecting] = useState(false);

  async function refresh() {
    await qc.invalidateQueries({ queryKey: agentmailKeys.all(wsId) });
  }

  async function connect(mode: "hosted" | "bring_your_own") {
    if (connecting) return;
    setConnecting(true);
    try {
      await api.connectAgentMail(
        wsId,
        mode === "bring_your_own" ? { mode, org_key: orgKey } : { mode },
      );
      setOrgKey("");
      await refresh();
      toast.success(t(($) => $.agentmail.toast_connected));
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t(($) => $.agentmail.toast_connect_failed));
    } finally {
      setConnecting(false);
    }
  }

  async function disconnect() {
    if (disconnecting) return;
    setDisconnecting(true);
    try {
      await api.disconnectAgentMail(wsId);
      await refresh();
      setDisconnectOpen(false);
      toast.success(t(($) => $.agentmail.toast_disconnected));
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t(($) => $.agentmail.toast_disconnect_failed));
    } finally {
      setDisconnecting(false);
    }
  }

  return (
    <SettingsTab
      title={t(($) => $.page.tabs.agentmail)}
      description={t(($) => $.agentmail.page_description)}
    >
      <section className="space-y-3">
        <h2 className="text-body font-semibold">{t(($) => $.agentmail.section_connection)}</h2>
        <Card>
          <CardContent className="space-y-4">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-start gap-3">
                <div className="rounded-md border bg-muted/50 p-2 text-muted-foreground">
                  <Mail className="h-4 w-4" />
                </div>
                <div className="space-y-1">
                  <p className="text-body font-medium">{t(($) => $.agentmail.connection_title)}</p>
                  <p className="text-caption text-muted-foreground">
                    {connected
                      ? data?.source === "hosted"
                        ? t(($) => $.agentmail.hosted_connected)
                        : t(($) => $.agentmail.byo_connected)
                      : canManage
                        ? t(($) => $.agentmail.not_connected)
                        : t(($) => $.agentmail.contact_admin)}
                  </p>
                </div>
              </div>
              {canManage && connected ? (
                <Button variant="outline" size="sm" onClick={() => setDisconnectOpen(true)}>
                  {t(($) => $.agentmail.disconnect)}
                </Button>
              ) : null}
            </div>

            {canManage && !connected && available ? (
              <div className="space-y-3">
                {hostedAvailable ? (
                  <Button size="sm" onClick={() => void connect("hosted")} disabled={connecting}>
                    {connecting
                      ? t(($) => $.agentmail.connecting)
                      : t(($) => $.agentmail.connect_hosted)}
                  </Button>
                ) : null}
                <div className="space-y-2">
                  <Label htmlFor="agentmail-org-key" className="text-body">
                    {t(($) => $.agentmail.org_key_label)}
                  </Label>
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <Input
                      id="agentmail-org-key"
                      type="password"
                      autoComplete="off"
                      value={orgKey}
                      onChange={(event) => setOrgKey(event.target.value)}
                      placeholder={t(($) => $.agentmail.org_key_placeholder)}
                    />
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => void connect("bring_your_own")}
                      disabled={connecting || orgKey.trim() === ""}
                    >
                      {t(($) => $.agentmail.connect_byo)}
                    </Button>
                  </div>
                </div>
              </div>
            ) : null}

            {!canManage && connected ? (
              <p className="text-caption text-muted-foreground">
                {t(($) => $.agentmail.read_only_hint)}
              </p>
            ) : null}
          </CardContent>
        </Card>
      </section>

      <section className="space-y-3">
        <h2 className="text-body font-semibold">{t(($) => $.agentmail.section_inboxes)}</h2>
        <Card>
          <CardContent className="space-y-3">
            {inboxes.length === 0 ? (
              <p className="text-caption text-muted-foreground">
                {t(($) => $.agentmail.inboxes_empty)}
              </p>
            ) : (
              <ul className="space-y-2">
                {inboxes.map((inbox) => (
                  <li key={inbox.agent_id} className="flex items-baseline justify-between gap-3">
                    <span className="min-w-0 truncate text-body">
                      {inbox.display_name || inbox.agent_id}
                    </span>
                    <code className="shrink-0 text-caption text-muted-foreground">
                      {inbox.address}
                    </code>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </section>

      <AlertDialog open={disconnectOpen} onOpenChange={setDisconnectOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t(($) => $.agentmail.disconnect_title)}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(($) => $.agentmail.disconnect_description)}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t(($) => $.agentmail.disconnect_cancel)}</AlertDialogCancel>
            <AlertDialogAction onClick={() => void disconnect()} disabled={disconnecting}>
              {disconnecting
                ? t(($) => $.agentmail.disconnecting)
                : t(($) => $.agentmail.disconnect_action)}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SettingsTab>
  );
}
