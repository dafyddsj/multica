export type MemoryScope =
  | "workspace"
  | "initiative"
  | "project"
  | "issue"
  | "squad"
  | "agent"
  | "user";

export type MemoryKind = "fact" | "preference" | "procedure" | "observation";

export interface MemoryEntry {
  id: string;
  workspace_id: string | null;
  scope: MemoryScope;
  owner_id: string;
  body: string;
  kind: MemoryKind;
  provenance: Record<string, unknown>;
  created_by_type?: string | null;
  created_by_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface MemoryListResponse {
  entries: MemoryEntry[];
  total: number;
}

export interface MemoryHit {
  id: string;
  scope: MemoryScope | string;
  owner_id: string;
  body: string;
  kind: MemoryKind | string;
}

export interface MemoryRecallResponse {
  hits: MemoryHit[];
}

export interface CreateMemoryRequest {
  scope: MemoryScope;
  owner_id: string;
  body: string;
  kind?: MemoryKind;
  provenance?: Record<string, unknown>;
}

export interface UpdateMemoryRequest {
  body?: string;
  kind?: MemoryKind;
  provenance?: Record<string, unknown>;
}

export const MEMORY_ENABLED_SETTING = "memory_enabled";

export function isWorkspaceMemoryEnabled(
  settings: Record<string, unknown> | undefined,
): boolean {
  return settings?.memory_enabled === true;
}
