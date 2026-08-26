import { useLocalSearchParams, router } from "expo-router";
import { useQuery } from "@tanstack/react-query";
import { ProjectStatusPickerBody } from "@/components/project/pickers/project-status-picker-body";
import { initiativeDetailOptions } from "@/data/queries/initiatives";
import { useUpdateInitiative } from "@/data/mutations/initiatives";
import { useWorkspaceStore } from "@/data/workspace-store";

export default function InitiativeStatusPickerRoute() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const wsId = useWorkspaceStore((s) => s.currentWorkspaceId);
  const { data: initiative } = useQuery(initiativeDetailOptions(wsId, id));
  const updateInitiative = useUpdateInitiative(id);

  return (
    <ProjectStatusPickerBody
      value={initiative?.status ?? "planned"}
      onChange={(next) => {
        updateInitiative.mutate({ status: next });
        router.back();
      }}
    />
  );
}
