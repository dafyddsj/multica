/**
 * @vitest-environment jsdom
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import en from "../../locales/en/settings.json";
import { StatusesTab } from "./statuses-tab";

vi.mock("./entity-statuses-panel", () => ({
  EntityStatusesPanel: ({ resourceType }: { resourceType: string }) => (
    <div>{`entity-panel-${resourceType}`}</div>
  ),
}));
vi.mock("./issue-statuses-tab", () => ({
  IssueStatusesTab: ({ embedded }: { embedded?: boolean }) => (
    <div>{embedded ? "issue-panel-embedded" : "issue-panel"}</div>
  ),
}));
vi.mock("../../i18n", () => ({
  useT: () => ({
    t: (accessor: (dict: unknown) => string) => accessor(en),
  }),
}));

afterEach(() => {
  cleanup();
});

describe("StatusesTab", () => {
  it("defaults to Initiatives and switches scopes like Labels", async () => {
    const user = userEvent.setup();
    render(<StatusesTab />);

    expect(screen.getByText(en.statuses.title)).toBeInTheDocument();
    expect(screen.getByText("entity-panel-initiative")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: en.statuses.scopes.project }));
    expect(screen.getByText("entity-panel-project")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: en.statuses.scopes.issue }));
    expect(screen.getByText("issue-panel-embedded")).toBeInTheDocument();
  });
});
