#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
fail=0

need_file() {
	if [ ! -f "$1" ]; then
		printf 'FAIL: missing fork file %s\n' "$1" >&2
		fail=1
	fi
}

need_absent() {
	if [ -e "$1" ]; then
		printf 'FAIL: upstream leftover should not exist at %s\n' "$1" >&2
		fail=1
	fi
}

need_grep() {
	local path="$1"
	local pat="$2"
	if ! grep -q -- "$pat" "$path"; then
		printf 'FAIL: %s no longer contains %s\n' "$path" "$pat" >&2
		fail=1
	fi
}

need_absent_grep() {
	local path="$1"
	local pat="$2"
	if [ -f "$path" ] && grep -q -- "$pat" "$path"; then
		printf 'FAIL: %s gained %s\n' "$path" "$pat" >&2
		fail=1
	fi
}

need_file server/migrations/432_initiatives.up.sql
need_file server/migrations/439_agent_co_authored_by_email.up.sql
need_file server/migrations/440_agent_paused.up.sql
need_file server/migrations/441_agent_conversation_starters_rename.up.sql
need_file server/migrations/442_agent_execution_lanes.up.sql
need_file server/migrations/443_memory_entry.up.sql
need_file server/migrations/454_user_clerk_user_id.up.sql
need_file server/migrations/456_workspace_clerk_org_id.up.sql
need_file server/migrations/459_runtime_profile_add_amp.up.sql
need_file server/migrations/460_agentmail_tables.up.sql
need_file server/migrations/463_runtime_profile_add_devin.up.sql
need_file server/migrations/464_runtime_profile_add_goose.up.sql
need_file server/migrations/468_budget.up.sql
need_file server/migrations/485_drop_agent_runtime_last_seen_at_index.up.sql
need_file server/migrations/485_drop_agent_runtime_last_seen_at_index.down.sql
need_file server/migrations/486_agent_runtime_online_last_seen_index.up.sql
need_file server/migrations/486_agent_runtime_online_last_seen_index.down.sql
need_file server/migrations/487_agent_runtime_offline_last_seen_index.up.sql
need_file server/migrations/487_agent_runtime_offline_last_seen_index.down.sql
need_file server/migrations/488_github_pr_head_sha_index.up.sql
need_file server/migrations/488_github_pr_head_sha_index.down.sql
need_file server/migrations/489_vcs_reference_only_repair.up.sql
need_file server/migrations/489_vcs_reference_only_repair.down.sql
need_file server/migrations/490_autopilot_trigger_created_by.up.sql
need_file server/migrations/490_autopilot_trigger_created_by.down.sql
need_file apps/web/app/\(auth\)/login/page.test.tsx
need_file server/internal/clerk/sdk.go

need_absent server/migrations/437_drop_agent_runtime_last_seen_at_index.up.sql
need_absent server/migrations/438_agent_runtime_online_last_seen_index.up.sql
need_absent server/migrations/440_github_pr_head_sha_index.up.sql
need_absent server/migrations/441_runtime_profile_add_codearts.up.sql
need_absent server/migrations/442_vcs_reference_only_repair.up.sql
need_absent server/migrations/448_autopilot_quota_rejection_notified_at.up.sql
need_absent server/migrations/449_autopilot_trigger_created_by.up.sql
need_absent packages/views/modals/issue-limit-upgrade-dialog.tsx
need_absent server/internal/service/issue_limit.go
need_absent server/pkg/agent/codearts.go

need_grep server/pkg/db/generated/models.go 'CoAuthoredByEmail'
need_grep server/pkg/db/generated/models.go 'PausedAt'
need_grep server/pkg/db/generated/models.go 'PausedByBudgetID'
need_grep server/pkg/db/generated/models.go 'ExecutionLane'
need_grep server/pkg/db/generated/models.go 'ClerkUserID'
need_grep server/pkg/db/generated/models.go 'ClerkOrgID'
need_grep server/pkg/db/generated/models.go 'type Initiative struct'
need_grep server/pkg/db/generated/models.go 'type MemoryEntry struct'
need_grep server/pkg/db/generated/models.go 'type AgentmailInbox struct'
need_grep server/pkg/agent/agent_supported_types_test.go '"amp": true'
need_grep server/pkg/agent/agent_supported_types_test.go '"devin": true'
need_grep server/pkg/agent/agent_supported_types_test.go '"goose": true'
need_absent_grep server/pkg/agent/agent_supported_types_test.go 'codearts'
need_grep server/cmd/migrate/main.go '485_drop_agent_runtime_last_seen_at_index'
need_grep server/cmd/migrate/main.go '486_agent_runtime_online_last_seen_index'
need_grep server/cmd/migrate/main.go '487_agent_runtime_offline_last_seen_index'
need_grep server/cmd/migrate/main.go '488_github_pr_head_sha_index'

if [ "$fail" -ne 0 ]; then
	exit 1
fi
printf 'OK: fork overlay intact\n'
