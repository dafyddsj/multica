import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

const state = vi.hoisted(() => ({
  user: {
    id: "user-1",
    onboarded_at: null as string | null,
  },
  isLoading: false,
  workspaces: [] as { id: string; slug: string }[],
  workspacesReady: true,
  pathname: "/onboarding",
  replace: vi.fn(),
  push: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: state.replace, push: state.push }),
  usePathname: () => state.pathname,
}));

vi.mock("@multica/core/auth", () => ({
  useAuthStore: (
    selector: (auth: {
      user: typeof state.user;
      isLoading: boolean;
    }) => unknown,
  ) => selector({ user: state.user, isLoading: state.isLoading }),
}));

vi.mock("@multica/core/workspace", () => ({
  useWorkspaceList: () => ({
    workspaces: state.workspaces,
    ready: state.workspacesReady,
  }),
}));

vi.mock("@multica/views/onboarding", () => ({
  OnboardingFlow: ({
    onComplete,
  }: {
    onComplete: (
      ws?: { id: string; slug: string },
      destination?: { kind: "chat" | "issue"; sessionId?: string; issueId?: string },
    ) => void;
  }) => (
    <div>
      <button
        type="button"
        onClick={() => onComplete({ id: "ws-1", slug: "test-org" })}
      >
        skip existing
      </button>
      <button
        type="button"
        onClick={() =>
          onComplete(
            { id: "ws-1", slug: "test-org" },
            { kind: "chat", sessionId: "sess-1" },
          )
        }
      >
        finish with chat
      </button>
    </div>
  ),
  CliInstallInstructions: () => null,
}));

import OnboardingPage from "./page";

function renderPage() {
  return render(<OnboardingPage />);
}

describe("OnboardingPage", () => {
  beforeEach(() => {
    state.user = { id: "user-1", onboarded_at: null };
    state.isLoading = false;
    state.workspaces = [{ id: "ws-1", slug: "test-org" }];
    state.workspacesReady = true;
    state.pathname = "/onboarding";
    state.replace.mockReset();
    state.push.mockReset();
  });

  it("shows a spinner and replaces to the synced workspace after skip-existing", async () => {
    const { rerender } = renderPage();

    await userEvent.click(screen.getByRole("button", { name: "skip existing" }));
    expect(state.push).toHaveBeenCalledWith("/test-org/issues");

    // completeOnboarding → refreshMe flips onboarded_at while we are
    // still on /onboarding (or after the workspace layout bounced us).
    state.user = { id: "user-1", onboarded_at: "2026-08-27T10:11:14Z" };
    rerender(<OnboardingPage />);

    expect(screen.getByRole("status", { name: "Loading" })).toBeInTheDocument();
    await waitFor(() => {
      expect(state.replace).toHaveBeenCalledWith("/test-org/issues");
    });
  });

  it("still lets a chat landing win over the issues-list guard", async () => {
    const { rerender } = renderPage();

    await userEvent.click(screen.getByRole("button", { name: "finish with chat" }));
    expect(state.push).toHaveBeenCalledWith("/test-org/chat?session=sess-1");

    state.user = { id: "user-1", onboarded_at: "2026-08-27T10:11:14Z" };
    rerender(<OnboardingPage />);

    await act(async () => {});
    expect(state.replace).not.toHaveBeenCalled();
  });
});
