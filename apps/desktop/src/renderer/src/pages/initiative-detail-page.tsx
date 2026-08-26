import { useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { InitiativeDetail } from "@multica/views/initiatives/components";
import { useWorkspaceId } from "@multica/core/hooks";
import { initiativeDetailOptions } from "@multica/core/initiatives/queries";
import { useDocumentTitle } from "@/hooks/use-document-title";

export function InitiativeDetailPage() {
  const { id } = useParams<{ id: string }>();
  const wsId = useWorkspaceId();
  const { data: initiative } = useQuery(initiativeDetailOptions(wsId, id!));

  useDocumentTitle(initiative ? initiative.title : "Initiative");

  if (!id) return null;
  return <InitiativeDetail initiativeId={id} />;
}
