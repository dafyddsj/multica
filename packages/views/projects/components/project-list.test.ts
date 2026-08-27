// @vitest-environment node
import { describe, expect, it } from "vitest";
import type { Project } from "@multica/core/types";
import { EMPTY_PROJECT_FILTERS } from "@multica/core/projects";
import {
  NO_INITIATIVE_FILTER,
  compareProjects,
  countActiveFilters,
  groupProjectsByInitiative,
  matchesProjectSearch,
  projectPassesFilters,
  sortProjectGroups,
} from "./project-list";

function makeProject(overrides: Partial<Project> = {}): Project {
  return {
    id: "p1",
    workspace_id: "ws",
    title: "Alpha",
    description: null,
    icon: null,
    status: "planned",
    priority: "none",
    lead_type: null,
    lead_id: null,
    start_date: null,
    due_date: null,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    issue_count: 0,
    done_count: 0,
    resource_count: 0,
    initiative_id: null,
    ...overrides,
  };
}

const titles: Record<string, string> = {
  "init-a": "Atlas",
  "init-b": "Beacon",
};

const titleOf = (id: string) => titles[id];

describe("projectPassesFilters", () => {
  it("treats an empty initiatives list as inactive", () => {
    const project = makeProject({ initiative_id: "init-a" });
    expect(projectPassesFilters(project, EMPTY_PROJECT_FILTERS)).toBe(true);
  });

  it("keeps a project whose initiative is selected", () => {
    const project = makeProject({ initiative_id: "init-a" });
    expect(
      projectPassesFilters(project, { ...EMPTY_PROJECT_FILTERS, initiatives: ["init-a"] }),
    ).toBe(true);
  });

  it("drops a project whose initiative is not selected", () => {
    const project = makeProject({ initiative_id: "init-a" });
    expect(
      projectPassesFilters(project, { ...EMPTY_PROJECT_FILTERS, initiatives: ["init-b"] }),
    ).toBe(false);
  });

  it("matches unassigned projects with the none sentinel", () => {
    const project = makeProject({ initiative_id: null });
    expect(
      projectPassesFilters(project, {
        ...EMPTY_PROJECT_FILTERS,
        initiatives: [NO_INITIATIVE_FILTER],
      }),
    ).toBe(true);
    expect(
      projectPassesFilters(project, { ...EMPTY_PROJECT_FILTERS, initiatives: ["init-a"] }),
    ).toBe(false);
  });
});

describe("countActiveFilters", () => {
  it("counts the initiative dimension", () => {
    expect(countActiveFilters(EMPTY_PROJECT_FILTERS)).toBe(0);
    expect(
      countActiveFilters({ ...EMPTY_PROJECT_FILTERS, initiatives: ["init-a"] }),
    ).toBe(1);
  });
});

describe("matchesProjectSearch", () => {
  it("matches title and pinyin", () => {
    const project = makeProject({ title: "Launch" });
    expect(matchesProjectSearch(project, "lau", () => false)).toBe(true);
    expect(matchesProjectSearch(project, "zzz", () => true)).toBe(true);
    expect(matchesProjectSearch(project, "zzz", () => false)).toBe(false);
  });
});

describe("projectPassesFilters", () => {
  it("ANDs initiative with other dimensions", () => {
    const match = makeProject({ initiative_id: "init-a", status: "planned" });
    const otherStatus = makeProject({
      id: "p2",
      initiative_id: "init-a",
      status: "completed",
    });
    const filters = { ...EMPTY_PROJECT_FILTERS, initiatives: ["init-a"], statuses: ["planned"] };
    expect(projectPassesFilters(match, filters)).toBe(true);
    expect(projectPassesFilters(otherStatus, filters)).toBe(false);
  });
});

describe("compareProjects", () => {
  it("sorts by initiative title, then name", () => {
    const a = makeProject({ id: "1", title: "Zulu", initiative_id: "init-a" });
    const b = makeProject({ id: "2", title: "Alpha", initiative_id: "init-b" });
    const c = makeProject({ id: "3", title: "Mid", initiative_id: "init-a" });
    const sorted = [a, b, c].sort((left, right) =>
      compareProjects(left, right, "initiative", "asc", titleOf),
    );
    expect(sorted.map((p) => p.id)).toEqual(["3", "1", "2"]);
  });

  it("reverses named titles on desc and keeps unassigned last", () => {
    const none = makeProject({ id: "n", title: "Loose", initiative_id: null });
    const a = makeProject({ id: "1", title: "Zulu", initiative_id: "init-a" });
    const b = makeProject({ id: "2", title: "Alpha", initiative_id: "init-b" });
    const sorted = [none, a, b].sort((left, right) =>
      compareProjects(left, right, "initiative", "desc", titleOf),
    );
    expect(sorted.map((p) => p.id)).toEqual(["2", "1", "n"]);
  });

  it("sorts unresolved ids after named titles and before unassigned", () => {
    const named = makeProject({ id: "1", title: "Zulu", initiative_id: "init-a" });
    const unresolved = makeProject({ id: "u", title: "Ghost", initiative_id: "missing" });
    const none = makeProject({ id: "n", title: "Loose", initiative_id: null });
    const sorted = [none, unresolved, named].sort((left, right) =>
      compareProjects(left, right, "initiative", "asc", titleOf),
    );
    expect(sorted.map((p) => p.id)).toEqual(["1", "u", "n"]);
  });
});

describe("groupProjectsByInitiative", () => {
  it("sorts named groups before none, and honors desc", () => {
    const none = makeProject({ id: "n", title: "Loose", initiative_id: null });
    const a = makeProject({ id: "a", title: "One", initiative_id: "init-b" });
    const b = makeProject({ id: "b", title: "Two", initiative_id: "init-a" });
    const grouped = sortProjectGroups(
      groupProjectsByInitiative([none, a, b]),
      titleOf,
      "asc",
    );
    expect(grouped.map((g) => g.key)).toEqual(["init-a", "init-b", NO_INITIATIVE_FILTER]);
    expect(grouped[0]?.projects.map((p) => p.id)).toEqual(["b"]);

    const desc = sortProjectGroups(groupProjectsByInitiative([none, a, b]), titleOf, "desc");
    expect(desc.map((g) => g.key)).toEqual(["init-b", "init-a", NO_INITIATIVE_FILTER]);
  });
});
