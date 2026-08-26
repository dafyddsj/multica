import { useLocalSearchParams, router } from "expo-router";
import { useQuery } from "@tanstack/react-query";
import { ProjectLeadPickerBody } from "@/components/project/pickers/project-lead-picker-body";
import { initiativeDetailOptions } from "@/data/queries/initiatives";
import { useUpdateInitiative } from "@/data/mutations/initiatives";
import { useWorkspaceStore } from "@/data/workspace-store";
import { useNativeSearchBar } from "@/lib/use-native-search-bar";

export default function InitiativeLeadPickerRoute() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const wsId = useWorkspaceStore((s) => s.currentWorkspaceId);
  const { data: initiative } = useQuery(initiativeDetailOptions(wsId, id));
  const updateInitiative = useUpdateInitiative(id);
  const query = useNativeSearchBar("Search members or agents", {
    autoFocus: true,
  });

  const value =
    initiative?.lead_type && initiative?.lead_id
      ? { type: initiative.lead_type, id: initiative.lead_id }
      : null;

  return (
    <ProjectLeadPickerBody
      value={value}
      query={query}
      onChange={(next) => {
        if (next === null) {
          updateInitiative.mutate({ lead_type: null, lead_id: null });
        } else {
          updateInitiative.mutate({ lead_type: next.type, lead_id: next.id });
        }
        router.back();
      }}
    />
  );
}
