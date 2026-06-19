package pricing

import (
	"regexp"
	"strings"

	"github.com/user/tt/internal/transcript"
)

var dateSuffix = regexp.MustCompile(`-\d{8}$`)

type modelPricing struct {
	input         float64
	output        float64
	cacheRead     float64
	cacheCreation float64
}

var table = map[string]modelPricing{
	"claude-fable-5":    {10.00, 50.00, 1.00, 12.50},
	"claude-opus-4-8":   {5.00, 25.00, 0.50, 6.25},
	"claude-opus-4-7":   {5.00, 25.00, 0.50, 6.25},
	"claude-opus-4-6":   {5.00, 25.00, 0.50, 6.25},
	"claude-opus-4-5":   {5.00, 25.00, 0.50, 6.25},
	"claude-sonnet-4-6": {3.00, 15.00, 0.30, 3.75},
	"claude-sonnet-4-5": {3.00, 15.00, 0.30, 3.75},
	"claude-haiku-4-5":  {1.00, 5.00, 0.10, 1.25},
	"claude-haiku-3-5":  {0.80, 4.00, 0.08, 1.00},
}

func normalize(model string) string {
	if i := strings.LastIndex(model, "/"); i >= 0 {
		model = model[i+1:]
	}
	return dateSuffix.ReplaceAllString(model, "")
}

// Calculate returns estimated cost in USD, or nil for unknown models.
// cacheCreate5m and cacheCreate1h are both priced at the cacheCreation rate.
func Calculate(model string, inputTokens, outputTokens, cacheRead, cacheCreation, cacheCreate5m, cacheCreate1h int) *float64 {
	p, ok := table[normalize(model)]
	if !ok {
		return nil
	}
	cost := float64(inputTokens)/1e6*p.input +
		float64(outputTokens)/1e6*p.output +
		float64(cacheRead)/1e6*p.cacheRead +
		float64(cacheCreation)/1e6*p.cacheCreation +
		float64(cacheCreate5m)/1e6*p.cacheCreation +
		float64(cacheCreate1h)/1e6*p.cacheCreation
	return &cost
}

func CalculateForUsage(u transcript.ModelUsage) *float64 {
	return Calculate(
		u.Model,
		u.InputTokens,
		u.OutputTokens,
		u.CacheReadTokens,
		u.CacheCreationTokens,
		u.CacheCreation5m,
		u.CacheCreation1h,
	)
}
