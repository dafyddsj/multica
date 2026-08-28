-- name: CreateBudget :one
INSERT INTO budget (
    id, workspace_id, scope, owner_id, period,
    limit_usd_ticks, soften_at_percent, over_limit, created_by
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, gen_random_uuid()),
    $1, $2, $3, COALESCE(sqlc.narg('period'), 'monthly'),
    $4, sqlc.narg('soften_at_percent'), $5, $6
)
RETURNING *;

-- name: GetBudget :one
SELECT * FROM budget WHERE id = $1;

-- name: GetBudgetInWorkspace :one
SELECT * FROM budget WHERE id = $1 AND workspace_id = $2;

-- name: GetBudgetByScopeOwner :one
SELECT * FROM budget
WHERE workspace_id = $1 AND scope = $2 AND owner_id = $3;

-- name: ListBudgets :many
SELECT * FROM budget
WHERE workspace_id = $1
ORDER BY created_at;

-- name: UpdateBudget :one
UPDATE budget SET
    limit_usd_ticks = COALESCE(sqlc.narg('limit_usd_ticks'), limit_usd_ticks),
    soften_at_percent = CASE
        WHEN sqlc.narg('set_soften_at_percent')::bool THEN sqlc.narg('soften_at_percent')
        ELSE soften_at_percent
    END,
    over_limit = COALESCE(sqlc.narg('over_limit'), over_limit),
    updated_at = now()
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: DeleteBudget :exec
DELETE FROM budget WHERE id = $1 AND workspace_id = $2;

-- name: CreateBudgetPeriod :one
INSERT INTO budget_period (
    id, budget_id, workspace_id, period_start, period_end,
    spent_usd_ticks, unpriced_line_count
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, gen_random_uuid()),
    $1, $2, $3, $4,
    COALESCE(sqlc.narg('spent_usd_ticks'), 0),
    COALESCE(sqlc.narg('unpriced_line_count'), 0)
)
RETURNING *;

-- name: GetBudgetPeriod :one
SELECT * FROM budget_period WHERE id = $1;

-- name: GetBudgetPeriodByStart :one
SELECT * FROM budget_period
WHERE budget_id = $1 AND period_start = $2;

-- name: ListBudgetPeriods :many
SELECT * FROM budget_period
WHERE budget_id = $1
ORDER BY period_start DESC;

-- name: CreateBudgetDebit :one
INSERT INTO budget_debit (
    id, workspace_id, budget_id, budget_period_id,
    task_id, provider, model, amount_usd_ticks, priced_by
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, gen_random_uuid()),
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetBudgetDebit :one
SELECT * FROM budget_debit
WHERE budget_id = $1 AND task_id = $2 AND provider = $3 AND model = $4;

-- name: CreateBudgetWaiver :one
INSERT INTO budget_waiver (
    id, workspace_id, scope, owner_id, starts_at, ends_at, created_by, reason
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, gen_random_uuid()),
    $1, $2, $3, $4, $5, $6, sqlc.narg('reason')
)
RETURNING *;

-- name: GetBudgetWaiver :one
SELECT * FROM budget_waiver WHERE id = $1;

-- name: GetBudgetWaiverInWorkspace :one
SELECT * FROM budget_waiver WHERE id = $1 AND workspace_id = $2;

-- name: ListBudgetWaivers :many
SELECT * FROM budget_waiver
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: ListActiveBudgetWaivers :many
SELECT * FROM budget_waiver
WHERE workspace_id = $1
  AND starts_at <= $2
  AND ends_at > $2
ORDER BY created_at DESC;

-- name: DeleteBudgetWaiver :exec
DELETE FROM budget_waiver WHERE id = $1 AND workspace_id = $2;

-- name: DeleteBudgetDebits :exec
DELETE FROM budget_debit WHERE budget_id = $1;

-- name: DeleteBudgetPeriods :exec
DELETE FROM budget_period WHERE budget_id = $1;

-- name: UpsertBudgetDebit :one
INSERT INTO budget_debit (
    id, workspace_id, budget_id, budget_period_id,
    task_id, provider, model, amount_usd_ticks, priced_by
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, gen_random_uuid()),
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (budget_id, task_id, provider, model)
DO UPDATE SET
    amount_usd_ticks = EXCLUDED.amount_usd_ticks,
    priced_by = EXCLUDED.priced_by,
    budget_period_id = EXCLUDED.budget_period_id
RETURNING *;

-- name: RecalcBudgetPeriodTotals :one
UPDATE budget_period SET
    spent_usd_ticks = COALESCE((
        SELECT SUM(amount_usd_ticks) FROM budget_debit
        WHERE budget_period_id = budget_period.id AND priced_by <> 'unpriced'
    ), 0),
    unpriced_line_count = COALESCE((
        SELECT COUNT(*)::int FROM budget_debit
        WHERE budget_period_id = budget_period.id AND priced_by = 'unpriced'
    ), 0)
WHERE budget_period.id = $1
RETURNING *;

-- name: ListBudgetUsageForBackfill :many
SELECT
    tu.task_id,
    tu.provider,
    tu.model,
    tu.input_tokens,
    tu.output_tokens,
    tu.cache_read_tokens,
    tu.cache_write_tokens,
    tu.cost_usd_ticks
FROM task_usage tu
JOIN agent_task_queue atq ON atq.id = tu.task_id
JOIN agent a ON a.id = atq.agent_id
WHERE a.workspace_id = sqlc.arg('workspace_id')
  AND tu.updated_at >= sqlc.arg('period_start')
  AND tu.updated_at < sqlc.arg('period_end')
  AND (
    (sqlc.arg('scope')::text = 'agent' AND atq.agent_id = sqlc.arg('owner_id'))
    OR (sqlc.arg('scope')::text = 'squad' AND atq.budget_origin_squad_id = sqlc.arg('owner_id'))
    OR (sqlc.arg('scope')::text = 'project' AND atq.budget_project_id = sqlc.arg('owner_id'))
    OR (sqlc.arg('scope')::text = 'initiative' AND atq.budget_initiative_id = sqlc.arg('owner_id'))
  );

-- name: SquadHasOriginStamp :one
SELECT EXISTS (
    SELECT 1
    FROM agent_task_queue atq
    JOIN agent a ON a.id = atq.agent_id
    WHERE a.workspace_id = $1
      AND atq.budget_origin_squad_id = $2
)::bool;

-- name: CountOverlappingBudgetWaivers :one
SELECT COUNT(*)::int
FROM budget_waiver
WHERE workspace_id = $1
  AND scope = $2
  AND owner_id = $3
  AND starts_at < sqlc.arg('ends_at')
  AND ends_at > sqlc.arg('starts_at');
