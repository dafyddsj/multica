-- One open month per budget. Spent is the ledger, not a live rollup.
CREATE TABLE budget_period (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    budget_id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    spent_usd_ticks BIGINT NOT NULL DEFAULT 0 CHECK (spent_usd_ticks >= 0),
    unpriced_line_count INTEGER NOT NULL DEFAULT 0 CHECK (unpriced_line_count >= 0),
    CHECK (period_end > period_start)
);
