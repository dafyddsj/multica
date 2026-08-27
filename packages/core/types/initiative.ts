export type InitiativeStatus =
  | "planned"
  | "in_progress"
  | "paused"
  | "completed"
  | "cancelled";

export type InitiativePriority = "urgent" | "high" | "medium" | "low" | "none";

export interface Initiative {
  id: string;
  workspace_id: string;
  title: string;
  description: string | null;
  icon: string | null;
  status: InitiativeStatus;
  priority: InitiativePriority;
  lead_type: "member" | "agent" | null;
  lead_id: string | null;
  start_date: string | null;
  due_date: string | null;
  issue_prefix: string | null;
  created_at: string;
  updated_at: string;
  project_count: number;
  issue_count: number;
  done_count: number;
}

export interface CreateInitiativeRequest {
  title: string;
  description?: string;
  icon?: string;
  status?: InitiativeStatus;
  priority?: InitiativePriority;
  lead_type?: "member" | "agent";
  lead_id?: string;
  start_date?: string;
  due_date?: string;
  issue_prefix?: string | null;
}

export interface UpdateInitiativeRequest {
  title?: string;
  description?: string | null;
  icon?: string | null;
  status?: InitiativeStatus;
  priority?: InitiativePriority;
  lead_type?: "member" | "agent" | null;
  lead_id?: string | null;
  start_date?: string | null;
  due_date?: string | null;
  issue_prefix?: string | null;
}

export interface ListInitiativesResponse {
  initiatives: Initiative[];
  total: number;
}

export interface SearchInitiativeResult extends Initiative {
  match_source: "title" | "description";
  matched_snippet?: string;
}

export interface SearchInitiativesResponse {
  initiatives: SearchInitiativeResult[];
  total: number;
}
