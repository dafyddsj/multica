/**
 * Initiatives realtime — listing-level. Mounted globally for the workspace
 * session so the More → Initiatives list stays fresh off-screen.
 *
 * Event coverage:
 *   - initiative:created / updated / deleted → invalidate list + detail
 *   - initiative:deleted → also invalidate projects (children detach)
 *   - project:created / updated / deleted → invalidate initiatives
 *     (project_count / issue_count / done_count live on the initiative)
 *   - reconnect → invalidate initiative list
 *
 * Initiative events are id-only on delete and rare enough that refetch
 * beats guessing list membership. Project events change counts we cannot
 * patch locally without the full initiative payload.
 */
import { useQueryClient } from "@tanstack/react-query";
import { initiativeKeys } from "@/data/queries/initiatives";
import { useWSSubscriptions } from "@/lib/use-ws-subscriptions";
import {
  invalidateInitiativeCaches,
  invalidateProjectsAfterInitiativeDeleted,
} from "./initiative-ws-updaters";

export function useInitiativesRealtime() {
  const qc = useQueryClient();

  useWSSubscriptions(
    (ws, wsId) => {
      const invalidateInitiatives = () => invalidateInitiativeCaches(qc, wsId);
      const invalidateList = () =>
        qc.invalidateQueries({ queryKey: initiativeKeys.list(wsId) });

      return [
        ws.on("initiative:created", invalidateInitiatives),
        ws.on("initiative:updated", invalidateInitiatives),
        ws.on("initiative:deleted", () => {
          invalidateInitiativeCaches(qc, wsId);
          invalidateProjectsAfterInitiativeDeleted(qc, wsId);
        }),
        ws.on("project:created", invalidateInitiatives),
        ws.on("project:updated", invalidateInitiatives),
        ws.on("project:deleted", invalidateInitiatives),
        ws.onReconnect(invalidateList),
      ];
    },
    [qc],
  );
}
