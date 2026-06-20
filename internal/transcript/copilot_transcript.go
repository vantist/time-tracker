package transcript

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

type copilotUsage struct {
	InputTokens      int `json:"inputTokens"`
	OutputTokens     int `json:"outputTokens"`
	CacheReadTokens  int `json:"cacheReadTokens"`
	CacheWriteTokens int `json:"cacheWriteTokens"`
	ReasoningTokens  int `json:"reasoningTokens"`
}

type copilotModelMetrics struct {
	Usage copilotUsage `json:"usage"`
}

type copilotEvent struct {
	Type string `json:"type"`
	Data struct {
		ModelMetrics map[string]copilotModelMetrics `json:"modelMetrics"`
	} `json:"data"`
}

// ParseCopilotLog reads events.jsonl and extracts model metrics from session.shutdown.
func ParseCopilotLog(path string) (WindowResult, error) {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return WindowResult{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)

	// Group/sum usages by model name
	modelUsages := make(map[string]ModelUsage)

	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}

		var event copilotEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		if event.Type == "session.shutdown" {
			for modelName, metrics := range event.Data.ModelMetrics {
				u, exists := modelUsages[modelName]
				if !exists {
					u = ModelUsage{
						Model:      modelName,
						IsSubagent: false,
					}
				}
				u.InputTokens += metrics.Usage.InputTokens
				u.OutputTokens += metrics.Usage.OutputTokens + metrics.Usage.ReasoningTokens
				u.CacheReadTokens += metrics.Usage.CacheReadTokens
				u.CacheCreationTokens += metrics.Usage.CacheWriteTokens
				modelUsages[modelName] = u
			}
		}
	}

	if err := sc.Err(); err != nil {
		return WindowResult{}, err
	}

	var usages []ModelUsage
	for _, mu := range modelUsages {
		usages = append(usages, mu)
	}

	return WindowResult{Usages: usages}, nil
}
