package costrate

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPriceTicksAuthoritativeWins(t *testing.T) {
	ticks, by := PriceTicks(Line{
		Provider:     "anthropic",
		Model:        "claude-opus-5",
		CostUSDTicks: 42,
		InputTokens:  1_000_000,
	})
	if ticks != 42 || by != PricedByProvider {
		t.Fatalf("got %d %s, want 42 provider", ticks, by)
	}
}

func TestPriceTicksCatalogHit(t *testing.T) {
	ticks, by := PriceTicks(Line{
		Provider:    "anthropic",
		Model:       "claude-opus-5",
		InputTokens: 1_000_000,
	})
	if by != PricedByRateTable {
		t.Fatalf("by = %s, want rate_table", by)
	}
	// $5 / MTok input → 5 * 1e10 ticks
	if ticks != 5*TicksPerUSD {
		t.Fatalf("ticks = %d, want %d", ticks, 5*TicksPerUSD)
	}
}

func TestPriceTicksUnknownModelIsUnpriced(t *testing.T) {
	ticks, by := PriceTicks(Line{
		Provider:    "acme",
		Model:       "mystery-9",
		InputTokens: 1_000_000,
	})
	if ticks != 0 || by != PricedByUnpriced {
		t.Fatalf("got %d %s, want 0 unpriced", ticks, by)
	}
}

func TestLookupStripsSnapshotAndContextTag(t *testing.T) {
	rate, ok := Lookup("anthropic", "anthropic/claude-opus-4.7[1m]")
	if !ok || rate.Input != 5 {
		t.Fatalf("lookup = %+v ok=%v", rate, ok)
	}
}

func TestLookupCursorAutoNeedsProvider(t *testing.T) {
	if _, ok := Lookup("", "auto"); ok {
		t.Fatal("bare auto must stay unpriced")
	}
	rate, ok := Lookup("cursor", "auto")
	if !ok || rate.Input != 1.25 {
		t.Fatalf("cursor/auto = %+v ok=%v", rate, ok)
	}
}

func TestGeneratedCatalogHashesMatch(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	tsPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "packages", "core", "costrate", "table.ts")
	body, err := os.ReadFile(tsPath)
	if err != nil {
		t.Fatalf("read ts catalog: %v", err)
	}
	const prefix = `export const CATALOG_HASH = "`
	start := strings.Index(string(body), prefix)
	if start < 0 {
		t.Fatal("ts catalog missing CATALOG_HASH")
	}
	rest := string(body)[start+len(prefix):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatal("ts catalog hash unterminated")
	}
	tsHash := rest[:end]
	if tsHash != CatalogHash {
		t.Fatalf("catalog hash drift: go=%s ts=%s (run pnpm generate:model-rates)", CatalogHash, tsHash)
	}
}
