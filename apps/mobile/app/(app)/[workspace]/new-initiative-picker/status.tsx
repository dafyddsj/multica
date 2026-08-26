import { router } from "expo-router";
import { ProjectStatusPickerBody } from "@/components/project/pickers/project-status-picker-body";
import { useNewInitiativeDraftStore } from "@/data/stores/new-initiative-draft-store";

export default function NewInitiativeStatusPickerRoute() {
  const status = useNewInitiativeDraftStore((s) => s.status);
  const setStatus = useNewInitiativeDraftStore((s) => s.setStatus);

  return (
    <ProjectStatusPickerBody
      value={status}
      onChange={(next) => {
        setStatus(next);
        router.back();
      }}
    />
  );
}
