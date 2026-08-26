import { router } from "expo-router";
import { ProjectPriorityPickerBody } from "@/components/project/pickers/project-priority-picker-body";
import { useNewInitiativeDraftStore } from "@/data/stores/new-initiative-draft-store";

export default function NewInitiativePriorityPickerRoute() {
  const priority = useNewInitiativeDraftStore((s) => s.priority);
  const setPriority = useNewInitiativeDraftStore((s) => s.setPriority);

  return (
    <ProjectPriorityPickerBody
      value={priority}
      onChange={(next) => {
        setPriority(next);
        router.back();
      }}
    />
  );
}
