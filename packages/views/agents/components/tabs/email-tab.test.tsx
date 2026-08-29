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
const domainsRef = vi.hoisted(() => ({
  current: { domains: ["agentmail.to", "acme.test"] },
}));
const foldersRef = vi.hoisted(() => ({
  current: { folders: ["sales"] },
}));
const accountInboxesRef = vi.hoisted(() => ({
  current: { inboxes: [] as { inbox_id: string; email: string; linked?: boolean }[] },
}));
const mailboxRef = vi.hoisted(() => ({
  current: { items: [] as { kind: string; id: string; subject: string }[] },
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: (opts: { queryKey: unknown[] }) => {
    const key = JSON.stringify(opts.queryKey);
    if (key.includes("mailbox")) return { data: mailboxRef.current, isError: false };
    if (key.includes("folders")) return { data: foldersRef.current, isError: false };
    if (key.includes("account-inboxes")) return { data: accountInboxesRef.current, isError: false };
    if (key.includes("domains")) return { data: domainsRef.current, isError: false };
    if (key.includes("thread")) return { data: { threads: [], messages: [] }, isError: false };
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

const agent = { id: "agent-1", name: "Ada Mail" } as Agent;

describe("EmailTab", () => {
  beforeEach(() => {
    mockGrant.mockReset().mockResolvedValue({ enabled: true });
    mockRevoke.mockReset().mockResolvedValue(undefined);
    mockInvalidate.mockReset();
    workspaceRef.current = { connected: true, available: true, can_manage: true, inboxes: [] };
    inboxRef.current = { agent_id: "agent-1", enabled: false, address: "" };
    domainsRef.current = { domains: ["agentmail.to", "acme.test"] };
    foldersRef.current = { folders: ["sales"] };
    mailboxRef.current = { items: [] };
    accountInboxesRef.current = { inboxes: [] };
  });

  it("creates an inbox with the chosen username and domain", async () => {
    const user = userEvent.setup();
    render(<EmailTab agent={agent} />, { wrapper: I18nWrapper });
    const username = screen.getByLabelText("Address");
    await user.clear(username);
    await user.type(username, "ada");
    await user.click(screen.getByRole("button", { name: "Create inbox" }));
    expect(mockGrant).toHaveBeenCalledWith("agent-1", {
      mode: "create",
      username: "ada",
      domain: "agentmail.to",
    });
  });

  it("links an existing inbox from the account", async () => {
    const user = userEvent.setup();
    accountInboxesRef.current = {
      inboxes: [{ inbox_id: "inb_keep", email: "kept@agentmail.to" }],
    };
    render(<EmailTab agent={agent} />, { wrapper: I18nWrapper });
    await user.click(screen.getByRole("tab", { name: "Link existing" }));
    await user.click(screen.getByRole("button", { name: "Link inbox" }));
    expect(mockGrant).toHaveBeenCalledWith("agent-1", {
      mode: "link",
      inbox_id: "inb_keep",
    });
  });

  it("shows mailbox folders when the inbox is active", () => {
    inboxRef.current = { agent_id: "agent-1", enabled: true, address: "ada@agentmail.to" };
    render(<EmailTab agent={agent} />, { wrapper: I18nWrapper });
    expect(screen.getByText("ada@agentmail.to")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Inbox" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Sent" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Drafts" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Scheduled" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "All Mail" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Trash" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "sales" })).toBeInTheDocument();
    expect(screen.getByText("No messages yet.")).toBeInTheDocument();
  });

  it("points at Settings when the workspace is not connected", () => {
    workspaceRef.current = { connected: false, available: true, can_manage: true, inboxes: [] };
    render(<EmailTab agent={agent} />, { wrapper: I18nWrapper });
    expect(screen.getByRole("link", { name: "Settings → Email" })).toHaveAttribute(
      "href",
      "/acme/settings?tab=agentmail",
    );
    expect(screen.queryByRole("button", { name: "Create inbox" })).not.toBeInTheDocument();
  });

  it("asks whether to keep or delete the inbox, and delete needs a second confirm", async () => {
    const user = userEvent.setup();
    inboxRef.current = { agent_id: "agent-1", enabled: true, address: "ada@agentmail.to" };
    render(<EmailTab agent={agent} />, { wrapper: I18nWrapper });
    await user.click(screen.getByRole("switch"));
    expect(screen.getByRole("heading", { name: "Turn off email?" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Keep inbox" }));
    expect(mockRevoke).toHaveBeenCalledWith("agent-1", { deleteRemote: false });

    mockRevoke.mockClear();
    await user.click(screen.getByRole("switch"));
    await user.click(screen.getByRole("button", { name: "Delete inbox" }));
    expect(screen.getByRole("heading", { name: "Delete this inbox?" })).toBeInTheDocument();
    expect(mockRevoke).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Delete inbox" }));
    expect(mockRevoke).toHaveBeenCalledWith("agent-1", { deleteRemote: true });
  });
});
