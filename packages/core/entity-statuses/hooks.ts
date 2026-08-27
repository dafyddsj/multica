import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import type { EntityStatusResourceType } from "../types";
import {
  buildEntityStatusCatalog,
  entityStatusListOptions,
  type EntityStatusCatalog,
} from "./queries";

export function useEntityStatuses(
  wsId: string,
  resourceType: EntityStatusResourceType,
): EntityStatusCatalog {
  const { data, isPending, isError, refetch } = useQuery({
    ...entityStatusListOptions(wsId, resourceType),
    enabled: Boolean(wsId),
  });
  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);
  return useMemo(
    () => buildEntityStatusCatalog(data, resourceType, { isPending, isError, retry }),
    [data, isError, isPending, resourceType, retry],
  );
}
