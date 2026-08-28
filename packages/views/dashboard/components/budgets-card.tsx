"use client";

import { useMemo, useState } from "react";
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
import { Textarea } from "@multica/ui/components/ui/textarea";
import { useT } from "../../i18n";
import { BudgetFormDialog, type BudgetFormSubmit, type BudgetOwnerLists } from "./budget-form-dialog";
import {
  canWaiveScope,
  defaultWaiverWindow,
  formatUtcDate,
  isWaiverLive,
  isWaiverWriteRole,
  ownerLabel,
  parseBudgetAccountState,
  shouldDrawBar,
  spendPercent,
  ticksToUsd,
} from "./budget-form";

export function BudgetsCard({
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
    <section className="rounded-lg border bg-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h2 className="text-title font-medium">{t(($) => $.budgets.title)}</h2>
          <p className="mt-1 text-caption text-muted-foreground">
            {t(($) => $.budgets.caption)}
          </p>
        </div>
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

      {budgets.length === 0 ? (
        <p className="mt-4 text-caption text-muted-foreground">{t(($) => $.budgets.empty)}</p>
      ) : (
        <ul className="mt-4 space-y-3" aria-label={t(($) => $.budgets.title)}>
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
    </section>
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
    <li className="rounded-md border border-border p-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
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
          {showBar ? (
            <div className="mt-2 flex items-baseline gap-1 text-caption text-muted-foreground">
              <CurrencyNumberFlow value={spentUsd} locales={locales} />
              <span>/</span>
              <CurrencyNumberFlow value={limitUsd} locales={locales} />
            </div>
          ) : null}
        </div>
        <div className="flex flex-wrap items-center gap-1">
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
      </div>
      {showBar && period ? (
        <Progress
          className="mt-3"
          value={spendPercent(period.spent_usd_ticks, budget.limit_usd_ticks)}
          aria-label={t(($) => $.budgets.bar_label)}
        />
      ) : null}
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
