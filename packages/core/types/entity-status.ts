/**
 * Workspace Initiative and Project status catalogs.
 *
 * The 5 categories map one-to-one onto the 5 built-in statuses: a category's
 * value IS its canonical status key. A custom status names a category and
 * inherits that meaning (active vs done, grouping) in full.
 */

export type EntityStatusResourceType = "initiative" | "project";

export type EntityStatusCategory =
  | "planned"
  | "in_progress"
  | "paused"
  | "completed"
  | "cancelled";

export interface EntityStatusEntry {
  id: string;
  workspace_id: string;
  resource_type: EntityStatusResourceType;
  /** Stable machine handle stored on project.status / initiative.status. */
  key: string;
  name: string;
  description: string;
  category: EntityStatusCategory;
  /** "#rrggbb". */
  color: string;
  is_system: boolean;
  position: number;
  archived_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ListEntityStatusesResponse {
  statuses: EntityStatusEntry[];
  resource_type: EntityStatusResourceType | "";
  categories: EntityStatusCategory[];
  total: number;
}

export interface CreateEntityStatusRequest {
  resource_type: EntityStatusResourceType;
  key?: string;
  name: string;
  description?: string;
  category: EntityStatusCategory;
  color: string;
}

export interface UpdateEntityStatusRequest {
  name?: string;
  description?: string;
  color?: string;
  position?: number;
}
