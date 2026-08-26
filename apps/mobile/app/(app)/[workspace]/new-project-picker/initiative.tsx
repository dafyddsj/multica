import { router } from "expo-router";
import { InitiativePickerBody } from "@/components/initiative/pickers/initiative-picker-body";
import { useNewProjectDraftStore } from "@/data/stores/new-project-draft-store";
import { useNativeSearchBar } from "@/lib/use-native-search-bar";

export default function NewProjectInitiativePickerRoute() {
  const initiative = useNewProjectDraftStore((s) => s.initiative);
  const setInitiative = useNewProjectDraftStore((s) => s.setInitiative);
  const query = useNativeSearchBar("Search initiatives", { autoFocus: true });

  return (
    <InitiativePickerBody
      value={initiative}
      query={query}
      onChange={(next) => {
        setInitiative(next);
        router.back();
      }}
    />
  );
}
