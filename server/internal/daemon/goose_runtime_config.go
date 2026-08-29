package daemon

import (
	"encoding/json"
	"log/slog"
	"strings"
)

// decodeGooseProvider reads runtime_config.goose_provider. The JSON key is
// namespaced so the bag can hold other runtimes' keys (same pattern as
// OpenClaw's typed bag). Malformed JSON inherits Goose config instead of
// failing the task.
func decodeGooseProvider(raw json.RawMessage, logger *slog.Logger) string {
	if len(raw) == 0 {
		return ""
	}
	var cfg struct {
		Provider string `json:"goose_provider"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		logger.Warn("goose runtime_config: parse failed; inheriting GOOSE_PROVIDER", "error", err)
		return ""
	}
	return strings.TrimSpace(cfg.Provider)
}
