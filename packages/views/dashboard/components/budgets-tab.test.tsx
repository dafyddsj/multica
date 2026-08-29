import type { ReactNode } from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithI18n } from "../../test/i18n";
import type { Budget, BudgetWaiver, MemberRole } from "@multica/core/types";
import { TICKS_PER_USD } from "./budget-form";
import type { BudgetOwnerLists } from "./budget-form-dialog";

const budgetsRef = vi.hoisted(() => ({ current: [] as Budget[] }));
const waiversRef = vi.hoisted(() => ({ current: [] as BudgetWaiver[] }));
const roleRef = vi.hoisted(() => ({ current: "owner" as MemberRole }));
const loadingRef = vi.hoisted(() => ({ current: false }));

vi.mock("@multica/core/budgets", () => ({
  useBudgets: () => ({
    data: { budgets: budgetsRef.current },
    isLoading: loadingRef.current,
  }),
  useBudgetWaivers: () => ({
    data: { waivers: waiversRef.current },
    isLoading: loadingRef.current,
  }),
  useCreateBudget: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateBudget: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteBudget: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useCreateBudgetWaiver: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteBudgetWaiver: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@multica/core/permissions", () => ({
  useCurrentMember: () => ({
    userId: "user-1",
    role: roleRef.current,
    member: null,
    isLoading: false,
  }),
}));

vi.mock("@multica/ui/components/ui/dialog", () => ({
  Dialog: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children: ReactNode }) => <h2>{children}</h2>,
  DialogDescription: ({ children }: { children: ReactNode }) => <p>{children}</p>,
  DialogFooter: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

vi.mock("@multica/ui/components/ui/alert-dialog", () => ({
  AlertDialog: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? <div>{children}</div> : null,
  AlertDialogContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  AlertDialogHeader: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  AlertDialogTitle: ({ children }: { children: ReactNode }) => <h2>{children}</h2>,
  AlertDialogDescription: ({ children }: { children: ReactNode }) => <p>{children}</p>,
  AlertDialogFooter: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  AlertDialogAction: ({ children, onClick }: { children: ReactNode; onClick?: () => void }) => (
    <button type="button" onClick={onClick}>
      {children}
    </button>
  ),
  AlertDialogCancel: ({ children }: { children: ReactNode }) => (
    <button type="button">{children}</button>
  ),
}));

import { BudgetsTab } from "./budgets-tab";

const LISTS: BudgetOwnerLists = {
  projects: [{ id: "proj-1", title: "Alpha" }],
  initiatives: [{ id: "init-1", title: "Launch" }],
  agents: [{ id: "agent-1", name: "Writer", archived_at: null }],
  squads: [{ id: "squad-1", name: "Core", archived_at: null }],
};

function period(state: string, spentUsd = 10): Budget["current_period"] {
  return {
    period_start: "2026-08-01T00:00:00.000Z",
    period_end: "2026-09-01T00:00:00.000Z",
    spent_usd_ticks: spentUsd * TICKS_PER_USD,
    unpriced_line_count: state === "pricing_incomplete" ? 2 : 0,
    state,
  };
}

function budget(partial: Partial<Budget> & Pick<Budget, "id" | "scope" | "owner_id">): Budget {
  return {
    limit_usd_ticks: 50 * TICKS_PER_USD,
    soften_at_percent: 80,
    over_limit: "pause",
    current_period: period("ok"),
    ...partial,
  };
}

function renderTab() {
  return renderWithI18n(
    <BudgetsTab wsId="ws-1" locales="en-US" lists={LISTS} />,
  );
}

describe("BudgetsTab", () => {
  beforeEach(() => {
    budgetsRef.current = [];
    waiversRef.current = [];
    roleRef.current = "owner";
    loadingRef.current = false;
  });

  it("does not claim an empty list while budgets are loading", () => {
    loadingRef.current = true;
    renderTab();

    expect(screen.queryByText("No budgets yet.")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add budget" })).toBeInTheDocument();
  });

  it("shows the unattributed empty state without a spend bar", () => {
    budgetsRef.current = [
      budget({
        id: "b-unattr",
        scope: "squad",
        owner_id: "squad-1",
        current_period: period("unattributed", 0),
      }),
    ];
    renderTab();

    expect(screen.getByText("Applies to")).toBeInTheDocument();
    expect(
      screen.getByText(
        "No spend can be attributed yet. The bar stays empty until tasks are attributed here.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
  });

  it("shows an exhausted chip", () => {
    budgetsRef.current = [
      budget({
        id: "b-exh",
        scope: "project",
        owner_id: "proj-1",
        current_period: period("exhausted", 50),
      }),
    ];
    renderTab();

    expect(screen.getByText("At limit")).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toBeInTheDocument();
  });

  it("shows allow-over copy when the budget may exceed", () => {
    budgetsRef.current = [
      budget({
        id: "b-allow",
        scope: "project",
        owner_id: "proj-1",
        over_limit: "allow",
        current_period: period("ok", 10),
      }),
    ];
    renderTab();

    expect(screen.getByText("May exceed this month")).toBeInTheDocument();
  });

  it("shows the waived chip from a live waiver", () => {
    budgetsRef.current = [
      budget({
        id: "b-wav",
        scope: "project",
        owner_id: "proj-1",
        current_period: period("waived", 10),
      }),
    ];
    waiversRef.current = [
      {
        id: "w-1",
        scope: "project",
        owner_id: "proj-1",
        starts_at: "2020-01-01T00:00:00.000Z",
        ends_at: "2099-01-01T00:00:00.000Z",
        created_by: "user-1",
        reason: null,
      },
    ];
    renderTab();

    expect(screen.getByText("Limits waived until Jan 1, 2099")).toBeInTheDocument();
  });

  it("hides Bypass limits for a member on a project row", () => {
    roleRef.current = "member";
    budgetsRef.current = [
      budget({ id: "b-proj", scope: "project", owner_id: "proj-1" }),
    ];
    renderTab();

    expect(screen.queryByRole("button", { name: "Bypass limits" })).not.toBeInTheDocument();
  });

  it("hides Bypass limits on an agent row even for an owner", () => {
    roleRef.current = "owner";
    budgetsRef.current = [
      budget({ id: "b-agent", scope: "agent", owner_id: "agent-1" }),
    ];
    renderTab();

    expect(screen.queryByRole("button", { name: "Bypass limits" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
  });

  it("shows Bypass limits on a project row for an owner", () => {
    roleRef.current = "owner";
    budgetsRef.current = [
      budget({ id: "b-proj", scope: "project", owner_id: "proj-1" }),
    ];
    renderTab();

    expect(screen.getByRole("button", { name: "Bypass limits" })).toBeInTheDocument();
  });
});
