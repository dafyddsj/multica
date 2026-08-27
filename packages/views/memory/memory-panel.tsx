"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Brain, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { Textarea } from "@multica/ui/components/ui/textarea";
import { useFeatureEnabled } from "@multica/core/config";
import { MEMORY_V1_FLAG } from "@multica/core/feature-flags";
import { useWorkspaceId } from "@multica/core/hooks";
import { useCurrentWorkspace } from "@multica/core/paths";
import {
  isWorkspaceMemoryEnabled,
  memoryListOptions,
  useCreateMemory,
  useForgetMemory,
  type MemoryKind,
  type MemoryScope,
} from "@multica/core/memory";
import { useT } from "../i18n";

const KINDS: MemoryKind[] = ["fact", "preference", "procedure", "observation"];

export function MemoryPanel({
  scope,
  ownerId,
  canWrite = true,
  compact = false,
  hideHeading = false,
}: {
  scope: MemoryScope;
  ownerId: string;
  canWrite?: boolean;
  compact?: boolean;
  hideHeading?: boolean;
}) {
  const { t } = useT("settings");
  const flagEnabled = useFeatureEnabled(MEMORY_V1_FLAG, false);
  const workspace = useCurrentWorkspace();
  const wsId = useWorkspaceId();
  const labsEnabled = isWorkspaceMemoryEnabled(workspace?.settings);
  const [query, setQuery] = useState("");
  const [body, setBody] = useState("");
  const [kind, setKind] = useState<MemoryKind>("fact");
  const { data, isLoading } = useQuery(
    memoryListOptions(wsId, scope, ownerId, query.trim() || undefined),
  );
  const createMemory = useCreateMemory();
  const forgetMemory = useForgetMemory();
  const entries = data?.entries ?? [];
  const kindLabels = useMemo(
    () => ({
      fact: t(($) => $.memory.kind_fact),
      preference: t(($) => $.memory.kind_preference),
      procedure: t(($) => $.memory.kind_procedure),
      observation: t(($) => $.memory.kind_observation),
    }),
    [t],
  );

  if (!flagEnabled || !labsEnabled || !ownerId) {
    return null;
  }

  async function handleAdd() {
    const next = body.trim();
    if (!next) return;
    try {
      await createMemory.mutateAsync({ scope, owner_id: ownerId, body: next, kind });
      setBody("");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t(($) => $.memory.toast_failed));
    }
  }

  async function handleForget(id: string) {
    try {
      await forgetMemory.mutateAsync(id);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t(($) => $.memory.toast_failed));
    }
  }

  return (
    <section className={compact ? "space-y-2" : "space-y-3"}>
      {hideHeading ? null : (
        <div className="flex items-center gap-2 px-2">
          <Brain className="h-3.5 w-3.5 text-muted-foreground" />
          <h3 className="text-caption font-medium">{t(($) => $.memory.title)}</h3>
          {data?.total ? (
            <span className="text-caption text-muted-foreground">{data.total}</span>
          ) : null}
        </div>
      )}
      <Input
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder={t(($) => $.memory.search_placeholder)}
        aria-label={t(($) => $.memory.search_placeholder)}
        className="h-8"
      />
      {isLoading ? (
        <p className="px-2 text-caption text-muted-foreground">{t(($) => $.memory.loading)}</p>
      ) : entries.length === 0 ? (
        <p className="px-2 text-caption text-muted-foreground">{t(($) => $.memory.empty)}</p>
      ) : (
        <ul className="space-y-2">
          {entries.map((entry) => (
            <li
              key={entry.id}
              className="flex items-start justify-between gap-2 rounded-md border border-surface-border px-2 py-1.5"
            >
              <div className="min-w-0">
                <p className="text-caption text-muted-foreground">
                  {kindLabels[entry.kind] ?? entry.kind}
                </p>
                <p className="text-body whitespace-pre-wrap break-words">{entry.body}</p>
              </div>
              {canWrite ? (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t(($) => $.memory.forget)}
                  onClick={() => handleForget(entry.id)}
                  disabled={forgetMemory.isPending}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              ) : null}
            </li>
          ))}
        </ul>
      )}
      {canWrite ? (
        <div className="space-y-2">
          <Textarea
            value={body}
            onChange={(event) => setBody(event.target.value)}
            placeholder={t(($) => $.memory.add_placeholder)}
            rows={compact ? 2 : 3}
            className="resize-y"
          />
          <div className="flex items-center justify-between gap-2">
            <select
              className="h-8 rounded-md border border-input bg-background px-2 text-caption"
              value={kind}
              onChange={(event) => setKind(event.target.value as MemoryKind)}
              aria-label={t(($) => $.memory.kind_label)}
            >
              {KINDS.map((value) => (
                <option key={value} value={value}>
                  {kindLabels[value]}
                </option>
              ))}
            </select>
            <Button
              type="button"
              size="sm"
              onClick={handleAdd}
              disabled={!body.trim() || createMemory.isPending}
            >
              {t(($) => $.memory.add)}
            </Button>
          </div>
        </div>
      ) : null}
    </section>
  );
}
