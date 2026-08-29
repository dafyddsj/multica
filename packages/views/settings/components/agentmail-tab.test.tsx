import type { ReactNode } from "react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { I18nProvider } from "@multica/core/i18n/react";
import { configStore } from "@multica/core/config";
import enCommon from "../../locales/en/common.json";
import enSettings from "../../locales/en/settings.json";

const mockConnect = vi.hoisted(() => vi.fn());
const mockDisconnect = vi.hoisted(() => vi.fn());
const mockInvalidate = vi.hoisted(() => vi.fn());

const workspaceRef = vi.hoisted(() => ({
  current: {
    available: true,
    hosted_available: true,
    connected: false,
    can_manage: true,
    inboxes: [] as { agent_id: string; enabled: boolean; address?: string; display_name?: string }[],
  },
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({ data: workspaceRef.current }),
  useQueryClient: () => ({
    invalidateQueries: mockInvalidate,
  }),
  queryOptions: <T,>(opts: T) => opts,
}));

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "workspace-1",
}));

vi.mock("@multica/core/api", () => ({
  api: {
    connectAgentMail: mockConnect,
    disconnectAgentMail: mockDisconnect,
  },
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { AgentMailTab } from "./agentmail-tab";

const TEST_RESOURCES = {
  en: { common: enCommon, settings: enSettings },
};

function I18nWrapper({ children }: { children: ReactNode }) {
  return (
    <I18nProvider locale="en" resources={TEST_RESOURCES}>
      {children}
    </I18nProvider>
  );
}

describe("AgentMailTab", () => {
  beforeEach(() => {
    mockConnect.mockReset();
    mockDisconnect.mockReset();
    mockInvalidate.mockReset();
    workspaceRef.current = {
      available: true,
      hosted_available: true,
      connected: false,
      can_manage: true,
      inboxes: [],
    };
    configStore.getState().setAuthConfig({
      allowSignup: true,
      agentmailAvailable: true,
      agentmailHostedAvailable: true,
    });
  });

  it("connects hosted when the admin clicks Connect hosted", async () => {
    mockConnect.mockResolvedValue({ connected: true });
    const user = userEvent.setup();
    render(<AgentMailTab />, { wrapper: I18nWrapper });

    await user.click(screen.getByRole("button", { name: "Connect hosted" }));
    expect(mockConnect).toHaveBeenCalledWith("workspace-1", { mode: "hosted" });
  });

  it("lists granted inbox addresses for members", () => {
    workspaceRef.current = {
      available: true,
      hosted_available: true,
      connected: true,
      can_manage: false,
      inboxes: [
        {
          agent_id: "agent-1",
          enabled: true,
          address: "ada@agentmail.to",
          display_name: "Ada",
        },
      ],
    };
    render(<AgentMailTab />, { wrapper: I18nWrapper });

    expect(screen.getByText("Ada")).toBeInTheDocument();
    expect(screen.getByText("ada@agentmail.to")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Disconnect" })).not.toBeInTheDocument();
  });
});
