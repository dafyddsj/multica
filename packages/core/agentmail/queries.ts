import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const agentmailKeys = {
  all: (wsId: string) => ["agentmail", wsId] as const,
  workspace: (wsId: string) => [...agentmailKeys.all(wsId), "workspace"] as const,
  domains: (wsId: string) => [...agentmailKeys.all(wsId), "domains"] as const,
  accountInboxes: (wsId: string) => [...agentmailKeys.all(wsId), "account-inboxes"] as const,
  inbox: (wsId: string, agentId: string) =>
    [...agentmailKeys.all(wsId), "inbox", agentId] as const,
  folders: (wsId: string, agentId: string) =>
    [...agentmailKeys.all(wsId), "folders", agentId] as const,
  mailbox: (wsId: string, agentId: string, section: string, label: string) =>
    [...agentmailKeys.all(wsId), "mailbox", agentId, section, label] as const,
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

export const agentmailDomainOptions = (wsId: string, enabled: boolean) =>
  queryOptions({
    queryKey: agentmailKeys.domains(wsId),
    queryFn: () => api.listAgentMailDomains(wsId),
    enabled: enabled && !!wsId,
  });

export const agentmailAccountInboxOptions = (wsId: string, enabled: boolean) =>
  queryOptions({
    queryKey: agentmailKeys.accountInboxes(wsId),
    queryFn: () => api.listAgentMailAccountInboxes(wsId),
    enabled: enabled && !!wsId,
  });

export const agentmailInboxOptions = (wsId: string, agentId: string) =>
  queryOptions({
    queryKey: agentmailKeys.inbox(wsId, agentId),
    queryFn: () => api.getAgentMailInbox(agentId),
    enabled: !!wsId && !!agentId,
  });

export const agentmailFoldersOptions = (wsId: string, agentId: string, enabled: boolean) =>
  queryOptions({
    queryKey: agentmailKeys.folders(wsId, agentId),
    queryFn: () => api.listAgentMailFolders(agentId),
    enabled: enabled && !!wsId && !!agentId,
  });

export const agentmailMailboxOptions = (
  wsId: string,
  agentId: string,
  section: string,
  label: string,
  enabled: boolean,
) =>
  queryOptions({
    queryKey: agentmailKeys.mailbox(wsId, agentId, section, label),
    queryFn: () => api.listAgentMailMailbox(agentId, { section, label: label || undefined }),
    enabled: enabled && !!wsId && !!agentId && !!section,
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
