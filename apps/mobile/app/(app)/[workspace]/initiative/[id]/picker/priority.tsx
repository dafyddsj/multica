import { useLocalSearchParams, router } from "expo-router";
import { useQuery } from "@tanstack/react-query";
import { ProjectPriorityPickerBody } from "@/components/project/pickers/project-priority-picker-body";
import { initiativeDetailOptions } from "@/data/queries/initiatives";
import { useUpdateInitiative } from "@/data/mutations/initiatives";
import { useWorkspaceStore } from "@/data/workspace-store";

export default function InitiativePriorityPickerRoute() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const wsId = useWorkspaceStore((s) => s.currentWorkspaceId);
  const { data: initiative } = useQuery(initiativeDetailOptions(wsId, id));
  const updateInitiative = useUpdateInitiative(id);

  return (
    <ProjectPriorityPickerBody
      value={initiative?.priority ?? "none"}
      onChange={(next) => {
        updateInitiative.mutate({ priority: next });
        router.back();
      }}
    />
  );
}
