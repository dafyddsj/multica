import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { I18nProvider } from "@multica/core/i18n/react";
import enCommon from "../../locales/en/common.json";
import enSettings from "../../locales/en/settings.json";

const data = vi.hoisted(() => ({
  flagEnabled: false,
  settings: {} as Record<string, unknown>,
}));

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({ setQueryData: vi.fn() }),
}));
vi.mock("@multica/core/config", () => ({
  useFeatureEnabled: () => data.flagEnabled,
}));
vi.mock("@multica/core/paths", () => ({
  useCurrentWorkspace: () => ({
    id: "ws-1",
    settings: data.settings,
  }),
}));
vi.mock("@multica/core/api", () => ({
  api: { updateWorkspace: vi.fn() },
}));
vi.mock("@multica/core/permissions", () => ({
  useCurrentMember: () => ({ role: "owner", isLoading: false }),
}));

import { LabsTab } from "./labs-tab";

const TEST_RESOURCES = { en: { common: enCommon, settings: enSettings } };

function Wrapper({ children }: { children: ReactNode }) {
  return <I18nProvider locale="en" resources={TEST_RESOURCES}>{children}</I18nProvider>;
}

describe("LabsTab", () => {
  beforeEach(() => {
    data.flagEnabled = false;
    data.settings = {};
  });

  it("keeps the empty state when memory_v1 is off", () => {
    render(<LabsTab />, { wrapper: Wrapper });
    expect(screen.getByText("No experiments yet")).toBeInTheDocument();
    expect(screen.queryByLabelText("Memory")).not.toBeInTheDocument();
  });

  it("shows the memory toggle when memory_v1 is on", () => {
    data.flagEnabled = true;
    render(<LabsTab />, { wrapper: Wrapper });
    expect(screen.getByLabelText("Memory")).toBeInTheDocument();
    expect(screen.getByRole("switch")).not.toBeChecked();
  });
});
