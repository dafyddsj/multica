"use client";

import { useMemo, useState } from "react";
import { Wallet } from "lucide-react";
import { toast } from "sonner";
import {
  useBudgetWaivers,
  useBudgets,
  useCreateBudget,
  useCreateBudgetWaiver,
  useDeleteBudget,
  useDeleteBudgetWaiver,
  useUpdateBudget,
} from "@multica/core/budgets";
import { useCurrentMember } from "@multica/core/permissions";
import type {
  Budget,
  BudgetScope,
  BudgetWaiver,
} from "@multica/core/types";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@multica/ui/components/ui/alert-dialog";
import { Badge } from "@multica/ui/components/ui/badge";
import { Button } from "@multica/ui/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@multica/ui/components/ui/dialog";
import { Label } from "@multica/ui/components/ui/label";
import { CurrencyNumberFlow } from "@multica/ui/components/ui/number-flow";
import { Progress } from "@multica/ui/components/ui/progress";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { Textarea } from "@multica/ui/components/ui/textarea";
import { useT } from "../../i18n";
import { BudgetFormDialog, type BudgetFormSubmit, type BudgetOwnerLists } from "./budget-form-dialog";
import {
  canWaiveScope,
  defaultWaiverWindow,
  formatUtcDate,
  formatUtcMonth,
  isWaiverLive,
  isWaiverWriteRole,
  ownerLabel,
  parseBudgetAccountState,
  shouldDrawBar,
  spendPercent,
  ticksToUsd,
} from "./budget-form";

const BUDGET_GRID =
  "grid grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)_auto] items-start gap-3";

export function BudgetsTab({
  wsId,
  lists,
  locales,
}: {
  wsId: string;
  lists: BudgetOwnerLists;
  locales: Intl.LocalesArgument;
}) {
  const { t } = useT("usage");
  const { role } = useCurrentMember(wsId);
  const budgetsQuery = useBudgets(wsId);
  const waiversQuery = useBudgetWaivers(wsId);
  const createBudget = useCreateBudget(wsId);
  const updateBudget = useUpdateBudget(wsId);
  const deleteBudget = useDeleteBudget(wsId);
  const createWaiver = useCreateBudgetWaiver(wsId);
  const deleteWaiver = useDeleteBudgetWaiver(wsId);

  const budgets = budgetsQuery.data?.budgets ?? [];
  const waivers = waiversQuery.data?.waivers ?? [];

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Budget | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Budget | null>(null);
  const [pendingWaiver, setPendingWaiver] = useState<Budget | null>(null);
  const [waiverReason, setWaiverReason] = useState("");

  const takenByScope = useMemo(() => {
    const taken: Record<BudgetScope, Set<string>> = {
      project: new Set(),
      initiative: new Set(),
      agent: new Set(),
      squad: new Set(),
    };
    for (const budget of budgets) {
      taken[budget.scope].add(budget.owner_id);
    }
    return taken;
  }, [budgets]);

  const liveWaiverByOwner = useMemo(() => {
    const now = new Date();
    const map = new Map<string, BudgetWaiver>();
    for (const waiver of waivers) {
      if (!isWaiverLive(waiver, now)) continue;
      map.set(`${waiver.scope}:${waiver.owner_id}`, waiver);
    }
    return map;
  }, [waivers]);

  const canWriteWaiver = isWaiverWriteRole(role);
  const monthCaption = t(($) => $.budgets.month_caption, {
    month: formatUtcMonth(new Date(), locales),
  });

  const handleSubmit = async (submit: BudgetFormSubmit) => {
    try {
      if (submit.mode === "create") {
        await createBudget.mutateAsync(submit.request);
      } else {
        await updateBudget.mutateAsync({ id: submit.id, ...submit.request });
      }
      setFormOpen(false);
      setEditing(null);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error));
    }
  };

  const handleDelete = async () => {
    if (!pendingDelete) return;
    try {
      await deleteBudget.mutateAsync(pendingDelete.id);
      setPendingDelete(null);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error));
    }
  };

  const handleCreateWaiver = async () => {
    if (!pendingWaiver || !canWaiveScope(pendingWaiver.scope)) return;
    const window = defaultWaiverWindow(new Date());
    try {
      await createWaiver.mutateAsync({
        scope: pendingWaiver.scope,
        owner_id: pendingWaiver.owner_id,
        starts_at: window.starts_at,
        ends_at: window.ends_at,
        reason: waiverReason.trim() === "" ? null : waiverReason.trim(),
      });
      setPendingWaiver(null);
      setWaiverReason("");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error));
    }
  };

  const handleEndWaiver = async (waiver: BudgetWaiver) => {
    try {
      await deleteWaiver.mutateAsync(waiver.id);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error));
    }
  };

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-3">
        <p className="text-caption text-muted-foreground">
          {t(($) => $.budgets.caption)}
        </p>
        <Button
          type="button"
          size="sm"
          onClick={() => {
            setEditing(null);
            setFormOpen(true);
          }}
        >
          {t(($) => $.budgets.add)}
        </Button>
      </div>

      {budgetsQuery.isLoading || waiversQuery.isLoading ? (
        <div className="space-y-3">
          <Skeleton className="h-28 rounded-lg" />
          <Skeleton className="h-48 rounded-lg" />
        </div>
      ) : budgets.length === 0 ? (
        <div className="flex flex-col items-center rounded-lg border border-dashed py-12 text-center">
          <Wallet className="h-6 w-6 text-faint-foreground" />
          <p className="mt-3 text-body font-medium">{t(($) => $.budgets.empty)}</p>
        </div>
      ) : (
        <div className="rounded-lg border bg-card">
          <div className="border-b px-4 pt-4 pb-3">
            <p className="text-caption text-muted-foreground">{monthCaption}</p>
          </div>
          <div
            className={`${BUDGET_GRID} border-b px-4 py-2 text-caption font-medium text-muted-foreground`}
          >
            <span>{t(($) => $.budgets.owner_label)}</span>
            <span className="text-right">{t(($) => $.budgets.header_spent)}</span>
            <span />
          </div>
          <ul aria-label={t(($) => $.budgets.title)} className="divide-y">
            {budgets.map((budget) => {
              const waiver = liveWaiverByOwner.get(`${budget.scope}:${budget.owner_id}`) ?? null;
              return (
                <BudgetRow
                  key={budget.id}
                  budget={budget}
                  waiver={waiver}
                  lists={lists}
                  locales={locales}
                  canWriteWaiver={canWriteWaiver}
                  onEdit={() => {
                    setEditing(budget);
                    setFormOpen(true);
                  }}
                  onDelete={() => setPendingDelete(budget)}
                  onBypass={() => {
                    setWaiverReason("");
                    setPendingWaiver(budget);
                  }}
                  onEndWaiver={() => {
                    if (waiver) void handleEndWaiver(waiver);
                  }}
                />
              );
            })}
          </ul>
        </div>
      )}

      <BudgetFormDialog
        open={formOpen}
        onOpenChange={(open) => {
          setFormOpen(open);
          if (!open) setEditing(null);
        }}
        budget={editing}
        lists={lists}
        takenByScope={takenByScope}
        pending={createBudget.isPending || updateBudget.isPending}
        onSubmit={handleSubmit}
      />

      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t(($) => $.budgets.delete_confirm_title)}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(($) => $.budgets.delete_confirm_body)}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t(($) => $.budgets.cancel)}</AlertDialogCancel>
            <AlertDialogAction onClick={() => void handleDelete()}>
              {t(($) => $.budgets.delete)}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog
        open={pendingWaiver !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingWaiver(null);
            setWaiverReason("");
          }
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t(($) => $.budgets.waiver.title)}</DialogTitle>
            <DialogDescription>{t(($) => $.budgets.waiver.body)}</DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="budget-waiver-reason">{t(($) => $.budgets.waiver.reason_label)}</Label>
            <Textarea
              id="budget-waiver-reason"
              value={waiverReason}
              onChange={(event) => setWaiverReason(event.target.value)}
              aria-label={t(($) => $.budgets.waiver.reason_label)}
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setPendingWaiver(null);
                setWaiverReason("");
              }}
            >
              {t(($) => $.budgets.cancel)}
            </Button>
            <Button
              type="button"
              onClick={() => void handleCreateWaiver()}
              disabled={createWaiver.isPending}
            >
              {t(($) => $.budgets.waiver.confirm)}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function BudgetRow({
  budget,
  waiver,
  lists,
  locales,
  canWriteWaiver,
  onEdit,
  onDelete,
  onBypass,
  onEndWaiver,
}: {
  budget: Budget;
  waiver: BudgetWaiver | null;
  lists: BudgetOwnerLists;
  locales: Intl.LocalesArgument;
  canWriteWaiver: boolean;
  onEdit: () => void;
  onDelete: () => void;
  onBypass: () => void;
  onEndWaiver: () => void;
}) {
  const { t } = useT("usage");
  const name = ownerLabel({
    scope: budget.scope,
    ownerId: budget.owner_id,
    projects: lists.projects,
    initiatives: lists.initiatives,
    agents: lists.agents,
    squads: lists.squads,
  });
  const period = budget.current_period;
  const state = period ? parseBudgetAccountState(period.state) : "ok";
  const showBar =
    period != null && state != null && shouldDrawBar(state);
  const spentUsd = period ? ticksToUsd(period.spent_usd_ticks) : 0;
  const limitUsd = ticksToUsd(budget.limit_usd_ticks);
  const showWaiver = canWriteWaiver && canWaiveScope(budget.scope);

  return (
    <li className={`${BUDGET_GRID} px-4 py-3`}>
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <p className="text-body font-medium">{name}</p>
          <span className="text-caption text-muted-foreground">
            {scopeLabel(budget.scope, {
              project: t(($) => $.budgets.scope.project),
              initiative: t(($) => $.budgets.scope.initiative),
              agent: t(($) => $.budgets.scope.agent),
              squad: t(($) => $.budgets.scope.squad),
            })}
          </span>
          {budget.over_limit === "allow" ? (
            <Badge variant="outline">{t(($) => $.budgets.state.allow_over)}</Badge>
          ) : null}
          <StateChip budget={budget} waiver={waiver} locales={locales} />
        </div>
        {state === "unattributed" ? (
          <p className="mt-2 text-caption text-muted-foreground">
            {t(($) => $.budgets.state.unattributed)}
          </p>
        ) : null}
        {state === "pricing_incomplete" ? (
          <p className="mt-2 text-caption text-muted-foreground">
            {t(($) => $.budgets.state.pricing_incomplete)}
          </p>
        ) : null}
      </div>
      <div className="min-w-0">
        {showBar ? (
          <>
            <div className="flex items-baseline justify-end gap-1 text-right text-caption text-muted-foreground">
              <CurrencyNumberFlow value={spentUsd} locales={locales} />
              <span>/</span>
              <CurrencyNumberFlow value={limitUsd} locales={locales} />
            </div>
            {period ? (
              <Progress
                className="mt-2"
                value={spendPercent(period.spent_usd_ticks, budget.limit_usd_ticks)}
                aria-label={t(($) => $.budgets.bar_label)}
              />
            ) : null}
          </>
        ) : null}
      </div>
      <div className="flex flex-wrap items-center justify-end gap-1">
        {showWaiver && waiver ? (
          <Button type="button" size="sm" variant="ghost" onClick={onEndWaiver}>
            {t(($) => $.budgets.waiver.end)}
          </Button>
        ) : null}
        {showWaiver && !waiver ? (
          <Button type="button" size="sm" variant="ghost" onClick={onBypass}>
            {t(($) => $.budgets.waiver.bypass)}
          </Button>
        ) : null}
        <Button type="button" size="sm" variant="ghost" onClick={onEdit}>
          {t(($) => $.budgets.edit)}
        </Button>
        <Button type="button" size="sm" variant="ghost" onClick={onDelete}>
          {t(($) => $.budgets.delete)}
        </Button>
      </div>
    </li>
  );
}

function StateChip({
  budget,
  waiver,
  locales,
}: {
  budget: Budget;
  waiver: BudgetWaiver | null;
  locales: Intl.LocalesArgument;
}) {
  const { t } = useT("usage");
  if (waiver) {
    return (
      <Badge variant="secondary">
        {t(($) => $.budgets.waiver.chip, {
          date: formatUtcDate(waiver.ends_at, locales),
        })}
      </Badge>
    );
  }
  const state = budget.current_period
    ? parseBudgetAccountState(budget.current_period.state)
    : "ok";
  switch (state) {
    case "softened":
      return <Badge variant="secondary">{t(($) => $.budgets.state.softened)}</Badge>;
    case "exhausted":
      return <Badge variant="destructive">{t(($) => $.budgets.state.exhausted)}</Badge>;
    case "waived":
      return (
        <Badge variant="secondary">
          {t(($) => $.budgets.waiver.chip, {
            date: formatUtcDate(budget.current_period?.period_end ?? "", locales),
          })}
        </Badge>
      );
    case "ok":
    case "pricing_incomplete":
    case "unattributed":
    case null:
      return null;
    default: {
      const _exhaustive: never = state;
      return _exhaustive;
    }
  }
}

function scopeLabel(scope: BudgetScope, labels: Record<BudgetScope, string>): string {
  return labels[scope];
}
