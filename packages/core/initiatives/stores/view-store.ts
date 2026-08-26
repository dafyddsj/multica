"use client";

import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import { createWorkspaceAwareStorage, registerForWorkspaceRehydration } from "../../platform/workspace-storage";
import { defaultStorage } from "../../platform/storage";

export type InitiativeViewMode = "compact" | "comfortable";

export type InitiativeSortField = "name" | "priority" | "status" | "progress" | "created";

export type InitiativeSortDirection = "asc" | "desc";

export const INITIATIVE_SORT_DEFAULT_DIRECTION: Record<
  InitiativeSortField,
  InitiativeSortDirection
> = {
  name: "asc",
  priority: "desc",
  status: "asc",
  progress: "desc",
  created: "desc",
};

export interface InitiativeListFilters {
  statuses: string[];
  priorities: string[];
  leads: string[];
}

export const EMPTY_INITIATIVE_FILTERS: InitiativeListFilters = {
  statuses: [],
  priorities: [],
  leads: [],
};

export type InitiativeColumnKey = "priority" | "progress" | "lead" | "projects" | "created";

export const INITIATIVE_DEFAULT_HIDDEN_COLUMNS: InitiativeColumnKey[] = ["projects"];

export interface InitiativeViewState {
  viewMode: InitiativeViewMode;
  sortField: InitiativeSortField;
  sortDirection: InitiativeSortDirection;
  hiddenColumns: InitiativeColumnKey[];
  filters: InitiativeListFilters;
  setViewMode: (mode: InitiativeViewMode) => void;
  toggleSort: (field: InitiativeSortField) => void;
  setSortField: (field: InitiativeSortField) => void;
  setSortDirection: (direction: InitiativeSortDirection) => void;
  toggleColumn: (key: InitiativeColumnKey) => void;
  toggleFilter: (key: keyof InitiativeListFilters, value: string) => void;
  clearFilters: () => void;
}

const DEFAULTS = {
  viewMode: "compact" as InitiativeViewMode,
  sortField: "created" as InitiativeSortField,
  sortDirection: INITIATIVE_SORT_DEFAULT_DIRECTION.created,
  hiddenColumns: INITIATIVE_DEFAULT_HIDDEN_COLUMNS,
  filters: EMPTY_INITIATIVE_FILTERS,
};

export const useInitiativeViewStore = create<InitiativeViewState>()(
  persist(
    (set) => ({
      ...DEFAULTS,
      setViewMode: (mode) => set({ viewMode: mode }),
      toggleSort: (field) =>
        set((state) =>
          state.sortField === field
            ? { sortDirection: state.sortDirection === "asc" ? "desc" : "asc" }
            : {
                sortField: field,
                sortDirection: INITIATIVE_SORT_DEFAULT_DIRECTION[field],
              },
        ),
      setSortField: (field) =>
        set((state) =>
          state.sortField === field
            ? {}
            : {
                sortField: field,
                sortDirection: INITIATIVE_SORT_DEFAULT_DIRECTION[field],
              },
        ),
      setSortDirection: (direction) => set({ sortDirection: direction }),
      toggleColumn: (key) =>
        set((state) => ({
          hiddenColumns: state.hiddenColumns.includes(key)
            ? state.hiddenColumns.filter((k) => k !== key)
            : [...state.hiddenColumns, key],
        })),
      toggleFilter: (key, value) =>
        set((state) => {
          const list = state.filters[key] as string[];
          const next = list.includes(value)
            ? list.filter((v) => v !== value)
            : [...list, value];
          return { filters: { ...state.filters, [key]: next } };
        }),
      clearFilters: () => set({ filters: EMPTY_INITIATIVE_FILTERS }),
    }),
    {
      name: "multica_initiatives_view",
      storage: createJSONStorage(() => createWorkspaceAwareStorage(defaultStorage)),
      partialize: (state) => ({
        viewMode: state.viewMode,
        sortField: state.sortField,
        sortDirection: state.sortDirection,
        hiddenColumns: state.hiddenColumns,
        filters: state.filters,
      }),
      merge: (persisted, current) => {
        if (!persisted) return { ...current, ...DEFAULTS };
        const p = persisted as Partial<InitiativeViewState>;
        return {
          ...current,
          ...p,
          filters: { ...EMPTY_INITIATIVE_FILTERS, ...(p.filters ?? {}) },
        };
      },
    }
  )
);

registerForWorkspaceRehydration(() => useInitiativeViewStore.persist.rehydrate());
