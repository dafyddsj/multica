/**
 * Draft state for the New Initiative modal.
 *
 * Same rationale as `new-project-draft-store.ts`: formSheet picker routes
 * have no parent-child relationship with the create modal, so status and
 * priority cross screens through this store.
 */
import { useEffect, useRef } from "react";
import { create } from "zustand";
import type {
  InitiativePriority,
  InitiativeStatus,
} from "@multica/core/types";

interface NewInitiativeDraftState {
  status: InitiativeStatus;
  priority: InitiativePriority;
  setStatus: (next: InitiativeStatus) => void;
  setPriority: (next: InitiativePriority) => void;
  reset: () => void;
}

const INITIAL: Pick<NewInitiativeDraftState, "status" | "priority"> = {
  status: "planned",
  priority: "none",
};

export const useNewInitiativeDraftStore = create<NewInitiativeDraftState>(
  (set) => ({
    ...INITIAL,
    setStatus: (next) => set({ status: next }),
    setPriority: (next) => set({ priority: next }),
    reset: () => set({ ...INITIAL }),
  }),
);

export function useNewInitiativeDraftResetOnWorkspaceChange(
  wsId: string | null,
) {
  const prevRef = useRef(wsId);
  useEffect(() => {
    if (prevRef.current !== wsId) {
      useNewInitiativeDraftStore.getState().reset();
      prevRef.current = wsId;
    }
  }, [wsId]);
}
