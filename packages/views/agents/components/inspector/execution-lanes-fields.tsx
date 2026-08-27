"use client";

import { AGENT_EXECUTION_LANES_FLAG } from "@multica/core/feature-flags";
import { useFeatureEnabled } from "@multica/core/config";
import type { AgentRuntime, MemberWithUser } from "@multica/core/types";
import { Switch } from "@multica/ui/components/ui/switch";
import { SettingsRow } from "../../../settings/components/settings-layout";
import { useT } from "../../../i18n";
import { ModelPicker } from "./model-picker";
import { RuntimePicker } from "./runtime-picker";
import { ThinkingSettingField } from "./thinking-prop-row";
import { ServiceTierSettingField } from "./service-tier-setting-field";

export interface ExecutionLanesValue {
  lightweightModel: string;
  lightweightThinkingLevel: string;
  startLightweight: boolean;
  failoverRuntimeId: string;
  failoverModel: string;
  failoverThinkingLevel: string;
  failoverServiceTier: string;
}

export function ExecutionLanesFields({
  primaryRuntimeId,
  runtimes,
  members,
  currentUserId,
  canEdit,
  canDiscoverPrimary,
  value,
  onChange,
}: {
  primaryRuntimeId: string;
  runtimes: AgentRuntime[];
  members: MemberWithUser[];
  currentUserId: string | null;
  canEdit: boolean;
  canDiscoverPrimary: boolean;
  value: ExecutionLanesValue;
  onChange: (next: Partial<ExecutionLanesValue>) => void;
}) {
  const enabled = useFeatureEnabled(AGENT_EXECUTION_LANES_FLAG, false);
  const { t } = useT("agents");
  if (!enabled) return null;

  const failoverRuntimeId = value.failoverRuntimeId || primaryRuntimeId;
  const failoverRuntime = runtimes.find((runtime) => runtime.id === failoverRuntimeId);
  const canDiscoverFailover = failoverRuntime?.status === "online";

  return (
    <>
      <SettingsRow
        label={t(($) => $.inspector.prop_lightweight_model)}
        size="select-wide"
      >
        <ModelPicker
          variant="field"
          showLabel={false}
          runtimeId={primaryRuntimeId}
          runtimeOnline={canDiscoverPrimary}
          value={value.lightweightModel}
          canEdit={canEdit}
          emptyLabel={t(($) => $.pickers.model_none)}
          onChange={(lightweightModel) =>
            onChange({
              lightweightModel,
              lightweightThinkingLevel:
                lightweightModel === value.lightweightModel
                  ? value.lightweightThinkingLevel
                  : "",
            })
          }
        />
      </SettingsRow>
      <ThinkingSettingField
        label={t(($) => $.inspector.prop_lightweight_thinking)}
        runtimeId={primaryRuntimeId}
        runtimeOnline={canDiscoverPrimary}
        provider={
          runtimes.find((runtime) => runtime.id === primaryRuntimeId)?.provider ??
          ""
        }
        model={value.lightweightModel}
        value={value.lightweightThinkingLevel}
        canEdit={canEdit}
        onChange={(lightweightThinkingLevel) =>
          onChange({ lightweightThinkingLevel })
        }
      />
      <SettingsRow
        label={t(($) => $.inspector.prop_start_lightweight)}
        size="select-wide"
      >
        <Switch
          checked={value.startLightweight}
          disabled={!canEdit || !value.lightweightModel}
          onCheckedChange={(startLightweight) => onChange({ startLightweight })}
          aria-label={t(($) => $.inspector.prop_start_lightweight)}
        />
      </SettingsRow>
      <SettingsRow
        label={t(($) => $.inspector.prop_failover_runtime)}
        size="select-wide"
      >
        <RuntimePicker
          variant="field"
          showLabel={false}
          value={value.failoverRuntimeId}
          runtimes={runtimes}
          members={members}
          currentUserId={currentUserId}
          canEdit={canEdit}
          onChange={(failoverRuntimeId) =>
            onChange({
              failoverRuntimeId,
              failoverModel: "",
              failoverThinkingLevel: "",
              failoverServiceTier: "",
            })
          }
        />
      </SettingsRow>
      <SettingsRow
        label={t(($) => $.inspector.prop_failover_model)}
        size="select-wide"
      >
        <ModelPicker
          variant="field"
          showLabel={false}
          runtimeId={failoverRuntimeId}
          runtimeOnline={canDiscoverFailover}
          value={value.failoverModel}
          canEdit={canEdit}
          emptyLabel={t(($) => $.pickers.model_none)}
          onChange={(failoverModel) =>
            onChange({
              failoverModel,
              failoverThinkingLevel:
                failoverModel === value.failoverModel
                  ? value.failoverThinkingLevel
                  : "",
              failoverServiceTier:
                failoverModel === value.failoverModel
                  ? value.failoverServiceTier
                  : "",
            })
          }
        />
      </SettingsRow>
      <ThinkingSettingField
        label={t(($) => $.inspector.prop_failover_thinking)}
        runtimeId={failoverRuntimeId}
        runtimeOnline={canDiscoverFailover}
        provider={failoverRuntime?.provider ?? ""}
        model={value.failoverModel}
        value={value.failoverThinkingLevel}
        canEdit={canEdit}
        onChange={(failoverThinkingLevel) => onChange({ failoverThinkingLevel })}
      />
      <ServiceTierSettingField
        label={t(($) => $.inspector.prop_failover_speed)}
        runtimeId={failoverRuntimeId}
        runtimeOnline={canDiscoverFailover}
        provider={failoverRuntime?.provider ?? ""}
        model={value.failoverModel}
        value={value.failoverServiceTier}
        canEdit={canEdit}
        onChange={(failoverServiceTier) => onChange({ failoverServiceTier })}
      />
    </>
  );
}
