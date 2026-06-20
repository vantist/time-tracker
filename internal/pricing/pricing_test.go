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

func TestCalculate(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		inputTokens   int
		outputTokens  int
		cacheRead     int
		cacheCreation int
		cache5m       int
		cache1h       int
		want          float64
		wantNil       bool
	}{
		{
			name:          "sonnet-4-6",
			model:         "claude-sonnet-4-6",
			inputTokens:   1000,
			outputTokens:  200,
			cacheRead:     500,
			cacheCreation: 0,
			want:          0.00615,
		},
		{
			name:         "unknown model",
			model:        "gpt-5-unknown",
			inputTokens:  1000,
			outputTokens: 200,
			wantNil:      true,
		},
		{
			name:        "vertex prefix",
			model:       "vertex_ai/claude-sonnet-4-6",
			inputTokens: 1_000_000,
			want:        3.00,
		},
		{
			name:        "haiku-4-5 no prefix",
			model:       "claude-haiku-4-5",
			inputTokens: 1_000_000,
			want:        1.00,
		},
		{
			name:        "opus-4-8 new pricing",
			model:       "claude-opus-4-8",
			inputTokens: 1_000_000,
			want:        5.00,
		},
		{
			name:        "date suffix stripping",
			model:       "claude-haiku-4-5-20251001",
			inputTokens: 1_000_000,
			want:        1.00,
		},
		{
			name:        "unknown model after normalize",
			model:       "vertex_ai/gpt-5-unknown",
			inputTokens: 1000,
			wantNil:     true,
		},
		{
			name:    "cache 5m 1h",
			model:   "claude-sonnet-4-6",
			cache5m: 1000,
			cache1h: 2000,
			want:    0.01125,
		},
		{
			name:        "gpt 5.4",
			model:       "gpt-5.4",
			inputTokens: 1_000_000,
			want:        5.00,
		},
		{
			name:        "gpt 5 mini",
			model:       "gpt-5-mini",
			inputTokens: 1_000_000,
			want:        0.15,
		},
		{
			name:        "suffix normalization - latest",
			model:       "claude-haiku-4-5-latest",
			inputTokens: 1_000_000,
			want:        1.00,
		},
		{
			name:        "suffix normalization - preview",
			model:       "claude-haiku-4-5-preview",
			inputTokens: 1_000_000,
			want:        1.00,
		},
		{
			name:        "suffix normalization - exp",
			model:       "claude-haiku-4-5-exp",
			inputTokens: 1_000_000,
			want:        1.00,
		},
		{
			name:        "suffix normalization - 002",
			model:       "claude-haiku-4-5-002",
			inputTokens: 1_000_000,
			want:        1.00,
		},
		{
			name:        "suffix normalization - gemini pro 002",
			model:       "gemini-1.5-pro-002",
			inputTokens: 1_000_000,
			want:        1.25,
		},
		{
			name:        "suffix normalization - claude latest",
			model:       "claude-3-5-sonnet-latest",
			inputTokens: 1_000_000,
			want:        3.00,
		},
		{
			name:        "suffix normalization - gpt preview",
			model:       "gpt-4o-preview",
			inputTokens: 1_000_000,
			want:        2.50,
		},
		{
			name:        "gemini-3.5-flash",
			model:       "gemini-3.5-flash",
			inputTokens: 1_000_000,
			want:        1.50,
		},
		{
			name:          "claude-3-5-sonnet",
			model:         "claude-3-5-sonnet",
			inputTokens:   1_000_000,
			cacheCreation: 1_000_000,
			want:          6.75,
		},
		{
			name:         "o1",
			model:        "o1",
			inputTokens:  1_000_000,
			outputTokens: 500_000,
			want:         45.00,
		},
		{
			name:      "gpt-4o",
			model:     "gpt-4o",
			inputTokens: 1_000_000,
			cacheRead: 2_000_000,
			want:      5.00,
		},
		{
			name:         "grok-code-fast-1",
			model:        "grok-code-fast-1",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			want:         3.00,
		},
		{
			name:         "mai-code-1-flash",
			model:        "mai-code-1-flash",
			inputTokens:  2_000_000,
			outputTokens: 1_000_000,
			want:         6.00,
		},
		{
			name:         "raptor-mini",
			model:        "raptor-mini",
			inputTokens:  4_000_000,
			outputTokens: 1_000_000,
			want:         3.00,
		},
		{
			name:         "gemini-2.5-flash-lite",
			model:        "gemini-2.5-flash-lite",
			inputTokens:  10_000_000,
			outputTokens: 5_000_000,
			want:         3.00,
		},
		{
			name:        "claude-fable-5",
			model:       "claude-fable-5",
			inputTokens: 1_000_000,
			cacheRead:   1_000_000,
			want:        11.00,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pricing.Calculate(tc.model, tc.inputTokens, tc.outputTokens, tc.cacheRead, tc.cacheCreation, tc.cache5m, tc.cache1h)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil for %s, got %v", tc.model, *got)
				}
			} else {
				assertCost(t, got, tc.want)
			}
		})
	}
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
