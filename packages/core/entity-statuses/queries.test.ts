// @vitest-environment node
import { describe, expect, it } from "vitest";
import type { EntityStatusEntry } from "../types";
import {
  buildEntityStatusCatalog,
  compareEntityStatusEntries,
  isClosedEntityStatus,
  isEntityStatusCategory,
} from "./queries";

function entry(overrides: Partial<EntityStatusEntry>): EntityStatusEntry {
  return {
    id: overrides.key ?? "id",
    workspace_id: "ws-1",
    resource_type: "project",
    key: "shipping",
    name: "Shipping",
    description: "",
    category: "in_progress",
    color: "#22c55e",
    is_system: false,
    position: 1,
    archived_at: null,
    created_at: "",
    updated_at: "",
    ...overrides,
  };
}

describe("entity status catalog", () => {
  it("resolves custom keys to their category and built-ins to themselves", () => {
    const catalog = buildEntityStatusCatalog(
      [
        entry({ key: "planned", name: "Not started", category: "planned", is_system: true, position: 0 }),
        entry({ key: "shipping", name: "Shipping", category: "in_progress" }),
      ],
      "project",
    );
    expect(catalog.categoryOf("shipping")).toBe("in_progress");
    expect(catalog.categoryOf("planned")).toBe("planned");
    expect(catalog.categoryOf("completed")).toBe("completed");
    expect(catalog.labelOf("shipping")).toBe("Shipping");
    expect(catalog.labelOf("planned")).toBe("Not started");
    expect(catalog.colorOf("shipping")).toBe("#22c55e");
    expect(catalog.colorOf("planned")).toBe("#22c55e");
  });

  it("treats completed and cancelled categories as closed", () => {
    const catalog = buildEntityStatusCatalog(
      [entry({ key: "wont", name: "Won't", category: "cancelled" })],
      "initiative",
    );
    expect(isClosedEntityStatus("wont", catalog)).toBe(true);
    expect(isClosedEntityStatus("completed")).toBe(true);
    expect(isClosedEntityStatus("planned")).toBe(false);
  });

  it("sorts by category rank then position", () => {
    const rows = [
      entry({ key: "b", category: "planned", position: 2 }),
      entry({ key: "a", category: "planned", position: 1 }),
      entry({ key: "c", category: "completed", position: 1 }),
    ];
    const sorted = [...rows].sort(compareEntityStatusEntries);
    expect(sorted.map((r) => r.key)).toEqual(["a", "b", "c"]);
  });

  it("recognizes the five categories", () => {
    expect(isEntityStatusCategory("paused")).toBe(true);
    expect(isEntityStatusCategory("todo")).toBe(false);
  });
});
