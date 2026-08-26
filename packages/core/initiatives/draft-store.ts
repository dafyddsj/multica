import type { InitiativeStatus, InitiativePriority } from "../types";
import { createDraftStore } from "../drafts/create-draft-store";

interface InitiativeDraft {
  title: string;
  description: string;
  status: InitiativeStatus;
  priority: InitiativePriority;
  leadType?: "member" | "agent";
  leadId?: string;
  icon?: string;
  startDate?: string;
  dueDate?: string;
}

const EMPTY_DRAFT: InitiativeDraft = {
  title: "",
  description: "",
  status: "planned",
  priority: "none",
  leadType: undefined,
  leadId: undefined,
  icon: undefined,
  startDate: undefined,
  dueDate: undefined,
};

export const useInitiativeDraftStore = createDraftStore<InitiativeDraft>({
  storageKey: "multica_initiative_draft",
  emptyData: EMPTY_DRAFT,
  hasMeaningful: (d) => !!(d.title || d.description),
});
