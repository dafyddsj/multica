/**
 * Per-initiative realtime. Mounted by the initiative detail screen;
 * cleans up on navigate-away.
 *
 *   - initiative:updated matching id → invalidate this detail
 *   - initiative:deleted matching id → drop caches, fire onDeleted
 *   - project:* → invalidate this detail (counts) so the header stays live
 *   - reconnect → invalidate this detail
 */
import { useQueryClient } from "@tanstack/react-query";
import { initiativeKeys } from "@/data/queries/initiatives";
import { useWSSubscriptions } from "@/lib/use-ws-subscriptions";
import { dropInitiativeCaches } from "./initiative-ws-updaters";

export function useInitiativeRealtime(
  initiativeId: string | undefined,
  onDeleted?: () => void,
) {
  const qc = useQueryClient();

  useWSSubscriptions(
    (ws, wsId) => {
      if (!initiativeId) return;

      const invalidateThis = () =>
        qc.invalidateQueries({
          queryKey: initiativeKeys.detail(wsId, initiativeId),
        });

      return [
        ws.on("initiative:updated", (payload) => {
          if (payload.initiative.id !== initiativeId) return;
          invalidateThis();
        }),
        ws.on("initiative:deleted", (payload) => {
          if (payload.initiative_id !== initiativeId) return;
          dropInitiativeCaches(qc, wsId, initiativeId);
          onDeleted?.();
        }),
        ws.on("project:created", invalidateThis),
        ws.on("project:updated", invalidateThis),
        ws.on("project:deleted", invalidateThis),
        ws.onReconnect(invalidateThis),
      ];
    },
    [initiativeId, qc, onDeleted],
  );
}
