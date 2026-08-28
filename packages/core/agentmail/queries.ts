import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const agentmailKeys = {
  all: (wsId: string) => ["agentmail", wsId] as const,
  workspace: (wsId: string) => [...agentmailKeys.all(wsId), "workspace"] as const,
  inbox: (wsId: string, agentId: string) =>
    [...agentmailKeys.all(wsId), "inbox", agentId] as const,
};

export const agentmailWorkspaceOptions = (wsId: string) =>
  queryOptions({
    queryKey: agentmailKeys.workspace(wsId),
    queryFn: () => api.getAgentMail(wsId),
    enabled: !!wsId,
  });

export const agentmailInboxOptions = (wsId: string, agentId: string) =>
  queryOptions({
    queryKey: agentmailKeys.inbox(wsId, agentId),
    queryFn: () => api.getAgentMailInbox(agentId),
    enabled: !!wsId && !!agentId,
  });
