import type { ReactNode } from "react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { I18nProvider } from "@multica/core/i18n/react";
import type { Agent } from "@multica/core/types";
import enCommon from "../../../locales/en/common.json";
import enAgents from "../../../locales/en/agents.json";

const mockGrant = vi.hoisted(() => vi.fn());
const mockRevoke = vi.hoisted(() => vi.fn());
const mockInvalidate = vi.hoisted(() => vi.fn());

const workspaceRef = vi.hoisted(() => ({
  current: { connected: true, available: true, can_manage: true, inboxes: [] },
}));
const inboxRef = vi.hoisted(() => ({
  current: { agent_id: "agent-1", enabled: false, address: "" },
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: (opts: { queryKey: unknown[] }) => {
    const key = JSON.stringify(opts.queryKey);
    if (key.includes("inbox")) return { data: inboxRef.current };
    return { data: workspaceRef.current };
  },
  useQueryClient: () => ({ invalidateQueries: mockInvalidate }),
  queryOptions: <T,>(opts: T) => opts,
}));

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "workspace-1",
}));

vi.mock("@multica/core/paths", () => ({
  useWorkspacePaths: () => ({ settings: () => "/acme/settings" }),
}));

vi.mock("@multica/core/api", () => ({
  api: {
    grantAgentMailInbox: mockGrant,
    revokeAgentMailInbox: mockRevoke,
  },
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("../../../navigation", () => ({
  AppLink: ({ href, children }: { href: string; children: ReactNode }) => (
    <a href={href}>{children}</a>
  ),
}));

import { EmailTab } from "./email-tab";

const TEST_RESOURCES = { en: { common: enCommon, agents: enAgents } };

function I18nWrapper({ children }: { children: ReactNode }) {
  return (
    <I18nProvider locale="en" resources={TEST_RESOURCES}>
      {children}
    </I18nProvider>
  );
}

const agent = { id: "agent-1", name: "Ada" } as Agent;

describe("EmailTab", () => {
  beforeEach(() => {
    mockGrant.mockReset().mockResolvedValue({ enabled: true });
    mockRevoke.mockReset().mockResolvedValue(undefined);
    mockInvalidate.mockReset();
    workspaceRef.current = { connected: true, available: true, can_manage: true, inboxes: [] };
    inboxRef.current = { agent_id: "agent-1", enabled: false, address: "" };
  });

  it("grants an inbox when the switch is turned on", async () => {
    const user = userEvent.setup();
    render(<EmailTab agent={agent} />, { wrapper: I18nWrapper });
    await user.click(screen.getByRole("switch"));
    expect(mockGrant).toHaveBeenCalledWith("agent-1");
  });

  it("points at Settings when the workspace is not connected", () => {
    workspaceRef.current = { connected: false, available: true, can_manage: true, inboxes: [] };
    render(<EmailTab agent={agent} />, { wrapper: I18nWrapper });
    expect(screen.getByRole("link", { name: "Settings → Email" })).toHaveAttribute(
      "href",
      "/acme/settings?tab=agentmail",
    );
    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
  });
});
