import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Agent, AgentRuntime } from "@multica/core/types";
import { renderWithI18n } from "../../test/i18n";
import { AgentDetailInspector } from "./agent-detail-inspector";

vi.mock("@tanstack/react-query", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@tanstack/react-query")>()),
  useQuery: () => ({ data: undefined, isSuccess: false }),
}));

vi.mock("../../common/avatar-upload-control", () => ({
  AvatarUploadControl: () => <div data-testid="avatar-upload" />,
}));

vi.mock("./inspector/model-picker", () => ({
  ModelPicker: () => <div data-testid="model-picker" />,
}));

vi.mock("./inspector/runtime-picker", () => ({
  RuntimePicker: ({ onChange }: { onChange: (id: string) => void }) => (
    <button
      type="button"
      data-testid="switch-runtime"
      onClick={() => onChange("runtime-claude")}
    >
      switch
    </button>
  ),
}));

vi.mock("./inspector/thinking-prop-row", () => ({
  ThinkingSettingField: () => <div data-testid="thinking-field" />,
}));

vi.mock("./inspector/service-tier-setting-field", () => ({
  ServiceTierSettingField: () => <div data-testid="service-tier-field" />,
}));

vi.mock("./inspector/execution-lanes-fields", () => ({
  ExecutionLanesFields: () => <div data-testid="execution-lanes" />,
}));

const gooseRuntime = {
  id: "runtime-goose",
  provider: "goose",
} as AgentRuntime;

const claudeRuntime = {
  id: "runtime-claude",
  provider: "claude",
} as AgentRuntime;

function renderInspector(
  onUpdate: (id: string, data: Record<string, unknown>) => Promise<void>,
  runtime: AgentRuntime | null,
  agentOverrides: Partial<Agent> = {},
) {
  const agent = {
    id: "agent-1",
    workspace_id: "workspace-1",
    name: "Lambda",
    description: "Test agent",
    runtime_id: runtime?.id ?? "runtime-1",
    runtime_config: {},
    ...agentOverrides,
  } as Agent;

  return renderWithI18n(
    <AgentDetailInspector
      agent={agent}
      runtime={runtime}
      runtimes={[gooseRuntime, claudeRuntime]}
      members={[]}
      currentUserId="user-1"
      canEdit
      onUpdate={onUpdate}
    />,
  );
}

describe("AgentDetailInspector Goose provider", () => {
  afterEach(() => {
    cleanup();
  });

  it("shows the field only for a Goose runtime", () => {
    renderInspector(vi.fn(async () => {}), gooseRuntime, {
      runtime_config: { goose_provider: "ollama" },
    });
    expect(screen.getByLabelText("Goose provider")).toHaveValue("ollama");
  });

  it("hides the field for other runtimes", () => {
    renderInspector(vi.fn(async () => {}), claudeRuntime, {
      runtime_config: { goose_provider: "ollama" },
    });
    expect(screen.queryByLabelText("Goose provider")).toBeNull();
  });

  it("saves on blur and skips a no-op", async () => {
    const onUpdate = vi.fn(async () => {});
    const user = userEvent.setup();
    renderInspector(onUpdate, gooseRuntime, {
      runtime_config: { goose_provider: "ollama", mode: "local" },
    });

    const field = screen.getByLabelText("Goose provider");
    await user.tab();
    expect(onUpdate).not.toHaveBeenCalled();

    await user.clear(field);
    await user.type(field, "openrouter");
    await user.tab();
    expect(onUpdate).toHaveBeenCalledWith("agent-1", {
      runtime_config: { goose_provider: "openrouter", mode: "local" },
    });
  });

  it("drops goose_provider when switching away from Goose", async () => {
    const onUpdate = vi.fn(async () => {});
    const user = userEvent.setup();
    renderInspector(onUpdate, gooseRuntime, {
      runtime_config: { goose_provider: "ollama", mode: "local" },
    });

    await user.click(screen.getByTestId("switch-runtime"));
    expect(onUpdate).toHaveBeenCalledWith(
      "agent-1",
      expect.objectContaining({
        runtime_id: "runtime-claude",
        model: "",
        runtime_config: { mode: "local" },
      }),
    );
  });
});
