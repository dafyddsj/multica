export type BudgetScope = "agent" | "squad" | "project" | "initiative";
export type BudgetOverLimit = "pause" | "allow";
export type BudgetAccountState =
  | "ok"
  | "softened"
  | "exhausted"
  | "pricing_incomplete"
  | "unattributed"
  | "waived";
export type WaiverScope = "project" | "initiative";

export interface BudgetPeriod {
  period_start: string;
  period_end: string;
  spent_usd_ticks: number;
  unpriced_line_count: number;
  state: string;
}

export interface Budget {
  id: string;
  scope: BudgetScope;
  owner_id: string;
  limit_usd_ticks: number;
  soften_at_percent: number | null;
  over_limit: BudgetOverLimit;
  current_period: BudgetPeriod | null;
}

export interface BudgetWaiver {
  id: string;
  scope: WaiverScope;
  owner_id: string;
  starts_at: string;
  ends_at: string;
  created_by: string;
  reason: string | null;
}

export interface ListBudgetsResponse {
  budgets: Budget[];
}

export interface ListBudgetWaiversResponse {
  waivers: BudgetWaiver[];
}

export interface CreateBudgetRequest {
  scope: BudgetScope;
  owner_id: string;
  limit_usd_ticks: number;
  soften_at_percent?: number | null;
  over_limit: BudgetOverLimit;
}

export interface UpdateBudgetRequest {
  limit_usd_ticks?: number;
  soften_at_percent?: number | null;
  over_limit?: BudgetOverLimit;
}

export interface CreateBudgetWaiverRequest {
  scope: WaiverScope;
  owner_id: string;
  starts_at?: string;
  ends_at?: string;
  reason?: string | null;
}
