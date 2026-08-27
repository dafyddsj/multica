// @vitest-environment node

import { describe, expect, it } from "vitest";
import {
  ALL_INITIATIVES,
  ALL_PROJECTS,
  projectValueForInitiativeChange,
  projectsForInitiative,
  resolveDashboardInitiativeId,
  resolveDashboardProjectId,
} from "./dashboard-scope";

const initiatives = [{ id: "init-1" }, { id: "init-2" }];
const projects = [
  { id: "proj-1", initiative_id: "init-1" },
  { id: "proj-2", initiative_id: "init-2" },
  { id: "proj-3", initiative_id: null },
];

describe("resolveDashboardInitiativeId", () => {
  it("treats the all-initiatives sentinel and unknown ids as no filter", () => {
    expect(resolveDashboardInitiativeId(ALL_INITIATIVES, initiatives)).toBeNull();
    expect(resolveDashboardInitiativeId("missing", initiatives)).toBeNull();
    expect(resolveDashboardInitiativeId("init-1", initiatives)).toBe("init-1");
  });
});

describe("projectsForInitiative / resolveDashboardProjectId", () => {
  it("narrows projects to the selected initiative and drops a leftover project", () => {
    const scoped = projectsForInitiative(projects, "init-1");
    expect(scoped.map((project) => project.id)).toEqual(["proj-1"]);
    expect(resolveDashboardProjectId("proj-2", scoped)).toBeNull();
    expect(resolveDashboardProjectId("proj-1", scoped)).toBe("proj-1");
    expect(resolveDashboardProjectId(ALL_PROJECTS, scoped)).toBeNull();
  });

  it("keeps the full list when no initiative is selected", () => {
    expect(projectsForInitiative(projects, null)).toEqual(projects);
    expect(resolveDashboardProjectId("proj-3", projects)).toBe("proj-3");
  });
});

describe("projectValueForInitiativeChange", () => {
  it("clears a project that the next initiative does not contain", () => {
    expect(projectValueForInitiativeChange("proj-1", null, "init-2", projects)).toBe(ALL_PROJECTS);
  });

  it("clears a project that the previous initiative had hidden", () => {
    expect(projectValueForInitiativeChange("proj-1", "init-2", null, projects)).toBe(ALL_PROJECTS);
  });

  it("keeps a project that belongs to the newly selected initiative", () => {
    expect(projectValueForInitiativeChange("proj-1", null, "init-1", projects)).toBe("proj-1");
  });

  it("keeps a visible project when broadening to all initiatives", () => {
    expect(projectValueForInitiativeChange("proj-1", "init-1", null, projects)).toBe("proj-1");
  });
});
