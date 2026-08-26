/**
 * Row model for the workspace search screen's single FlatList.
 *
 * Extracted from app/(app)/[workspace]/search.tsx so the ordering rules — in
 * particular the cross-type cancelled demotion (MUL-5824) — are unit-testable
 * without mounting the screen.
 */
import type {
  Issue,
  SearchIssueResult,
  SearchProjectResult,
  SearchInitiativeResult,
} from "@multica/core/types";
import {
  isProjectDirectHit,
  partitionAggregatedSearchResults,
  partitionStable,
} from "@multica/core/search/cancelled-rank";

export type RowItem =
  | { kind: "header"; key: string; title: string }
  | { kind: "issue"; key: string; issue: SearchIssueResult; query: string }
  | { kind: "project"; key: string; project: SearchProjectResult; query: string }
  | { kind: "initiative"; key: string; initiative: SearchInitiativeResult; query: string }
  | { kind: "recent"; key: string; issue: Issue };

/**
 * Builds the flat row list. Empty query → the Recent section; otherwise the
 * search results in cancelled-partition order:
 *
 *   Initiatives (live) → Projects (live) → Issues (live) → Cancelled
 *
 * The three searches are ranked independently, so a cancelled row from one
 * response would outrank a live row from another unless the partition happens
 * at this aggregation point. Direct hits stay in their live section.
 */
export function buildSearchRows({
  query,
  issues,
  projects,
  initiatives = [],
  recentIssues,
}: {
  query: string;
  issues: SearchIssueResult[];
  projects: SearchProjectResult[];
  initiatives?: SearchInitiativeResult[];
  recentIssues: Issue[];
}): RowItem[] {
  const trimmedQuery = query.trim();

  if (!trimmedQuery) {
    if (recentIssues.length === 0) return [];
    return [
      { kind: "header", key: "h-recent", title: "Recent" },
      ...recentIssues.map<RowItem>((issue) => ({
        kind: "recent",
        key: `r-${issue.id}`,
        issue,
      })),
    ];
  }

  const parts = partitionAggregatedSearchResults({
    issues,
    projects,
    query: trimmedQuery,
  });
  const initiativeParts = partitionStable(
    initiatives,
    (initiative) =>
      initiative.status === "cancelled" && !isProjectDirectHit(initiative, trimmedQuery),
  );

  const rows: RowItem[] = [];
  if (initiativeParts.live.length > 0) {
    rows.push({ kind: "header", key: "h-initiatives", title: "Initiatives" });
    for (const initiative of initiativeParts.live) {
      rows.push({
        kind: "initiative",
        key: `n-${initiative.id}`,
        initiative,
        query: trimmedQuery,
      });
    }
  }
  if (parts.liveProjects.length > 0) {
    rows.push({ kind: "header", key: "h-projects", title: "Projects" });
    for (const project of parts.liveProjects) {
      rows.push({ kind: "project", key: `p-${project.id}`, project, query: trimmedQuery });
    }
  }
  if (parts.liveIssues.length > 0) {
    rows.push({ kind: "header", key: "h-issues", title: "Issues" });
    for (const issue of parts.liveIssues) {
      rows.push({ kind: "issue", key: `i-${issue.id}`, issue, query: trimmedQuery });
    }
  }
  const hasCancelled =
    parts.hasCancelled || initiativeParts.cancelled.length > 0;
  if (hasCancelled) {
    rows.push({ kind: "header", key: "h-cancelled", title: "Cancelled" });
    for (const initiative of initiativeParts.cancelled) {
      rows.push({
        kind: "initiative",
        key: `n-${initiative.id}`,
        initiative,
        query: trimmedQuery,
      });
    }
    for (const project of parts.cancelledProjects) {
      rows.push({ kind: "project", key: `p-${project.id}`, project, query: trimmedQuery });
    }
    for (const issue of parts.cancelledIssues) {
      rows.push({ kind: "issue", key: `i-${issue.id}`, issue, query: trimmedQuery });
    }
  }
  return rows;
}
