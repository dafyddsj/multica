"use client";

import { useEffect, useMemo, useState } from "react";
import type {
  Budget,
  BudgetOverLimit,
  BudgetScope,
  CreateBudgetRequest,
  UpdateBudgetRequest,
} from "@multica/core/types";
import { Button } from "@multica/ui/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@multica/ui/components/ui/dialog";
import { Input } from "@multica/ui/components/ui/input";
import { Label } from "@multica/ui/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@multica/ui/components/ui/select";
import { Switch } from "@multica/ui/components/ui/switch";
import { useT } from "../../i18n";
import {
  DEFAULT_SOFTEN_PERCENT,
  collapseSoften,
  defaultSoften,
  expandSoften,
  parseLimitUsd,
  parseSoftenPercent,
  selectableOwners,
  ticksToUsd,
  usdToTicks,
  type SoftenChoice,
} from "./budget-form";

export type BudgetOwnerLists = {
  projects: readonly { id: string; title: string }[];
  initiatives: readonly { id: string; title: string }[];
  agents: readonly {
    id: string;
    name: string;
    archived_at: string | null;
    system_key?: string;
  }[];
  squads: readonly { id: string; name: string; archived_at: string | null }[];
};

export type BudgetFormSubmit =
  | { mode: "create"; request: CreateBudgetRequest }
  | { mode: "update"; id: string; request: UpdateBudgetRequest };

const SCOPES: BudgetScope[] = ["project", "initiative", "agent", "squad"];

export function BudgetFormDialog({
  open,
  onOpenChange,
  budget,
  lists,
  takenByScope,
  pending,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  budget: Budget | null;
  lists: BudgetOwnerLists;
  takenByScope: Record<BudgetScope, ReadonlySet<string>>;
  pending: boolean;
  onSubmit: (submit: BudgetFormSubmit) => Promise<void>;
}) {
  const { t } = useT("usage");
  const editing = budget !== null;

  const [scope, setScope] = useState<BudgetScope>("project");
  const [ownerId, setOwnerId] = useState("");
  const [limitUsd, setLimitUsd] = useState("");
  const [soften, setSoften] = useState<SoftenChoice>(defaultSoften);
  const [softenPercent, setSoftenPercent] = useState(String(DEFAULT_SOFTEN_PERCENT));
  const [overLimit, setOverLimit] = useState<BudgetOverLimit>("pause");

  useEffect(() => {
    if (!open) return;
    if (budget) {
      setScope(budget.scope);
      setOwnerId(budget.owner_id);
      setLimitUsd(String(ticksToUsd(budget.limit_usd_ticks)));
      const next = expandSoften(budget.soften_at_percent);
      setSoften(next);
      setSoftenPercent(next.kind === "at" ? String(next.percent) : "80");
      setOverLimit(budget.over_limit);
      return;
    }
    setScope("project");
    setOwnerId("");
    setLimitUsd("");
    const next = defaultSoften();
    setSoften(next);
    setSoftenPercent(String(DEFAULT_SOFTEN_PERCENT));
    setOverLimit("pause");
  }, [open, budget]);

  const owners = useMemo(
    () =>
      selectableOwners({
        scope,
        takenOwnerIds: editing ? new Set() : takenByScope[scope],
        projects: lists.projects,
        initiatives: lists.initiatives,
        agents: lists.agents,
        squads: lists.squads,
      }),
    [editing, lists, scope, takenByScope],
  );

  const scopeItems = SCOPES.map((value) => ({
    value,
    label: scopeLabel(value, {
      project: t(($) => $.budgets.scope.project),
      initiative: t(($) => $.budgets.scope.initiative),
      agent: t(($) => $.budgets.scope.agent),
      squad: t(($) => $.budgets.scope.squad),
    }),
  }));
  const ownerItems = owners.map((row) => ({ value: row.id, label: row.label }));
  const overLimitItems = [
    { value: "pause", label: t(($) => $.budgets.over_limit_pause) },
    { value: "allow", label: t(($) => $.budgets.over_limit_allow) },
  ];

  const parsedLimit = parseLimitUsd(limitUsd);
  const parsedPercent = parseSoftenPercent(softenPercent);
  const effectiveSoften: SoftenChoice =
    soften.kind === "off"
      ? { kind: "off" }
      : parsedPercent == null
        ? { kind: "at", percent: 80 }
        : { kind: "at", percent: parsedPercent };

  const canSubmit =
    parsedLimit != null &&
    (editing || ownerId !== "") &&
    (effectiveSoften.kind === "off" || parsedPercent != null);

  const handleScopeChange = (next: BudgetScope | null) => {
    if (!next || editing) return;
    setScope(next);
    setOwnerId("");
  };

  const handleSubmit = async () => {
    if (!canSubmit || parsedLimit == null) return;
    const limit_usd_ticks = usdToTicks(parsedLimit);
    const soften_at_percent = collapseSoften(effectiveSoften);
    if (budget) {
      await onSubmit({
        mode: "update",
        id: budget.id,
        request: { limit_usd_ticks, soften_at_percent, over_limit: overLimit },
      });
      return;
    }
    await onSubmit({
      mode: "create",
      request: {
        scope,
        owner_id: ownerId,
        limit_usd_ticks,
        soften_at_percent,
        over_limit: overLimit,
      },
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {editing
              ? t(($) => $.budgets.dialog_edit_title)
              : t(($) => $.budgets.dialog_create_title)}
          </DialogTitle>
          <DialogDescription>{t(($) => $.budgets.caption)}</DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-1">
          <div className="space-y-2">
            <Label htmlFor="budget-scope">{t(($) => $.budgets.scope_label)}</Label>
            <Select
              items={scopeItems}
              value={scope}
              onValueChange={handleScopeChange}
              disabled={editing}
            >
              <SelectTrigger id="budget-scope" className="w-full" aria-label={t(($) => $.budgets.scope_label)}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {scopeItems.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="budget-owner">{t(($) => $.budgets.owner_label)}</Label>
            <Select
              items={ownerItems}
              value={ownerId || null}
              onValueChange={(next) => {
                if (next && !editing) setOwnerId(next);
              }}
              disabled={editing}
            >
              <SelectTrigger id="budget-owner" className="w-full" aria-label={t(($) => $.budgets.owner_label)}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {ownerItems.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="budget-limit">{t(($) => $.budgets.limit_label)}</Label>
            <Input
              id="budget-limit"
              type="number"
              min="0"
              step="any"
              inputMode="decimal"
              value={limitUsd}
              onChange={(event) => setLimitUsd(event.target.value)}
              aria-label={t(($) => $.budgets.limit_label)}
            />
          </div>
          <div className="space-y-2">
            <div className="flex items-center justify-between gap-3">
              <Label htmlFor="budget-soften">{t(($) => $.budgets.soften_label)}</Label>
              <Switch
                id="budget-soften"
                checked={soften.kind === "at"}
                onCheckedChange={(checked) =>
                  setSoften(checked ? { kind: "at", percent: parsedPercent ?? 80 } : { kind: "off" })
                }
                aria-label={t(($) => $.budgets.soften_label)}
              />
            </div>
            <p className="text-caption text-muted-foreground">
              {t(($) => $.budgets.soften_hint)}
            </p>
            {soften.kind === "at" ? (
              <Input
                type="number"
                min="1"
                max="100"
                inputMode="numeric"
                value={softenPercent}
                onChange={(event) => setSoftenPercent(event.target.value)}
                aria-label={t(($) => $.budgets.soften_percent_label)}
              />
            ) : null}
          </div>
          <div className="space-y-2">
            <Label htmlFor="budget-over-limit">{t(($) => $.budgets.over_limit_label)}</Label>
            <Select
              items={overLimitItems}
              value={overLimit}
              onValueChange={(next) => {
                if (next === "pause" || next === "allow") setOverLimit(next);
              }}
            >
              <SelectTrigger
                id="budget-over-limit"
                className="w-full"
                aria-label={t(($) => $.budgets.over_limit_label)}
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {overLimitItems.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t(($) => $.budgets.cancel)}
          </Button>
          <Button type="button" onClick={() => void handleSubmit()} disabled={!canSubmit || pending}>
            {editing ? t(($) => $.budgets.save) : t(($) => $.budgets.create)}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function scopeLabel(
  scope: BudgetScope,
  labels: Record<BudgetScope, string>,
): string {
  return labels[scope];
}
