import type {
  BudgetAccountState,
  BudgetScope,
  MemberRole,
  WaiverScope,
} from "@multica/core/types";

export const TICKS_PER_USD = 10_000_000_000;
export const DEFAULT_SOFTEN_PERCENT = 80;

export type SoftenChoice = { kind: "off" } | { kind: "at"; percent: number };

export type WaiverChoice =
  | { kind: "inactive" }
  | { kind: "active"; starts_at: string; ends_at: string; reason: string | null };

export function usdToTicks(usd: number): number {
  return Math.round(usd * TICKS_PER_USD);
}

export function ticksToUsd(ticks: number): number {
  return ticks / TICKS_PER_USD;
}

export function parseLimitUsd(raw: string): number | null {
  const usd = Number(raw.trim());
  if (!Number.isFinite(usd) || usd <= 0) return null;
  return usd;
}

export function parseSoftenPercent(raw: string): number | null {
  const percent = Number(raw.trim());
  if (!Number.isFinite(percent) || percent < 1 || percent > 100) return null;
  return Math.round(percent);
}

export function defaultSoften(): SoftenChoice {
  return { kind: "at", percent: DEFAULT_SOFTEN_PERCENT };
}

export function expandSoften(percent: number | null): SoftenChoice {
  if (percent == null) return { kind: "off" };
  return { kind: "at", percent };
}

export function collapseSoften(choice: SoftenChoice): number | null {
  if (choice.kind === "off") return null;
  return choice.percent;
}

export function monthWindowUtc(now: Date): { start: Date; end: Date } {
  const start = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1));
  const end = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1));
  return { start, end };
}

export function defaultWaiverWindow(now: Date): { starts_at: string; ends_at: string } {
  return {
    starts_at: now.toISOString(),
    ends_at: monthWindowUtc(now).end.toISOString(),
  };
}

export function shouldDrawBar(state: string): boolean {
  return state !== "unattributed" && state !== "pricing_incomplete";
}

export function isWaiverWriteRole(role: MemberRole | null): boolean {
  return role === "owner" || role === "admin";
}

export function canWaiveScope(scope: string): scope is WaiverScope {
  return scope === "project" || scope === "initiative";
}

export function isWaiverLive(
  waiver: { starts_at: string; ends_at: string },
  now: Date,
): boolean {
  const start = Date.parse(waiver.starts_at);
  const end = Date.parse(waiver.ends_at);
  const at = now.getTime();
  return Number.isFinite(start) && Number.isFinite(end) && start <= at && at < end;
}

export function parseBudgetAccountState(state: string): BudgetAccountState | null {
  switch (state) {
    case "ok":
    case "softened":
    case "exhausted":
    case "pricing_incomplete":
    case "unattributed":
    case "waived":
      return state;
    default:
      return null;
  }
}

export function spendPercent(spentTicks: number, limitTicks: number): number {
  if (limitTicks <= 0) return 0;
  return Math.min(100, (spentTicks / limitTicks) * 100);
}

export function formatUtcDate(iso: string, locales: Intl.LocalesArgument): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return new Intl.DateTimeFormat(locales, {
    timeZone: "UTC",
    year: "numeric",
    month: "short",
    day: "numeric",
  }).format(date);
}

export function ownerLabel(args: {
  scope: BudgetScope;
  ownerId: string;
  projects: readonly { id: string; title: string }[];
  initiatives: readonly { id: string; title: string }[];
  agents: readonly { id: string; name: string }[];
  squads: readonly { id: string; name: string }[];
}): string {
  switch (args.scope) {
    case "project":
      return args.projects.find((row) => row.id === args.ownerId)?.title ?? args.ownerId;
    case "initiative":
      return (
        args.initiatives.find((row) => row.id === args.ownerId)?.title ?? args.ownerId
      );
    case "agent":
      return args.agents.find((row) => row.id === args.ownerId)?.name ?? args.ownerId;
    case "squad":
      return args.squads.find((row) => row.id === args.ownerId)?.name ?? args.ownerId;
    default: {
      const _exhaustive: never = args.scope;
      return _exhaustive;
    }
  }
}

export function isSelectableAgent(agent: {
  archived_at: string | null;
  system_key?: string;
}): boolean {
  return !agent.archived_at && !agent.system_key;
}

export function selectableOwners(args: {
  scope: BudgetScope;
  takenOwnerIds: ReadonlySet<string>;
  projects: readonly { id: string; title: string }[];
  initiatives: readonly { id: string; title: string }[];
  agents: readonly {
    id: string;
    name: string;
    archived_at: string | null;
    system_key?: string;
  }[];
  squads: readonly { id: string; name: string; archived_at: string | null }[];
}): { id: string; label: string }[] {
  switch (args.scope) {
    case "project":
      return args.projects
        .filter((row) => !args.takenOwnerIds.has(row.id))
        .map((row) => ({ id: row.id, label: row.title }));
    case "initiative":
      return args.initiatives
        .filter((row) => !args.takenOwnerIds.has(row.id))
        .map((row) => ({ id: row.id, label: row.title }));
    case "agent":
      return args.agents
        .filter((row) => isSelectableAgent(row) && !args.takenOwnerIds.has(row.id))
        .map((row) => ({ id: row.id, label: row.name }));
    case "squad":
      return args.squads
        .filter((row) => !row.archived_at && !args.takenOwnerIds.has(row.id))
        .map((row) => ({ id: row.id, label: row.name }));
    default: {
      const _exhaustive: never = args.scope;
      return _exhaustive;
    }
  }
}
