// @vitest-environment node
import { describe, expect, it } from "vitest";
import { isWorkspaceMemoryEnabled } from "../types/memory";

describe("isWorkspaceMemoryEnabled", () => {
  it("defaults off", () => {
    expect(isWorkspaceMemoryEnabled(undefined)).toBe(false);
    expect(isWorkspaceMemoryEnabled({})).toBe(false);
    expect(isWorkspaceMemoryEnabled({ memory_enabled: false })).toBe(false);
    expect(isWorkspaceMemoryEnabled({ memory_enabled: "true" })).toBe(false);
  });

  it("requires an explicit true boolean", () => {
    expect(isWorkspaceMemoryEnabled({ memory_enabled: true })).toBe(true);
  });
});
