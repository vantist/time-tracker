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
		MainModel    string                         `json:"mainModel"`
		CurrentModel string                         `json:"currentModel"`
		ModelMetrics map[string]copilotModelMetrics `json:"modelMetrics"`
	} `json:"data"`
}

// ParseCopilotLog reads events.jsonl and extracts model metrics from session.shutdown.
func ParseCopilotLog(path string) (WindowResult, error) {
	p := &CopilotProvider{}
	return p.ExtractWindow(path, 0, -1)
}

// CopilotProvider implements LogProvider for Copilot CLI events.jsonl logs.
type CopilotProvider struct{}

func (p *CopilotProvider) ResolvePath(sessionID string, stdinPath string) string {
	if stdinPath != "" {
		return stdinPath
	}
	return filepath.Join("~", ".copilot", "session-state", sessionID, "events.jsonl")
}

func (p *CopilotProvider) ExtractWindow(path string, fromOffset int, toOffset int) (WindowResult, error) {
	f, err := os.Open(expandHome(path))
	if err != nil {
		return WindowResult{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)

	type usageKey struct {
		model      string
		isSubagent bool
	}
	modelUsages := make(map[usageKey]ModelUsage)

	for idx := 0; sc.Scan(); idx++ {
		if idx < fromOffset {
			continue
		}
		if toOffset != -1 && idx >= toOffset {
			break
		}

		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}

		var event copilotEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		if event.Type == "session.shutdown" {
			mainModel := event.Data.MainModel
			if mainModel == "" {
				mainModel = event.Data.CurrentModel
			}
			for modelName, metrics := range event.Data.ModelMetrics {
				isSub := false
				if mainModel != "" && modelName != mainModel {
					isSub = true
				}
				k := usageKey{model: modelName, isSubagent: isSub}
				u, exists := modelUsages[k]
				if !exists {
					u = ModelUsage{
						Model:      modelName,
						IsSubagent: isSub,
					}
				}
				u.InputTokens += metrics.Usage.InputTokens
				u.OutputTokens += metrics.Usage.OutputTokens + metrics.Usage.ReasoningTokens
				u.CacheReadTokens += metrics.Usage.CacheReadTokens
				u.CacheCreationTokens += metrics.Usage.CacheWriteTokens
				modelUsages[k] = u
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

func (p *CopilotProvider) ExtractLastTurn(path string) (WindowResult, error) {
	return p.ExtractWindow(path, 0, -1)
}

func (p *CopilotProvider) SupportsSubagents() bool {
	return true
}
