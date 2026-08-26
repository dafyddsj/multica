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
