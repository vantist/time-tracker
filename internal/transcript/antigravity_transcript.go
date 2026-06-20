package transcript

// ParseAntigravityLog parses the transcript.jsonl file and returns main agent model usage.
func ParseAntigravityLog(path string) (WindowResult, error) {
	all, err := loadTranscript(path)
	if err != nil {
		return WindowResult{}, err
	}

	mainModel := findMainModel(all)
	acc := sumWindow(all, 0, len(all))

	var result WindowResult
	if acc.InputTokens > 0 || acc.OutputTokens > 0 || mainModel != "unknown" {
		result.Usages = append(result.Usages, makeMainUsage(mainModel, acc))
	}

	return result, nil
}
