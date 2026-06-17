package pricing

type modelPricing struct {
	input          float64
	output         float64
	cacheRead      float64
	cacheCreation  float64
}

var table = map[string]modelPricing{
	"claude-haiku-4-5-20251001": {0.80, 4.00, 0.08, 1.00},
	"claude-sonnet-4-6":         {3.00, 15.00, 0.30, 3.75},
	"claude-opus-4-8":           {15.00, 75.00, 1.50, 18.75},
}

// Calculate returns estimated cost in USD, or nil for unknown models.
func Calculate(model string, inputTokens, outputTokens, cacheRead, cacheCreation int) *float64 {
	p, ok := table[model]
	if !ok {
		return nil
	}
	cost := float64(inputTokens)/1e6*p.input +
		float64(outputTokens)/1e6*p.output +
		float64(cacheRead)/1e6*p.cacheRead +
		float64(cacheCreation)/1e6*p.cacheCreation
	return &cost
}
