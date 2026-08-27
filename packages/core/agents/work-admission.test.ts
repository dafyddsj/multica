// @vitest-environment node
import { describe, expect, it } from "vitest";
import {
  agentAcceptsNewWork,
  agentIsPaused,
  agentLifecycle,
} from "./work-admission";

describe("agentLifecycle", () => {
  it("is active when both timestamps are empty", () => {
    expect(agentLifecycle({ archived_at: null, paused_at: null })).toBe("active");
    expect(agentLifecycle({})).toBe("active");
  });

  it("is paused when only paused_at is set", () => {
    expect(
      agentLifecycle({ archived_at: null, paused_at: "2026-08-26T00:00:00Z" }),
    ).toBe("paused");
  });

  it("is archived when archived_at is set", () => {
    expect(
      agentLifecycle({ archived_at: "2026-08-26T00:00:00Z", paused_at: null }),
    ).toBe("archived");
  });

  it("lets archived win when both timestamps are set", () => {
    expect(
      agentLifecycle({
        archived_at: "2026-08-26T00:00:00Z",
        paused_at: "2026-08-25T00:00:00Z",
      }),
    ).toBe("archived");
  });
});

describe("agentAcceptsNewWork", () => {
  it("is true only for an active agent", () => {
    expect(agentAcceptsNewWork({ archived_at: null, paused_at: null })).toBe(true);
    expect(
      agentAcceptsNewWork({ archived_at: null, paused_at: "2026-08-26T00:00:00Z" }),
    ).toBe(false);
    expect(
      agentAcceptsNewWork({ archived_at: "2026-08-26T00:00:00Z", paused_at: null }),
    ).toBe(false);
  });
});

describe("agentIsPaused", () => {
  it("is true only for the paused lifecycle", () => {
    expect(agentIsPaused({ paused_at: "2026-08-26T00:00:00Z" })).toBe(true);
    expect(agentIsPaused({ archived_at: "2026-08-26T00:00:00Z", paused_at: "2026-08-26T00:00:00Z" })).toBe(false);
    expect(agentIsPaused({})).toBe(false);
  });
});
