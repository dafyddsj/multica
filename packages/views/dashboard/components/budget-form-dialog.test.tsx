import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithI18n } from "../../test/i18n";
import type { Budget } from "@multica/core/types";
import { BudgetFormDialog, type BudgetFormSubmit, type BudgetOwnerLists } from "./budget-form-dialog";
import { TICKS_PER_USD } from "./budget-form";

vi.mock("@multica/ui/components/ui/dialog", () => ({
  Dialog: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: ReactNode }) => <h2>{children}</h2>,
  DialogDescription: ({ children }: { children: ReactNode }) => <p>{children}</p>,
  DialogFooter: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

const LISTS: BudgetOwnerLists = {
  projects: [{ id: "proj-1", title: "Alpha" }],
  initiatives: [{ id: "init-1", title: "Launch" }],
  agents: [{ id: "agent-1", name: "Writer", archived_at: null }],
  squads: [{ id: "squad-1", name: "Core", archived_at: null }],
};

const EMPTY_TAKEN = {
  project: new Set<string>(),
  initiative: new Set<string>(),
  agent: new Set<string>(),
  squad: new Set<string>(),
};

function renderDialog({
  budget = null,
  onSubmit = vi.fn(async () => undefined),
}: {
  budget?: Budget | null;
  onSubmit?: (submit: BudgetFormSubmit) => Promise<void>;
} = {}) {
  const submit = vi.fn(onSubmit);
  renderWithI18n(
    <BudgetFormDialog
      open
      onOpenChange={vi.fn()}
      budget={budget}
      lists={LISTS}
      takenByScope={EMPTY_TAKEN}
      pending={false}
      onSubmit={submit}
    />,
  );
  return { onSubmit: submit };
}

describe("BudgetFormDialog", () => {
  it("labels the target picker with the selected scope", async () => {
    const user = userEvent.setup();
    renderDialog();

    expect(screen.getByRole("combobox", { name: "Project" })).toBeInTheDocument();
    await user.click(screen.getByRole("combobox", { name: "Scope" }));
    await user.click(await screen.findByRole("option", { name: "Agent" }));
    expect(screen.getByRole("combobox", { name: "Agent" })).toBeInTheDocument();
    expect(screen.queryByRole("combobox", { name: "Project" })).not.toBeInTheDocument();
  });

  it("defaults soften to 80 percent", () => {
    renderDialog();
    expect(screen.getByRole("switch", { name: "Soften at threshold" })).toBeChecked();
    expect(screen.getByLabelText("Threshold percent")).toHaveValue(80);
    expect(screen.getByRole("combobox", { name: "At the limit" })).toHaveTextContent(
      "Pause new work",
    );
  });

  it("submits a create payload with collapsed soften and pause", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderDialog();

    await user.click(screen.getByRole("combobox", { name: "Project" }));
    await user.click(await screen.findByRole("option", { name: "Alpha" }));
    await user.type(screen.getByLabelText("Monthly limit (USD)"), "50");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(onSubmit).toHaveBeenCalledWith({
      mode: "create",
      request: {
        scope: "project",
        owner_id: "proj-1",
        limit_usd_ticks: 50 * TICKS_PER_USD,
        soften_at_percent: 80,
        over_limit: "pause",
      },
    });
  });

  it("collapses soften off to null and can allow overspend", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderDialog();

    await user.click(screen.getByRole("combobox", { name: "Project" }));
    await user.click(await screen.findByRole("option", { name: "Alpha" }));
    await user.type(screen.getByLabelText("Monthly limit (USD)"), "50");
    await user.click(screen.getByRole("switch", { name: "Soften at threshold" }));
    expect(screen.queryByLabelText("Threshold percent")).not.toBeInTheDocument();
    await user.click(screen.getByRole("combobox", { name: "At the limit" }));
    await user.click(await screen.findByRole("option", { name: "Allow overspend" }));
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(onSubmit).toHaveBeenCalledWith({
      mode: "create",
      request: {
        scope: "project",
        owner_id: "proj-1",
        limit_usd_ticks: 50 * TICKS_PER_USD,
        soften_at_percent: null,
        over_limit: "allow",
      },
    });
  });
});
