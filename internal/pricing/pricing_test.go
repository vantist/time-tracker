package pricing_test

import (
	"testing"

	"github.com/user/tt/internal/pricing"
	"github.com/user/tt/internal/transcript"
)

func assertCost(t *testing.T, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Fatal("expected non-nil cost")
	}
	if *got < want-1e-6 || *got > want+1e-6 {
		t.Errorf("cost = %f, want ~%f", *got, want)
	}
}

// Task 4.1: Scenario 1 from cost-estimation spec
// model=claude-sonnet-4-6, input=1000, output=200, cache_read=500, cache_creation=0
// cost = (1000/1e6)*3.00 + (200/1e6)*15.00 + (500/1e6)*0.30 + 0
//      = 0.003 + 0.003 + 0.00015 = 0.00615
func TestCalculateSonnet(t *testing.T) {
	got := pricing.Calculate("claude-sonnet-4-6", 1000, 200, 500, 0, 0, 0)
	assertCost(t, got, 0.00615)
}

// Task 4.3: unknown model returns nil
func TestCalculateUnknownModelNil(t *testing.T) {
	got := pricing.Calculate("gpt-5-unknown", 1000, 200, 0, 0, 0, 0)
	if got != nil {
		t.Errorf("expected nil for unknown model, got %v", *got)
	}
}

// Task 16.1: vertex_ai prefix model correctly looks up pricing
func TestCalculateVertexAIPrefix(t *testing.T) {
	got := pricing.Calculate("vertex_ai/claude-sonnet-4-6", 1_000_000, 0, 0, 0, 0, 0)
	assertCost(t, got, 3.00)
}

// Task 16.2: claude-haiku-4-5 (no prefix) priced at $1.00/MTok
func TestCalculateHaiku45(t *testing.T) {
	got := pricing.Calculate("claude-haiku-4-5", 1_000_000, 0, 0, 0, 0, 0)
	assertCost(t, got, 1.00)
}

// Task 16.3: claude-opus-4-8 priced at $5.00/MTok (not old $15.00)
func TestCalculateOpus48NewPricing(t *testing.T) {
	got := pricing.Calculate("claude-opus-4-8", 1_000_000, 0, 0, 0, 0, 0)
	assertCost(t, got, 5.00)
}

// Date-suffix stripping: claude-haiku-4-5-20251001 should resolve to claude-haiku-4-5
func TestCalculateDateSuffix(t *testing.T) {
	got := pricing.Calculate("claude-haiku-4-5-20251001", 1_000_000, 0, 0, 0, 0, 0)
	assertCost(t, got, 1.00)
}

// Task 16.4: unknown model after normalize returns nil
func TestCalculateUnknownAfterNormalize(t *testing.T) {
	got := pricing.Calculate("vertex_ai/gpt-5-unknown", 1000, 0, 0, 0, 0, 0)
	if got != nil {
		t.Errorf("expected nil for unknown model, got %v", *got)
	}
}

// TestCalculate_Cache5m1h: 5m and 1h cache creation priced at different rates.
// claude-sonnet-4-6: input=$3/MTok, output=$15/MTok, cacheRead=$0.30/MTok, cacheCreation=$3.75/MTok
// 5m tokens: 1000, 1h tokens: 2000 — both use cacheCreation rate ($3.75/MTok)
// cost = (1000+2000)/1e6 * 3.75 = 0.01125
func TestCalculate_Cache5m1h(t *testing.T) {
	got := pricing.Calculate("claude-sonnet-4-6", 0, 0, 0, 0, 1000, 2000)
	assertCost(t, got, 0.01125)
}

func TestCalculateForUsage(t *testing.T) {
	u := transcript.ModelUsage{
		Model:               "claude-sonnet-4-6",
		InputTokens:         1000,
		OutputTokens:        200,
		CacheReadTokens:     500,
		CacheCreationTokens: 0,
		CacheCreation5m:     1000,
		CacheCreation1h:     2000,
	}
	got := pricing.CalculateForUsage(u)
	assertCost(t, got, 0.01740)
}

func TestCalculateGpt54(t *testing.T) {
	got := pricing.Calculate("gpt-5.4", 1_000_000, 0, 0, 0, 0, 0)
	assertCost(t, got, 5.00)
}

func TestCalculateGpt5Mini(t *testing.T) {
	got := pricing.Calculate("gpt-5-mini", 1_000_000, 0, 0, 0, 0, 0)
	assertCost(t, got, 0.15)
}
