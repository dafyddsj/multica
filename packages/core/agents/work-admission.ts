// Admission predicate for agent lifecycle. Archive retires. Pause holds.
// Archived wins when both timestamps are set. Every picker, mention, chat
// start, and mobile filter that decides "may this agent take new work"
// goes through these functions so a later schema default is one edit.

export type AgentLifecycle = "active" | "paused" | "archived";

export type AgentLifecycleFields = {
  archived_at?: string | null;
  paused_at?: string | null;
};

export function agentLifecycle(agent: AgentLifecycleFields): AgentLifecycle {
  if (agent.archived_at) return "archived";
  if (agent.paused_at) return "paused";
  return "active";
}

export function agentAcceptsNewWork(agent: AgentLifecycleFields): boolean {
  return agentLifecycle(agent) === "active";
}

export function agentIsPaused(agent: AgentLifecycleFields): boolean {
  return agentLifecycle(agent) === "paused";
}
