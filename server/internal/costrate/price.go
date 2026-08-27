package costrate

import (
	"math"
	"regexp"
	"strings"
)

const TicksPerUSD int64 = 10_000_000_000

type PricedBy string

const (
	PricedByProvider  PricedBy = "provider"
	PricedByRateTable PricedBy = "rate_table"
	PricedByUnpriced  PricedBy = "unpriced"
)

type Line struct {
	Provider         string
	Model            string
	CostUSDTicks     int64
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
}

var (
	dateSuffix     = regexp.MustCompile(`-(20\d{2}-\d{2}-\d{2}|20\d{6}|latest)$`)
	contextTag     = regexp.MustCompile(`\[[^\]]+\]$`)
	routingSegment = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
)

// PriceTicks prices one usage line. Authoritative ticks win. Otherwise the
// catalog. An unknown model is unpriced, not zero dollars.
func PriceTicks(line Line) (int64, PricedBy) {
	if line.CostUSDTicks > 0 {
		return line.CostUSDTicks, PricedByProvider
	}
	rate, ok := Lookup(line.Provider, line.Model)
	if !ok {
		return 0, PricedByUnpriced
	}
	usd := float64(line.InputTokens)*rate.Input +
		float64(line.OutputTokens)*rate.Output +
		float64(line.CacheReadTokens)*rate.CacheRead +
		float64(line.CacheWriteTokens)*rate.CacheWrite
	return usdToTicks(usd / 1_000_000), PricedByRateTable
}

func usdToTicks(usd float64) int64 {
	return int64(math.Round(usd * float64(TicksPerUSD)))
}

// Lookup finds a catalog row using the same candidate order as the TypeScript
// resolver: provider-qualified keys first, then bare model ids, after stripping
// routing prefixes, Anthropic dots, dated snapshots, and context tags.
func Lookup(provider, model string) (Rate, bool) {
	for _, key := range pricingCandidates(model, provider) {
		if rate, ok := table[key]; ok {
			return rate, true
		}
	}
	return Rate{}, false
}

func pricingCandidates(model, provider string) []string {
	base := canonicalCandidates(model)
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "" {
		return base
	}
	out := make([]string, 0, len(base)*2)
	seen := make(map[string]struct{}, len(base)*2)
	push := func(s string) {
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, c := range base {
		push(qualify(p, c))
	}
	for _, c := range base {
		push(c)
	}
	return out
}

func qualify(provider, key string) string {
	prefix := provider + "/"
	if strings.HasPrefix(key, prefix) {
		return key
	}
	return prefix + key
}

func canonicalCandidates(model string) []string {
	seen := make(map[string]struct{}, 8)
	out := make([]string, 0, 8)
	push := func(s string) {
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	raw := model
	noProvider := stripProvider(raw)
	dashed := canonAnthropic(noProvider)
	noTag := contextTag.ReplaceAllString(dashed, "")
	push(raw)
	push(noProvider)
	push(dashed)
	push(noTag)
	push(dateSuffix.ReplaceAllString(raw, ""))
	push(dateSuffix.ReplaceAllString(noProvider, ""))
	push(dateSuffix.ReplaceAllString(dashed, ""))
	push(dateSuffix.ReplaceAllString(noTag, ""))
	return out
}

func stripProvider(s string) string {
	out := s
	for {
		i := strings.IndexAny(out, "/:")
		if i <= 0 || !routingSegment.MatchString(strings.ToLower(out[:i])) {
			return out
		}
		out = out[i+1:]
	}
}

func canonAnthropic(s string) string {
	if strings.HasPrefix(s, "claude-") {
		return strings.ReplaceAll(s, ".", "-")
	}
	return s
}
