import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Initiative } from "@multica/core/types";
import { renderWithI18n } from "../../test/i18n";
import { NavigationProvider, type NavigationAdapter } from "../../navigation";
import { InitiativesPage } from "./initiatives-page";

const mocks = vi.hoisted(() => ({
  initiatives: [] as Initiative[],
  members: [] as Array<{ user_id: string; name: string; role: string }>,
  pins: [] as Array<{ item_type: string; item_id: string }>,
  openModal: vi.fn(),
  initiativeViewState: {
    viewMode: "compact",
    sortField: "name",
    sortDirection: "asc",
    hiddenColumns: [] as string[],
    filters: { statuses: [], priorities: [], leads: [] },
    setViewMode: vi.fn(),
    toggleSort: vi.fn(),
    setSortField: vi.fn(),
    setSortDirection: vi.fn(),
    toggleColumn: vi.fn(),
    toggleFilter: vi.fn(),
    clearFilters: vi.fn(),
  },
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: (options: { queryKey?: readonly unknown[] }) => {
    const key = options.queryKey?.[0];
    if (key === "initiatives") return { data: mocks.initiatives, isLoading: false };
    if (key === "members") return { data: mocks.members, isLoading: false };
    if (key === "pins") return { data: mocks.pins, isLoading: false };
    return { data: [], isLoading: false };
  },
}));

vi.mock("@multica/core/initiatives", () => ({
  initiativeListOptions: () => ({ queryKey: ["initiatives"] }),
  useUpdateInitiative: () => ({ mutate: vi.fn() }),
  useDeleteInitiative: () => ({ mutate: vi.fn() }),
  useInitiativeViewStore: Object.assign(
    (selector: (state: unknown) => unknown) => selector(mocks.initiativeViewState),
    { getState: () => mocks.initiativeViewState },
  ),
}));

vi.mock("@multica/core/pins", () => ({
  pinListOptions: () => ({ queryKey: ["pins"] }),
  useCreatePin: () => ({ mutate: vi.fn() }),
  useDeletePin: () => ({ mutate: vi.fn() }),
}));

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "workspace-1",
}));

vi.mock("@multica/core/paths", () => ({
  useWorkspacePaths: () => ({
    initiativeDetail: (id: string) => `/test-workspace/initiatives/${id}`,
    memberDetail: (id: string) => `/test-workspace/members/${id}`,
    agentDetail: (id: string) => `/test-workspace/agents/${id}`,
  }),
}));

vi.mock("@multica/core/auth", () => ({
  useAuthStore: (selector: (state: unknown) => unknown) =>
    selector({ user: { id: "user-1" } }),
}));

vi.mock("@multica/core/workspace/queries", () => ({
  memberListOptions: () => ({ queryKey: ["members"] }),
  agentListOptions: () => ({ queryKey: ["agents"] }),
}));

vi.mock("@multica/core/workspace/hooks", () => ({
  useActorName: () => ({
    getActorName: () => "Test Lead",
    getActorInitials: () => "TL",
    getActorAvatarUrl: () => null,
  }),
}));

vi.mock("@multica/core/modals", () => ({
  useModalStore: {
    getState: () => ({ open: mocks.openModal }),
  },
}));

vi.mock("@multica/ui/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuTrigger: ({ render }: { render: React.ReactNode }) => <>{render}</>,
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuLabel: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuItem: ({
    children,
    onClick,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
  }) => (
    <button type="button" onClick={onClick}>
      {children}
    </button>
  ),
  DropdownMenuCheckboxItem: ({ children }: { children: React.ReactNode }) => (
    <button type="button">{children}</button>
  ),
  DropdownMenuRadioGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuRadioItem: ({ children }: { children: React.ReactNode }) => (
    <button type="button">{children}</button>
  ),
  DropdownMenuSeparator: () => <hr />,
  DropdownMenuSub: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuSubContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DropdownMenuSubTrigger: ({ children }: { children: React.ReactNode }) => (
    <button type="button">{children}</button>
  ),
}));

vi.mock("@multica/ui/components/ui/popover", () => ({
  Popover: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  PopoverTrigger: ({ render }: { render: React.ReactNode }) => <>{render}</>,
  PopoverContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock("@multica/ui/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipTrigger: ({ render }: { render: React.ReactNode }) => <>{render}</>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => (
    <div role="tooltip">{children}</div>
  ),
}));

const INITIATIVE: Initiative = {
  id: "initiative-1",
  workspace_id: "workspace-1",
  title: "Platform",
  description: null,
  icon: "🎯",
  status: "in_progress",
  priority: "high",
  lead_type: null,
  lead_id: null,
  start_date: null,
  due_date: null,
  created_at: "2026-06-01T00:00:00Z",
  updated_at: "2026-06-01T00:00:00Z",
  project_count: 2,
  issue_count: 8,
  done_count: 3,
};

function makeAdapter(overrides: Partial<NavigationAdapter> = {}): NavigationAdapter {
  return {
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
    pathname: "/test-workspace/initiatives",
    searchParams: new URLSearchParams(),
    getShareableUrl: (p) => p,
    ...overrides,
  };
}

function renderPage(adapter = makeAdapter()) {
  renderWithI18n(
    <NavigationProvider value={adapter}>
      <InitiativesPage />
    </NavigationProvider>,
  );
  return adapter;
}

beforeEach(() => {
  mocks.initiatives = [INITIATIVE];
  mocks.members = [{ user_id: "user-1", name: "User One", role: "admin" }];
  mocks.pins = [];
  mocks.openModal.mockClear();
  mocks.initiativeViewState.viewMode = "compact";
});

describe("InitiativesPage", () => {
  it("renders the initiative name as text and navigates from the row", async () => {
    const user = userEvent.setup();
    const push = vi.fn();
    renderPage(makeAdapter({ push }));

    const row = screen.getByText(INITIATIVE.title).closest('[role="row"]');
    expect(row).not.toBeNull();
    expect(within(row as HTMLElement).getByText(INITIATIVE.title).tagName).toBe("SPAN");

    await user.click(row as HTMLElement);
    expect(push).toHaveBeenCalledWith("/test-workspace/initiatives/initiative-1");
  });

  it("opens the create-initiative modal from the page action", async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(screen.getByRole("button", { name: "New initiative" }));
    expect(mocks.openModal).toHaveBeenCalledWith("create-initiative");
  });

  it("shows an empty state when the workspace has no initiatives", () => {
    mocks.initiatives = [];
    renderPage();
    expect(screen.getByText("No initiatives yet")).toBeInTheDocument();
  });
});
