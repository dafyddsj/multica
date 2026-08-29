// @vitest-environment node
import { describe, expect, it } from "vitest";
import {
  DEFAULT_SOFTEN_PERCENT,
  TICKS_PER_USD,
  canWaiveScope,
  collapseSoften,
  defaultSoften,
  defaultWaiverWindow,
  expandSoften,
  isSelectableAgent,
  isWaiverLive,
  isWaiverWriteRole,
  monthWindowUtc,
  parseBudgetAccountState,
  parseLimitUsd,
  parseSoftenPercent,
  selectableOwners,
  shouldDrawBar,
  spendPercent,
  ticksToUsd,
  usdToTicks,
} from "./budget-form";

describe("budget form helpers", () => {
  it("converts fifty dollars to ticks and back", () => {
    expect(usdToTicks(50)).toBe(50 * TICKS_PER_USD);
    expect(ticksToUsd(50 * TICKS_PER_USD)).toBe(50);
  });

  it("rejects a non-positive limit", () => {
    expect(parseLimitUsd("")).toBeNull();
    expect(parseLimitUsd("0")).toBeNull();
    expect(parseLimitUsd("-2")).toBeNull();
    expect(parseLimitUsd("50")).toBe(50);
  });

  it("defaults soften to 80 and collapses the union on submit", () => {
    expect(defaultSoften()).toEqual({ kind: "at", percent: DEFAULT_SOFTEN_PERCENT });
    expect(collapseSoften({ kind: "at", percent: 80 })).toBe(80);
    expect(collapseSoften({ kind: "off" })).toBeNull();
    expect(expandSoften(null)).toEqual({ kind: "off" });
    expect(expandSoften(60)).toEqual({ kind: "at", percent: 60 });
    expect(parseSoftenPercent("80")).toBe(80);
    expect(parseSoftenPercent("0")).toBeNull();
    expect(parseSoftenPercent("101")).toBeNull();
  });

  it("uses the UTC calendar month with an exclusive end", () => {
    const now = new Date("2026-08-28T22:00:00.000Z");
    const window = monthWindowUtc(now);
    expect(window.start.toISOString()).toBe("2026-08-01T00:00:00.000Z");
    expect(window.end.toISOString()).toBe("2026-09-01T00:00:00.000Z");

    const waiver = defaultWaiverWindow(now);
    expect(waiver.starts_at).toBe(now.toISOString());
    expect(waiver.ends_at).toBe("2026-09-01T00:00:00.000Z");
  });

  it("does not draw a bar for unattributed or pricing_incomplete", () => {
    expect(shouldDrawBar("ok")).toBe(true);
    expect(shouldDrawBar("exhausted")).toBe(true);
    expect(shouldDrawBar("waived")).toBe(true);
    expect(shouldDrawBar("unattributed")).toBe(false);
    expect(shouldDrawBar("pricing_incomplete")).toBe(false);
    expect(spendPercent(25 * TICKS_PER_USD, 50 * TICKS_PER_USD)).toBe(50);
  });

  it("limits waiver writes to owner or admin on project and initiative", () => {
    expect(isWaiverWriteRole("owner")).toBe(true);
    expect(isWaiverWriteRole("admin")).toBe(true);
    expect(isWaiverWriteRole("member")).toBe(false);
    expect(isWaiverWriteRole(null)).toBe(false);
    expect(canWaiveScope("project")).toBe(true);
    expect(canWaiveScope("initiative")).toBe(true);
    expect(canWaiveScope("agent")).toBe(false);
    expect(canWaiveScope("squad")).toBe(false);
  });

  it("treats a waiver as live on the half-open window", () => {
    const waiver = {
      starts_at: "2026-08-10T00:00:00.000Z",
      ends_at: "2026-09-01T00:00:00.000Z",
    };
    expect(isWaiverLive(waiver, new Date("2026-08-10T00:00:00.000Z"))).toBe(true);
    expect(isWaiverLive(waiver, new Date("2026-08-28T12:00:00.000Z"))).toBe(true);
    expect(isWaiverLive(waiver, new Date("2026-09-01T00:00:00.000Z"))).toBe(false);
    expect(isWaiverLive(waiver, new Date("2026-08-09T23:59:59.000Z"))).toBe(false);
  });

  it("parses known account states and leaves unknown ones null", () => {
    expect(parseBudgetAccountState("exhausted")).toBe("exhausted");
    expect(parseBudgetAccountState("mystery")).toBeNull();
  });

  it("drops archived agents, system agents, and owners that already have a budget", () => {
    const owners = selectableOwners({
      scope: "agent",
      takenOwnerIds: new Set(["taken"]),
      projects: [],
      initiatives: [],
      agents: [
        { id: "taken", name: "Taken", archived_at: null },
        { id: "live", name: "Live", archived_at: null },
        { id: "archived", name: "Old", archived_at: "2026-01-01T00:00:00.000Z" },
        { id: "mika", name: "Mika", archived_at: null, system_key: "mika" },
      ],
      squads: [],
    });
    expect(owners).toEqual([{ id: "live", label: "Live" }]);
    expect(
      isSelectableAgent({ archived_at: null, system_key: "mika" }),
    ).toBe(false);
  });
});
