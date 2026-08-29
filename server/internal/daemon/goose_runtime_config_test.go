package daemon

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestDecodeGooseProvider(t *testing.T) {
	t.Parallel()

	if got := decodeGooseProvider(nil, quietLogger()); got != "" {
		t.Errorf("nil payload: got %q, want \"\"", got)
	}
	if got := decodeGooseProvider(json.RawMessage(`{}`), quietLogger()); got != "" {
		t.Errorf("empty object: got %q, want \"\"", got)
	}

	raw := json.RawMessage(`{"goose_provider":" ollama ","mode":"local"}`)
	if got := decodeGooseProvider(raw, quietLogger()); got != "ollama" {
		t.Errorf("sibling keys: got %q, want ollama", got)
	}

	if got := decodeGooseProvider(json.RawMessage(`{"provider":"ollama"}`), quietLogger()); got != "" {
		t.Errorf("bare provider key: got %q, want \"\"", got)
	}
}

func TestDecodeGooseProviderMalformedFailsSoft(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	if got := decodeGooseProvider(json.RawMessage(`{"goose_provider":`), logger); got != "" {
		t.Errorf("malformed payload: got %q, want \"\"", got)
	}
	if !strings.Contains(buf.String(), "parse failed") {
		t.Errorf("expected WARN about parse failure, got: %q", buf.String())
	}
}
