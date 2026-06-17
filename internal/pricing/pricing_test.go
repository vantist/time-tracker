package pricing_test

import (
	"testing"

	"github.com/user/tt/internal/pricing"
)

// Task 4.1: Scenario 1 from cost-estimation spec
// model=claude-sonnet-4-6, input=1000, output=200, cache_read=500, cache_creation=0
// cost = (1000/1e6)*3.00 + (200/1e6)*15.00 + (500/1e6)*0.30 + 0
//      = 0.003 + 0.003 + 0.00015 = 0.00615
func TestCalculateSonnet(t *testing.T) {
	got := pricing.Calculate("claude-sonnet-4-6", 1000, 200, 500, 0)
	if got == nil {
		t.Fatal("expected non-nil cost")
	}
	const want = 0.00615
	if *got < want-0.000001 || *got > want+0.000001 {
		t.Errorf("cost = %f, want ~%f", *got, want)
	}
}

// Task 4.3: unknown model returns nil
func TestCalculateUnknownModelNil(t *testing.T) {
	got := pricing.Calculate("gpt-5-unknown", 1000, 200, 0, 0)
	if got != nil {
		t.Errorf("expected nil for unknown model, got %v", *got)
	}
}

// Task 16.1: vertex_ai prefix model correctly looks up pricing
func TestCalculateVertexAIPrefix(t *testing.T) {
	got := pricing.Calculate("vertex_ai/claude-sonnet-4-6", 1_000_000, 0, 0, 0)
	if got == nil {
		t.Fatal("expected non-nil cost for vertex_ai/claude-sonnet-4-6")
	}
	const want = 3.00
	if *got < want-0.001 || *got > want+0.001 {
		t.Errorf("cost = %f, want ~%f", *got, want)
	}
}

// Task 16.2: claude-haiku-4-5 (no prefix) priced at $1.00/MTok
func TestCalculateHaiku45(t *testing.T) {
	got := pricing.Calculate("claude-haiku-4-5", 1_000_000, 0, 0, 0)
	if got == nil {
		t.Fatal("expected non-nil cost for claude-haiku-4-5")
	}
	const want = 1.00
	if *got < want-0.001 || *got > want+0.001 {
		t.Errorf("cost = %f, want ~%f", *got, want)
	}
}

// Task 16.3: claude-opus-4-8 priced at $5.00/MTok (not old $15.00)
func TestCalculateOpus48NewPricing(t *testing.T) {
	got := pricing.Calculate("claude-opus-4-8", 1_000_000, 0, 0, 0)
	if got == nil {
		t.Fatal("expected non-nil cost for claude-opus-4-8")
	}
	const want = 5.00
	if *got < want-0.001 || *got > want+0.001 {
		t.Errorf("cost = %f, want ~%f (old pricing was $15)", *got, want)
	}
}

// Date-suffix stripping: claude-haiku-4-5-20251001 should resolve to claude-haiku-4-5
func TestCalculateDateSuffix(t *testing.T) {
	got := pricing.Calculate("claude-haiku-4-5-20251001", 1_000_000, 0, 0, 0)
	if got == nil {
		t.Fatal("expected non-nil cost for claude-haiku-4-5-20251001 (date suffix should be stripped)")
	}
	const want = 1.00
	if *got < want-0.001 || *got > want+0.001 {
		t.Errorf("cost = %f, want ~%f", *got, want)
	}
}

// Task 16.4: unknown model after normalize returns nil
func TestCalculateUnknownAfterNormalize(t *testing.T) {
	got := pricing.Calculate("vertex_ai/gpt-5-unknown", 1000, 0, 0, 0)
	if got != nil {
		t.Errorf("expected nil for unknown model, got %v", *got)
	}
}
