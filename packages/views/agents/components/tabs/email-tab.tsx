"use client";

import { useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import type { Agent } from "@multica/core/types";
import { api } from "@multica/core/api";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import {
  agentmailAccountInboxOptions,
  agentmailDomainOptions,
  agentmailFoldersOptions,
  agentmailInboxOptions,
  agentmailKeys,
  agentmailMailboxOptions,
  agentmailThreadOptions,
  agentmailWorkspaceOptions,
  isAgentMailUsername,
  suggestedAgentMailUsername,
} from "@multica/core/agentmail";
import { Switch } from "@multica/ui/components/ui/switch";
import { Label } from "@multica/ui/components/ui/label";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@multica/ui/components/ui/select";
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
import { cn } from "@multica/ui/lib/utils";
import { AppLink } from "../../../navigation";
import { useT } from "../../../i18n";

type MailSection = "inbox" | "sent" | "drafts" | "scheduled" | "all" | "trash" | "folder";
type GrantMode = "create" | "link";
type RevokeStep = "choose" | "confirm" | null;

const FIXED_SECTIONS: { id: Exclude<MailSection, "folder">; labelKey: Exclude<MailSection, "folder"> }[] = [
  { id: "inbox", labelKey: "inbox" },
  { id: "sent", labelKey: "sent" },
  { id: "drafts", labelKey: "drafts" },
  { id: "scheduled", labelKey: "scheduled" },
  { id: "all", labelKey: "all" },
  { id: "trash", labelKey: "trash" },
];

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
  const [grantMode, setGrantMode] = useState<GrantMode>("create");
  const [username, setUsername] = useState(() => suggestedAgentMailUsername(agent.name));
  const [domain, setDomain] = useState("agentmail.to");
  const [inboxID, setInboxID] = useState("");
  const [revokeStep, setRevokeStep] = useState<RevokeStep>(null);
  const domainsQuery = useQuery(agentmailDomainOptions(wsId, connected && !enabled && grantMode === "create"));
  const accountQuery = useQuery(
    agentmailAccountInboxOptions(wsId, connected && !enabled && grantMode === "link"),
  );
  const domains = useMemo(() => {
    const listed = domainsQuery.data?.domains ?? [];
    return listed.length > 0 ? listed : ["agentmail.to"];
  }, [domainsQuery.data?.domains]);
  const availableInboxes = useMemo(
    () => (accountQuery.data?.inboxes ?? []).filter((row) => row.linked !== true),
    [accountQuery.data?.inboxes],
  );
  const selectedInbox = inboxID || availableInboxes[0]?.inbox_id || "";

  async function grant() {
    if (pending) return;
    if (grantMode === "link") {
      if (!selectedInbox) {
        toast.error(t(($) => $.email.link_empty));
        return;
      }
      setPending(true);
      try {
        await api.grantAgentMailInbox(agent.id, { mode: "link", inbox_id: selectedInbox });
        toast.success(t(($) => $.email.toast_granted));
        await qc.invalidateQueries({ queryKey: agentmailKeys.all(wsId) });
      } catch (e) {
        toast.error(e instanceof Error ? e.message : t(($) => $.email.toast_failed));
      } finally {
        setPending(false);
      }
      return;
    }
    const local = username.trim().toLowerCase();
    if (!isAgentMailUsername(local)) {
      toast.error(t(($) => $.email.username_invalid));
      return;
    }
    setPending(true);
    try {
      await api.grantAgentMailInbox(agent.id, { mode: "create", username: local, domain });
      toast.success(t(($) => $.email.toast_granted));
      await qc.invalidateQueries({ queryKey: agentmailKeys.all(wsId) });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t(($) => $.email.toast_failed));
    } finally {
      setPending(false);
    }
  }

  async function revoke(deleteRemote: boolean) {
    if (pending) return;
    setRevokeStep(null);
    setPending(true);
    try {
      await api.revokeAgentMailInbox(agent.id, { deleteRemote });
      toast.success(deleteRemote ? t(($) => $.email.toast_deleted) : t(($) => $.email.toast_kept));
      await qc.invalidateQueries({ queryKey: agentmailKeys.all(wsId) });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t(($) => $.email.toast_failed));
    } finally {
      setPending(false);
    }
  }

  if (!connected) {
    return (
      <div className="mx-auto max-w-3xl space-y-4 p-4 sm:p-6">
        <p className="text-body text-muted-foreground">{t(($) => $.email.description)}</p>
        <p className="text-body text-muted-foreground">
          {t(($) => $.email.not_connected_prefix)}{" "}
          <AppLink href={settingsHref} className="underline underline-offset-2">
            {t(($) => $.email.settings_link)}
          </AppLink>
          {t(($) => $.email.not_connected_suffix)}
        </p>
      </div>
    );
  }

  if (!enabled) {
    const preview = username.trim()
      ? `${username.trim().toLowerCase()}@${domain}`
      : `…@${domain}`;
    const domainItems = domains.map((value) => ({ value, label: value }));
    const inboxItems = availableInboxes.map((row) => ({
      value: row.inbox_id,
      label: row.email,
    }));
    return (
      <div className="mx-auto max-w-3xl space-y-6 p-4 sm:p-6">
        <p className="text-body text-muted-foreground">{t(($) => $.email.description)}</p>
        <div
          className="flex gap-1 rounded-lg bg-muted p-1"
          role="tablist"
          aria-label={t(($) => $.email.grant_mode_aria)}
        >
          {(["create", "link"] as const).map((mode) => {
            const active = grantMode === mode;
            return (
              <button
                key={mode}
                type="button"
                role="tab"
                aria-selected={active}
                onClick={() => setGrantMode(mode)}
                className={cn(
                  "flex-1 rounded-md px-3 py-1.5 text-caption transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                  active
                    ? "bg-background font-medium text-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground",
                )}
              >
                {mode === "create" ? t(($) => $.email.mode_create) : t(($) => $.email.mode_link)}
              </button>
            );
          })}
        </div>
        {grantMode === "link" ? (
          <div className="space-y-4">
            {inboxItems.length === 0 ? (
              <p className="text-body text-muted-foreground">{t(($) => $.email.link_empty)}</p>
            ) : (
              <div className="space-y-2">
                <Label className="text-body font-medium">{t(($) => $.email.link_label)}</Label>
                <Select
                  items={inboxItems}
                  value={selectedInbox}
                  onValueChange={(next) => {
                    if (typeof next === "string" && next) setInboxID(next);
                  }}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder={t(($) => $.email.link_placeholder)} />
                  </SelectTrigger>
                  <SelectContent align="start">
                    {inboxItems.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
            <Button
              type="button"
              onClick={() => void grant()}
              disabled={pending || inboxItems.length === 0}
            >
              {t(($) => $.email.link_inbox)}
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="agentmail-username" className="text-body font-medium">
                {t(($) => $.email.username_label)}
              </Label>
              <Input
                id="agentmail-username"
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                autoComplete="off"
                spellCheck={false}
                placeholder={t(($) => $.email.username_placeholder)}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-body font-medium">{t(($) => $.email.domain_label)}</Label>
              <Select
                items={domainItems}
                value={domain}
                onValueChange={(next) => {
                  if (typeof next === "string" && next) setDomain(next);
                }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="start">
                  {domainItems.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <p className="text-caption text-muted-foreground">
              {t(($) => $.email.address_preview)}{" "}
              <code>{preview}</code>
            </p>
            <Button type="button" onClick={() => void grant()} disabled={pending}>
              {t(($) => $.email.create_inbox)}
            </Button>
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="flex min-h-[620px] flex-1 flex-col">
      <div className="flex items-start justify-between gap-4 border-b px-4 py-3 sm:px-6">
        <div className="space-y-1">
          <Label htmlFor="agentmail-inbox" className="text-body font-medium">
            {t(($) => $.email.switch_label)}
          </Label>
          {address ? (
            <p className="text-caption text-muted-foreground">
              <code>{address}</code>
            </p>
          ) : null}
        </div>
        <Switch
          id="agentmail-inbox"
          checked
          onCheckedChange={(value) => {
            if (!value) setRevokeStep("choose");
          }}
          disabled={pending}
        />
      </div>
      <MailViewer agentId={agent.id} />
      <AlertDialog
        open={revokeStep === "choose"}
        onOpenChange={(open) => {
          if (!open) setRevokeStep(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t(($) => $.email.revoke_title)}</AlertDialogTitle>
            <AlertDialogDescription>{t(($) => $.email.revoke_description)}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t(($) => $.email.revoke_cancel)}</AlertDialogCancel>
            <AlertDialogAction onClick={() => void revoke(false)}>
              {t(($) => $.email.revoke_keep)}
            </AlertDialogAction>
            <AlertDialogAction
              variant="destructive"
              onClick={() => setRevokeStep("confirm")}
            >
              {t(($) => $.email.revoke_delete)}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <AlertDialog
        open={revokeStep === "confirm"}
        onOpenChange={(open) => {
          if (!open) setRevokeStep(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t(($) => $.email.revoke_delete_title)}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(($) => $.email.revoke_delete_description)}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t(($) => $.email.revoke_cancel)}</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={() => void revoke(true)}>
              {t(($) => $.email.revoke_delete_confirm)}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function MailViewer({ agentId }: { agentId: string }) {
  const { t } = useT("agents");
  const wsId = useWorkspaceId();
  const [section, setSection] = useState<MailSection>("inbox");
  const [folder, setFolder] = useState("");
  const [itemId, setItemId] = useState("");
  const [itemKind, setItemKind] = useState("");
  const foldersQuery = useQuery(agentmailFoldersOptions(wsId, agentId, true));
  const mailboxQuery = useQuery(
    agentmailMailboxOptions(
      wsId,
      agentId,
      section,
      section === "folder" ? folder : "",
      true,
    ),
  );
  const detailQuery = useQuery(
    agentmailThreadOptions(wsId, agentId, itemKind === "thread" ? itemId : ""),
  );
  const items = mailboxQuery.data?.items ?? [];
  const folders = foldersQuery.data?.folders ?? [];

  function openSection(next: MailSection, nextFolder = "") {
    setSection(next);
    setFolder(nextFolder);
    setItemId("");
    setItemKind("");
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col md:flex-row">
      <aside className="shrink-0 overflow-x-auto border-b border-surface-border p-2 md:w-52 md:overflow-y-auto md:border-b-0 md:border-r md:p-4">
        <div
          className="flex w-max min-w-full items-center gap-1 md:w-full md:flex-col md:items-stretch"
          role="tablist"
          aria-orientation="vertical"
          aria-label={t(($) => $.email.folders_aria)}
        >
          {FIXED_SECTIONS.map((row) => {
            const active = section === row.id;
            return (
              <button
                key={row.id}
                type="button"
                role="tab"
                aria-selected={active}
                onClick={() => openSection(row.id)}
                className={cn(
                  "flex h-8 shrink-0 items-center rounded-md px-2.5 text-left text-caption transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring md:w-full",
                  active
                    ? "bg-surface-selected font-medium text-surface-selected-foreground hover:bg-surface-selected"
                    : "text-muted-foreground hover:bg-surface-hover hover:text-foreground",
                )}
              >
                {t(($) => $.email.folders[row.labelKey])}
              </button>
            );
          })}
          {folders.map((name) => {
            const active = section === "folder" && folder === name;
            return (
              <button
                key={name}
                type="button"
                role="tab"
                aria-selected={active}
                onClick={() => openSection("folder", name)}
                className={cn(
                  "flex h-8 shrink-0 items-center rounded-md px-2.5 text-left text-caption transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring md:w-full",
                  active
                    ? "bg-surface-selected font-medium text-surface-selected-foreground hover:bg-surface-selected"
                    : "text-muted-foreground hover:bg-surface-hover hover:text-foreground",
                )}
              >
                {name}
              </button>
            );
          })}
        </div>
      </aside>

      <section className="min-w-0 flex-1 md:overflow-y-auto">
        <div className="mx-auto w-full max-w-3xl p-4 sm:p-6">
          {itemKind === "thread" && itemId ? (
            <ThreadDetail
              subject={detailQuery.data?.subject || t(($) => $.email.untitled)}
              messages={detailQuery.data?.messages ?? []}
              failed={detailQuery.isError}
              onBack={() => {
                setItemId("");
                setItemKind("");
              }}
            />
          ) : mailboxQuery.isError ? (
            <p className="text-body text-muted-foreground">{t(($) => $.email.viewer_load_failed)}</p>
          ) : items.length === 0 ? (
            <p className="text-body text-muted-foreground">{t(($) => $.email.viewer_empty)}</p>
          ) : (
            <div className="space-y-1">
              {items.map((item) => (
                <button
                  key={`${item.kind}:${item.id}`}
                  type="button"
                  className="flex w-full flex-col items-start gap-1 rounded-md px-2 py-2 text-left hover:bg-muted"
                  onClick={() => {
                    if (item.kind !== "thread") return;
                    setItemKind(item.kind);
                    setItemId(item.id);
                  }}
                >
                  <span className="text-body font-medium">
                    {item.subject || t(($) => $.email.untitled)}
                  </span>
                  {item.preview ? (
                    <span className="text-caption text-muted-foreground line-clamp-2">
                      {item.preview}
                    </span>
                  ) : null}
                  {item.participants.length > 0 ? (
                    <span className="text-caption text-muted-foreground">
                      {item.participants.join(", ")}
                    </span>
                  ) : null}
                </button>
              ))}
            </div>
          )}
        </div>
      </section>
    </div>
  );
}

function ThreadDetail({
  subject,
  messages,
  failed,
  onBack,
}: {
  subject: string;
  messages: { message_id: string; from: string; text: string }[];
  failed: boolean;
  onBack: () => void;
}) {
  const { t } = useT("agents");
  return (
    <div className="space-y-4">
      <Button type="button" variant="ghost" size="sm" onClick={onBack}>
        {t(($) => $.email.back)}
      </Button>
      <h3 className="text-title font-medium">{subject}</h3>
      {failed ? (
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
