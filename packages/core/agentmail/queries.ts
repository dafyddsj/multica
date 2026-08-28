import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const agentmailKeys = {
  all: (wsId: string) => ["agentmail", wsId] as const,
  workspace: (wsId: string) => [...agentmailKeys.all(wsId), "workspace"] as const,
  inbox: (wsId: string, agentId: string) =>
    [...agentmailKeys.all(wsId), "inbox", agentId] as const,
  threads: (wsId: string, agentId: string) =>
    [...agentmailKeys.all(wsId), "threads", agentId] as const,
  thread: (wsId: string, agentId: string, threadId: string) =>
    [...agentmailKeys.all(wsId), "thread", agentId, threadId] as const,
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

export const agentmailThreadsOptions = (wsId: string, agentId: string, enabled: boolean) =>
  queryOptions({
    queryKey: agentmailKeys.threads(wsId, agentId),
    queryFn: () => api.listAgentMailThreads(agentId),
    enabled: enabled && !!wsId && !!agentId,
  });

export const agentmailThreadOptions = (
  wsId: string,
  agentId: string,
  threadId: string,
) =>
  queryOptions({
    queryKey: agentmailKeys.thread(wsId, agentId, threadId),
    queryFn: () => api.getAgentMailThread(agentId, threadId),
    enabled: !!wsId && !!agentId && !!threadId,
  });
