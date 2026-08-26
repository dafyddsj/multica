/**
 * Initiative picker for an existing project. Reads the project from cache,
 * writes `initiative_id` via useUpdateProject.
 */
import { useMemo } from "react";
import { useLocalSearchParams, router } from "expo-router";
import { useQuery } from "@tanstack/react-query";
import { InitiativePickerBody } from "@/components/initiative/pickers/initiative-picker-body";
import {
  findInitiative,
  initiativeListOptions,
} from "@/data/queries/initiatives";
import { projectDetailOptions } from "@/data/queries/projects";
import { useUpdateProject } from "@/data/mutations/projects";
import { useWorkspaceStore } from "@/data/workspace-store";
import { useNativeSearchBar } from "@/lib/use-native-search-bar";

export default function ProjectInitiativePickerRoute() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const wsId = useWorkspaceStore((s) => s.currentWorkspaceId);
  const { data: project } = useQuery(projectDetailOptions(wsId, id));
  const { data: initiatives = [] } = useQuery(initiativeListOptions(wsId));
  const updateProject = useUpdateProject(id);
  const query = useNativeSearchBar("Search initiatives", { autoFocus: true });

  const value = useMemo(
    () => findInitiative(initiatives, project?.initiative_id ?? null) ?? null,
    [initiatives, project?.initiative_id],
  );

  return (
    <InitiativePickerBody
      value={value}
      query={query}
      onChange={(next) => {
        updateProject.mutate({ initiative_id: next?.id ?? null });
        router.back();
      }}
    />
  );
}
