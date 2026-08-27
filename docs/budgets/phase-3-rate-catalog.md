# Phase 3. Rate catalog

Back to [overview.md](overview.md).

## Goal

Go and TypeScript price an uncosted usage line from the same checked-in table. Custom browser rates stay a display override and never enter the gate.

## Changes

- Add `scripts/model-rates.json` and a generator (`pnpm generate:model-rates`) that writes `server/internal/costrate/table.go` and a TS module consumed by `estimateCost`.
- `costrate.PriceTicks` returns `(ticks, pricedBy)` where `pricedBy` is `provider`, `rate_table`, or `unpriced`.
- Switch `packages/views/runtimes/utils.ts` `estimateCost` to the generated table. Keep custom-rate overlays on the estimated half only.
- Do not change `task_usage` writes.

## Data structures

Catalog key is `(normalized_provider, model)`. Value is per-token rates for input, output, cache read, and cache write. Unknown key is `unpriced`, not zero dollars pretending to be a price.

## Verification

**Static.** A test that the generated Go and TS fixtures hash-equal. `PriceTicks` cases for authoritative ticks, catalog hit, unknown model. Existing runtime cost tests still pass.

**Runtime.** Analytics Cost KPI for a known model matches the previous number for the same token mix. A custom rate still changes only the explorer chart, not a later budget bar.
