import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { I18nProvider } from "@multica/core/i18n/react";
import enCommon from "../locales/en/common.json";
import enSettings from "../locales/en/settings.json";

const data = vi.hoisted(() => ({
  flagEnabled: false,
  settings: {} as Record<string, unknown>,
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({ data: { entries: [], total: 0 }, isLoading: false }),
}));
vi.mock("@multica/core/config", () => ({
  useFeatureEnabled: () => data.flagEnabled,
}));
vi.mock("@multica/core/hooks", () => ({ useWorkspaceId: () => "ws-1" }));
vi.mock("@multica/core/paths", () => ({
  useCurrentWorkspace: () => ({ id: "ws-1", settings: data.settings }),
}));
vi.mock("@multica/core/memory", async () => {
  const actual = await vi.importActual<typeof import("@multica/core/memory")>("@multica/core/memory");
  return {
    ...actual,
    memoryListOptions: () => ({ queryKey: ["memory"] }),
    useCreateMemory: () => ({ mutateAsync: vi.fn(), isPending: false }),
    useForgetMemory: () => ({ mutateAsync: vi.fn(), isPending: false }),
  };
});

import { MemoryPanel } from "./memory-panel";

const TEST_RESOURCES = { en: { common: enCommon, settings: enSettings } };

function Wrapper({ children }: { children: ReactNode }) {
  return <I18nProvider locale="en" resources={TEST_RESOURCES}>{children}</I18nProvider>;
}

describe("MemoryPanel", () => {
  it("renders nothing when the flag is off", () => {
    data.flagEnabled = false;
    data.settings = { memory_enabled: true };
    const { container } = render(
      <MemoryPanel scope="workspace" ownerId="ws-1" />,
      { wrapper: Wrapper },
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("renders nothing when Labs is off", () => {
    data.flagEnabled = true;
    data.settings = {};
    const { container } = render(
      <MemoryPanel scope="workspace" ownerId="ws-1" />,
      { wrapper: Wrapper },
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("shows the bank when both gates are on", () => {
    data.flagEnabled = true;
    data.settings = { memory_enabled: true };
    render(<MemoryPanel scope="workspace" ownerId="ws-1" />, { wrapper: Wrapper });
    expect(screen.getByText("Memory")).toBeInTheDocument();
    expect(screen.getByText("No notes in this bank yet.")).toBeInTheDocument();
  });
});
