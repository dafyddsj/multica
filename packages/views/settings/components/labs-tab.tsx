"use client";

import { useState } from "react";
import { Brain, FlaskConical } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@multica/ui/components/ui/empty";
import { Label } from "@multica/ui/components/ui/label";
import { Switch } from "@multica/ui/components/ui/switch";
import { api } from "@multica/core/api";
import { useFeatureEnabled } from "@multica/core/config";
import { MEMORY_V1_FLAG } from "@multica/core/feature-flags";
import { isWorkspaceMemoryEnabled } from "@multica/core/memory";
import { useCurrentWorkspace } from "@multica/core/paths";
import { useCurrentMember } from "@multica/core/permissions";
import { workspaceKeys } from "@multica/core/workspace/queries";
import type { Workspace } from "@multica/core/types";
import { useT } from "../../i18n";
import { SettingsCard, SettingsTab } from "./settings-layout";

export function LabsTab() {
  const { t } = useT("settings");
  const workspace = useCurrentWorkspace();
  const qc = useQueryClient();
  const flagEnabled = useFeatureEnabled(MEMORY_V1_FLAG, false);
  const member = useCurrentMember(workspace?.id ?? "");
  const canManage = member.role === "owner" || member.role === "admin";
  const enabled = isWorkspaceMemoryEnabled(workspace?.settings);
  const [saving, setSaving] = useState(false);

  async function persist(next: boolean) {
    if (!workspace || saving || !canManage) return;
    setSaving(true);
    try {
      const merged = {
        ...((workspace.settings as Record<string, unknown>) ?? {}),
        memory_enabled: next,
      };
      const updated = await api.updateWorkspace(workspace.id, { settings: merged });
      qc.setQueryData(workspaceKeys.list(), (old: Workspace[] | undefined) =>
        old?.map((ws) => (ws.id === updated.id ? updated : ws)),
      );
      toast.success(t(($) => $.auto_save.toast_saved), {
        id: "settings-auto-save",
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t(($) => $.labs.toast_failed));
    } finally {
      setSaving(false);
    }
  }

  return (
    <SettingsTab
      title={t(($) => $.page.tabs.labs)}
      description={t(($) => $.labs.page_description)}
    >
      {flagEnabled ? (
        <SettingsCard>
          <div className="flex items-start justify-between gap-4 px-4 py-3.5">
            <div className="flex items-start gap-3">
              <div className="rounded-md border bg-muted/50 p-2 text-muted-foreground">
                <Brain className="h-4 w-4" />
              </div>
              <div className="space-y-1">
                <Label htmlFor="labs-memory" className="text-body font-medium">
                  {t(($) => $.labs.memory_title)}
                </Label>
                <p className="text-caption leading-5 text-muted-foreground">
                  {t(($) => $.labs.memory_description)}
                </p>
              </div>
            </div>
            <Switch
              id="labs-memory"
              checked={enabled}
              disabled={!workspace || saving || !canManage}
              onCheckedChange={persist}
            />
          </div>
        </SettingsCard>
      ) : (
        <SettingsCard>
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <FlaskConical className="h-4 w-4" />
              </EmptyMedia>
              <EmptyTitle>{t(($) => $.labs.section_placeholder_title)}</EmptyTitle>
              <EmptyDescription>
                {t(($) => $.labs.section_placeholder_description)}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        </SettingsCard>
      )}
    </SettingsTab>
  );
}
