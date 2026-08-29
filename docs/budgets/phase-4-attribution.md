# Phase 4. Attribution

Back to [overview.md](overview.md).

## Goal

Every task carries a stable coverage tuple so a ledger can post without asking today's project or squad membership.

## Changes

- Stamp `agent_task_queue` (names can match existing columns where they already exist) with `budget_project_id`, `budget_initiative_id`, and `budget_origin_squad_id` at enqueue.
- Project and initiative come from the issue at create. Chat with no issue leaves them null.
- Origin squad is the root squad invocation. Worker tasks inherit the leader's `origin_squad_id`. Membership alone does not set it.
- One helper on `TaskService` builds the tuple. Table-test every enqueue path (issue, mention, deferred, quick-create, chat, recovery, autopilot, squad leader, squad worker).
- Claim reconstructs coverage from those columns. It does not re-join `issue.project_id`.

## Data structures

Immutable `TaskContext` (`agent`, optional `project`, optional `initiative`, optional `origin_squad`). Written once. A conflicting second write is an error.

## Verification

**Static.** The enqueue matrix test. A project move after enqueue does not change the stamped ids. A worker task has the same `origin_squad_id` as its leader.

**Runtime.** Assign an issue to a squad, let a worker run, confirm the worker row has the squad id. Move the issue to another project after the task is queued and confirm the stamp is unchanged.
