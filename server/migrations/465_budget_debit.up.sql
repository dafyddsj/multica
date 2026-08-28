-- Idempotent debit. Re-reports adjust the period by the delta.
CREATE TABLE budget_debit (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    budget_id UUID NOT NULL,
    budget_period_id UUID NOT NULL,
    task_id UUID NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    amount_usd_ticks BIGINT NOT NULL DEFAULT 0 CHECK (amount_usd_ticks >= 0),
    priced_by TEXT NOT NULL CHECK (priced_by IN ('provider', 'rate_table', 'unpriced'))
);
