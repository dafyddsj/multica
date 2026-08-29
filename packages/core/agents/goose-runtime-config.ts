// Goose-specific `runtime_config` schema.
//
// Stored under `agent.runtime_config` as freeform JSONB. The wire key is
// `goose_provider`, not `provider`, so the bag can hold other runtimes'
// keys the same way OpenClaw stores `mode` / `gateway`. Provider and model
// are independent Goose flags; a slash in a model id is not a provider
// boundary (OpenRouter ids look like `anthropic/claude-sonnet-4`).
// Empty / omitted means Goose uses `GOOSE_PROVIDER` from `goose configure`.
// Keep field names in lockstep with `server/internal/daemon/goose_runtime_config.go`.

export interface GooseRuntimeConfig {
  provider?: string;
}

export function parseGooseRuntimeConfig(raw: unknown): GooseRuntimeConfig {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return {};
  const root = raw as Record<string, unknown>;
  const out: GooseRuntimeConfig = {};
  if (typeof root.goose_provider === "string") {
    const provider = root.goose_provider.trim();
    if (provider !== "") out.provider = provider;
  }
  return out;
}

export function applyGooseProvider(
  raw: Record<string, unknown> | undefined,
  provider: string,
): Record<string, unknown> {
  const out: Record<string, unknown> = { ...(raw ?? {}) };
  const trimmed = provider.trim();
  if (trimmed !== "") {
    out.goose_provider = trimmed;
  } else {
    delete out.goose_provider;
  }
  return out;
}
