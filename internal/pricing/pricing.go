package pricing

import (
	"regexp"
	"strings"

	"github.com/user/tt/internal/transcript"
)

var dateSuffix = regexp.MustCompile(`-\d{8}$`)
var versionSuffix = regexp.MustCompile(`-(latest|preview|exp|\d{3})$`)

type modelPricing struct {
	input         float64
	output        float64
	cacheRead     float64
	cacheCreation float64
}

var table = map[string]modelPricing{
	"claude-fable-5":          {10.00, 50.00, 1.00, 12.50},
	"claude-opus-4-8":         {5.00, 25.00, 0.50, 6.25},
	"claude-opus-4-7":         {5.00, 25.00, 0.50, 6.25},
	"claude-opus-4-6":         {5.00, 25.00, 0.50, 6.25},
	"claude-opus-4-5":         {5.00, 25.00, 0.50, 6.25},
	"claude-sonnet-4-6":       {3.00, 15.00, 0.30, 3.75},
	"claude-sonnet-4-5":       {3.00, 15.00, 0.30, 3.75},
	"claude-haiku-4-5":        {1.00, 5.00, 0.10, 1.25},
	"claude-haiku-3-5":        {0.80, 4.00, 0.08, 1.00},
	"gpt-5.4":                 {5.00, 15.00, 0.50, 6.25},
	"gpt-5-mini":              {0.15, 0.60, 0.015, 0.1875},
	"claude-3-opus":           {15.00, 75.00, 1.50, 18.75},
	"claude-3-sonnet":         {3.00, 15.00, 0.30, 3.75},
	"claude-3-haiku":          {0.25, 1.25, 0.025, 0.3125},
	"claude-3-5-sonnet":       {3.00, 15.00, 0.30, 3.75},
	"claude-3-5-haiku":        {0.80, 4.00, 0.08, 1.00},
	"gpt-4o":                  {2.50, 10.00, 1.25, 0.00},
	"gpt-4o-mini":             {0.15, 0.60, 0.075, 0.00},
	"o1":                      {15.00, 60.00, 7.50, 0.00},
	"o1-mini":                 {3.00, 12.00, 1.50, 0.00},
	"o3-mini":                 {1.10, 4.40, 0.55, 0.00},
	"gpt-5.3-codex":           {1.75, 14.00, 0.875, 0.00},
	"gpt-5.4-codex":           {2.50, 15.00, 1.25, 0.00},
	"gpt-5.5-codex":           {5.00, 30.00, 2.50, 0.00},
	"gpt-5.4-mini":            {0.75, 3.00, 0.375, 0.00},
	"gpt-5.5":                 {5.00, 30.00, 2.50, 0.00},
	"mai-code-1-flash":        {0.75, 4.50, 0.075, 0.00},
	"raptor-mini":             {0.25, 2.00, 0.025, 0.00},
	"grok-code-fast-1":        {1.00, 2.00, 0.10, 0.00},
	"gemini-1.5-pro":          {1.25, 5.00, 0.125, 0.00},
	"gemini-1.5-flash":        {0.075, 0.30, 0.0075, 0.00},
	"gemini-2.5-pro":          {1.25, 10.00, 0.125, 0.00},
	"gemini-2.5-flash":        {0.30, 2.50, 0.03, 0.00},
	"gemini-2.5-flash-lite":   {0.10, 0.40, 0.01, 0.00},
	"gemini-3-flash":          {0.50, 3.00, 0.05, 0.00},
	"gemini-3.1-pro":          {2.00, 12.00, 0.20, 0.00},
	"gemini-3.1-flash-lite":   {0.25, 1.50, 0.025, 0.00},
	"gemini-3.5-flash":        {1.50, 9.00, 0.15, 0.00},
}

func normalize(model string) string {
	if i := strings.LastIndex(model, "/"); i >= 0 {
		model = model[i+1:]
	}
	model = dateSuffix.ReplaceAllString(model, "")
	model = versionSuffix.ReplaceAllString(model, "")
	return model
}

// Calculate returns estimated cost in USD, or nil for unknown models.
// cacheCreate5m and cacheCreate1h are both priced at the cacheCreation rate.
func Calculate(model string, inputTokens, outputTokens, cacheRead, cacheCreation, cacheCreate5m, cacheCreate1h int) *float64 {
	p, ok := table[normalize(model)]
	if !ok {
		return nil
	}
	totalCreation := cacheCreation + cacheCreate5m + cacheCreate1h
	cost := (float64(inputTokens)*p.input +
		float64(outputTokens)*p.output +
		float64(cacheRead)*p.cacheRead +
		float64(totalCreation)*p.cacheCreation) / 1e6
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
