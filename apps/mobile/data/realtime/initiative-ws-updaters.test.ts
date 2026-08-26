// @vitest-environment node
import { QueryClient } from "@tanstack/react-query";
import type { Initiative, Project } from "@multica/core/types";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/data/api", () => ({ api: {} }));

import { initiativeKeys, projectsForInitiative } from "@/data/queries/initiatives";
import { projectKeys } from "@/data/queries/projects";
import {
  dropInitiativeCaches,
  invalidateInitiativeCaches,
  invalidateProjectsAfterInitiativeDeleted,
} from "./initiative-ws-updaters";

describe("initiative WS updaters", () => {
  const wsId = "workspace-1";
  const initiativeId = "init-1";

  it("invalidates the whole initiative key tree", () => {
    const qc = new QueryClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");
    invalidateInitiativeCaches(qc, wsId);
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: initiativeKeys.all(wsId),
    });
  });

  it("refreshes projects after an initiative is deleted", () => {
    const qc = new QueryClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");
    invalidateProjectsAfterInitiativeDeleted(qc, wsId);
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: projectKeys.all(wsId),
    });
  });

  it("drops the deleted initiative from list and detail caches", () => {
    const qc = new QueryClient();
    qc.setQueryData<Initiative[]>(initiativeKeys.list(wsId), [
      { id: initiativeId, title: "Keep me gone" } as Initiative,
      { id: "init-2", title: "Stay" } as Initiative,
    ]);
    qc.setQueryData<Initiative>(initiativeKeys.detail(wsId, initiativeId), {
      id: initiativeId,
      title: "Keep me gone",
    } as Initiative);

    dropInitiativeCaches(qc, wsId, initiativeId);

    expect(qc.getQueryData<Initiative[]>(initiativeKeys.list(wsId))).toEqual([
      { id: "init-2", title: "Stay" },
    ]);
    expect(
      qc.getQueryData<Initiative>(initiativeKeys.detail(wsId, initiativeId)),
    ).toBeUndefined();
  });

  it("does not treat a project as deleted when an initiative is dropped", () => {
    const qc = new QueryClient();
    qc.setQueryData<Project[]>(projectKeys.list(wsId), [
      { id: "proj-1", title: "Child", initiative_id: initiativeId } as Project,
    ]);
    dropInitiativeCaches(qc, wsId, initiativeId);
    expect(qc.getQueryData<Project[]>(projectKeys.list(wsId))).toEqual([
      { id: "proj-1", title: "Child", initiative_id: initiativeId },
    ]);
  });
});

describe("projectsForInitiative", () => {
  it("keeps only projects whose initiative_id matches", () => {
    const projects = [
      { id: "a", initiative_id: "init-1" },
      { id: "b", initiative_id: null },
      { id: "c", initiative_id: "init-2" },
      { id: "d", initiative_id: "init-1" },
    ] as Project[];
    expect(projectsForInitiative(projects, "init-1").map((p) => p.id)).toEqual([
      "a",
      "d",
    ]);
  });
});
