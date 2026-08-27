import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Agent } from "@multica/core/types";
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
  RuntimePicker: () => <div data-testid="runtime-picker" />,
}));

vi.mock("./inspector/thinking-prop-row", () => ({
  ThinkingSettingField: () => <div data-testid="thinking-field" />,
}));

vi.mock("./inspector/service-tier-setting-field", () => ({
  ServiceTierSettingField: () => <div data-testid="service-tier-field" />,
}));

function renderInspector(
  onUpdate: (id: string, data: Record<string, unknown>) => Promise<void>,
  agentOverrides: Partial<Agent> = {},
) {
  const agent = {
    id: "agent-1",
    workspace_id: "workspace-1",
    name: "Lambda",
    description: "Test agent",
    runtime_id: "runtime-1",
    ...agentOverrides,
  } as Agent;

  return renderWithI18n(
    <AgentDetailInspector
      agent={agent}
      runtime={null}
      runtimes={[]}
      members={[]}
      currentUserId="user-1"
      canEdit
      onUpdate={onUpdate}
    />,
  );
}

describe("AgentDetailInspector Co-authored-by email", () => {
  afterEach(() => {
    cleanup();
  });

  it("shows the optional email field", () => {
    renderInspector(vi.fn(async () => {}));

    expect(screen.getByLabelText("Co-authored-by email")).toBeInTheDocument();
    expect(screen.getByText("Git")).toBeInTheDocument();
  });

  it("saves a typed email on blur", async () => {
    const onUpdate = vi.fn(async () => {});
    const user = userEvent.setup();
    renderInspector(onUpdate);

    const field = screen.getByLabelText("Co-authored-by email");
    await user.type(field, "Review@Example.com");
    await user.tab();

    expect(onUpdate).toHaveBeenCalledWith("agent-1", {
      co_authored_by_email: "Review@Example.com",
    });
  });

  it("clears the saved email when emptied", async () => {
    const onUpdate = vi.fn(async () => {});
    const user = userEvent.setup();
    renderInspector(onUpdate, { co_authored_by_email: "old@example.com" });

    const field = screen.getByLabelText("Co-authored-by email");
    await user.clear(field);
    await user.tab();

    expect(onUpdate).toHaveBeenCalledWith("agent-1", {
      co_authored_by_email: "",
    });
  });
});
