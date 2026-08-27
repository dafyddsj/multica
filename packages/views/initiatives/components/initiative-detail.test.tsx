import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Initiative, Project } from "@multica/core/types";
import { renderWithI18n } from "../../test/i18n";
import { NavigationProvider, type NavigationAdapter } from "../../navigation";
import { InitiativeDetail } from "./initiative-detail";

const mocks = vi.hoisted(() => ({
  projects: [] as Project[],
  push: vi.fn(),
  deleteInitiative: vi.fn(),
  toastSuccess: vi.fn(),
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: (options: { queryKey?: readonly unknown[] }) => {
    const key = options.queryKey ?? [];
    if (key[0] === "initiative-detail") return { data: INITIATIVE, isLoading: false };
    if (key[0] === "projects") return { data: mocks.projects, isLoading: false };
    if (key[0] === "members") {
      return { data: [{ user_id: "user-1", name: "User One", role: "admin" }], isLoading: false };
    }
    if (key[0] === "agents" || key[0] === "pins") return { data: [], isLoading: false };
    return { data: undefined, isLoading: false };
  },
}));

vi.mock("@multica/core/initiatives/queries", () => ({
  initiativeDetailOptions: () => ({ queryKey: ["initiative-detail"] }),
}));

vi.mock("@multica/core/initiatives/mutations", () => ({
  useUpdateInitiative: () => ({ mutate: vi.fn() }),
  useDeleteInitiative: () => ({ mutate: mocks.deleteInitiative }),
}));

vi.mock("@multica/core/projects/queries", () => ({
  projectListOptions: () => ({ queryKey: ["projects"] }),
}));

vi.mock("@multica/core/projects/mutations", () => ({
  useUpdateProject: () => ({ mutate: vi.fn() }),
}));

vi.mock("@multica/core/pins", () => ({
  pinListOptions: () => ({ queryKey: ["pins"] }),
  useCreatePin: () => ({ mutate: vi.fn() }),
  useDeletePin: () => ({ mutate: vi.fn() }),
}));

vi.mock("@multica/core/workspace/queries", () => ({
  memberListOptions: () => ({ queryKey: ["members"] }),
  agentListOptions: () => ({ queryKey: ["agents"] }),
}));

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "workspace-1",
}));

vi.mock("@multica/core/auth", () => ({
  useAuthStore: (selector: (state: { user: { id: string } }) => unknown) =>
    selector({ user: { id: "user-1" } }),
}));

vi.mock("../../memory", () => ({
  MemoryPanel: () => null,
}));

vi.mock("@multica/core/paths", () => ({
  useWorkspacePaths: () => ({
    initiatives: () => "/test-workspace/initiatives",
    projectDetail: (id: string) => `/test-workspace/projects/${id}`,
  }),
  useCurrentWorkspace: () => ({ id: "workspace-1", settings: {} }),
}));

vi.mock("@multica/core/workspace/hooks", () => ({
  useActorName: () => ({ getActorName: () => "User One" }),
}));

vi.mock("@multica/core/modals", () => ({
  useModalStore: { getState: () => ({ open: vi.fn() }) },
}));

vi.mock("sonner", () => ({
  toast: { success: mocks.toastSuccess, error: vi.fn() },
}));

vi.mock("react-resizable-panels", () => ({
  useDefaultLayout: () => ({ defaultLayout: undefined, onLayoutChanged: vi.fn() }),
  usePanelRef: () => ({
    current: { isCollapsed: () => false, expand: vi.fn(), collapse: vi.fn() },
  }),
}));

vi.mock("@multica/ui/hooks/use-mobile", () => ({
  useIsMobile: () => false,
}));

vi.mock("@multica/ui/components/common/emoji-picker", () => ({
  EmojiPicker: () => null,
}));

vi.mock("@multica/ui/components/ui/resizable", () => ({
  ResizablePanelGroup: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ResizablePanel: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  ResizableHandle: () => null,
}));

vi.mock("@multica/ui/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  DropdownMenuTrigger: ({ render }: { render: React.ReactNode }) => <>{render}</>,
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
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
  DropdownMenuSeparator: () => <hr />,
}));

vi.mock("@multica/ui/components/ui/popover", () => ({
  Popover: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  PopoverTrigger: ({ render }: { render: React.ReactNode }) => <>{render}</>,
  PopoverContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock("@multica/ui/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipTrigger: ({ render }: { render: React.ReactNode }) => <>{render}</>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock("@multica/ui/components/ui/sheet", () => ({
  Sheet: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  SheetContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock("@multica/ui/components/ui/alert-dialog", () => ({
  AlertDialog: ({
    open,
    children,
  }: {
    open: boolean;
    children: React.ReactNode;
  }) => (open ? <div role="alertdialog">{children}</div> : null),
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => <h2>{children}</h2>,
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => <p>{children}</p>,
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogCancel: ({ children }: { children: React.ReactNode }) => (
    <button type="button">{children}</button>
  ),
  AlertDialogAction: ({
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
}));

vi.mock("../../editor", () => ({
  TitleEditor: ({ defaultValue }: { defaultValue: string }) => <div>{defaultValue}</div>,
  ContentEditor: () => null,
}));

vi.mock("../../common/entity-status-picker", () => ({
  useEntityStatusPicker: () => ({
    options: [
      {
        key: "planned",
        label: "Planned",
        category: "planned",
        hex: null,
        dotClass: "bg-muted-foreground",
        badgeBg: "bg-muted",
        badgeText: "text-muted-foreground",
      },
    ],
    current: (key: string) => ({
      key,
      label: key,
      category: "planned",
      hex: null,
      dotClass: "bg-muted-foreground",
      badgeBg: "bg-muted",
      badgeText: "text-muted-foreground",
    }),
  }),
}));

vi.mock("../../common/actor-avatar", () => ({
  ActorAvatar: () => null,
}));

vi.mock("../../layout/breadcrumb-header", () => ({
  BreadcrumbHeader: ({ actions }: { actions: React.ReactNode }) => <header>{actions}</header>,
}));

vi.mock("../../layout/animated-right-sidebar", () => ({
  AnimatedRightSidebar: ({ children }: { children: React.ReactNode }) => <aside>{children}</aside>,
  getAnimatedRightSidebarInitialOpen: () => true,
  rightSidebarPanelMotionProps: {},
  useRightSidebarShortcut: vi.fn(),
  useAnimatedRightSidebarState: () => ({
    open: true,
    visualOpen: true,
    motionEnabled: false,
    beginToggle: vi.fn(),
    handleResize: vi.fn(),
  }),
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
  issue_prefix: null,
  created_at: "2026-06-01T00:00:00Z",
  updated_at: "2026-06-01T00:00:00Z",
  project_count: 1,
  issue_count: 4,
  done_count: 1,
};

const CHILD_PROJECT: Project = {
  id: "project-1",
  workspace_id: "workspace-1",
  title: "Week 16 Sprint",
  description: null,
  icon: null,
  status: "in_progress",
  priority: "none",
  lead_type: null,
  lead_id: null,
  start_date: null,
  due_date: null,
  created_at: "2026-06-01T00:00:00Z",
  updated_at: "2026-06-01T00:00:00Z",
  issue_count: 4,
  done_count: 1,
  resource_count: 0,
  initiative_id: "initiative-1",
};

function renderDetail() {
  const adapter: NavigationAdapter = {
    push: mocks.push,
    replace: vi.fn(),
    back: vi.fn(),
    pathname: "/test-workspace/initiatives/initiative-1",
    searchParams: new URLSearchParams(),
    getShareableUrl: (path) => path,
  };
  renderWithI18n(
    <NavigationProvider value={adapter}>
      <InitiativeDetail initiativeId={INITIATIVE.id} />
    </NavigationProvider>,
  );
}

beforeEach(() => {
  mocks.projects = [CHILD_PROJECT];
  mocks.push.mockReset();
  mocks.deleteInitiative.mockReset();
  mocks.toastSuccess.mockReset();
});

describe("InitiativeDetail", () => {
  it("lists child projects instead of an issue board", () => {
    renderDetail();
    expect(screen.getByText("Week 16 Sprint")).toBeInTheDocument();
    expect(screen.queryByText("No projects yet")).not.toBeInTheDocument();
  });

  it("shows an empty projects state when nothing is attached", () => {
    mocks.projects = [];
    renderDetail();
    expect(screen.getByText("No projects yet")).toBeInTheDocument();
  });

  it("requires confirmation and navigates only after deletion succeeds", async () => {
    const user = userEvent.setup();
    renderDetail();

    await user.click(screen.getByRole("button", { name: "Delete initiative" }));
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    expect(mocks.deleteInitiative).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Delete" }));
    expect(mocks.deleteInitiative).toHaveBeenCalledWith(
      INITIATIVE.id,
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
    expect(mocks.push).not.toHaveBeenCalled();

    const options = mocks.deleteInitiative.mock.calls[0]?.[1] as {
      onSuccess: () => void;
    };
    options.onSuccess();

    expect(mocks.toastSuccess).toHaveBeenCalledWith("Initiative deleted");
    expect(mocks.push).toHaveBeenCalledWith("/test-workspace/initiatives");
  });
});
