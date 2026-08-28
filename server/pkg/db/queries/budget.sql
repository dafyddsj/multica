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
    soften_at_percent = sqlc.narg('soften_at_percent'),
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
