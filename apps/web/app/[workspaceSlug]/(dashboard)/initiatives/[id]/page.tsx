"use client";

import { use } from "react";
import { InitiativeDetail } from "@multica/views/initiatives/components";

export default function InitiativeDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  return <InitiativeDetail initiativeId={id} />;
}
