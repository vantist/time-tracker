package transcript

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

// DebugLogEvent represents a single event in a debug log JSONL file.
type DebugLogEvent struct {
	V    int             `json:"v"`
	Type string          `json:"type"`
	TS   int64           `json:"ts"`
	Attrs json.RawMessage `json:"attrs"`
	Data  json.RawMessage `json:"data"`
}

// DebugLLMRequestAttrs contains token counts from an llm_request event.
type DebugLLMRequestAttrs struct {
	Model         string `json:"model"`
	InputTokens   int    `json:"inputTokens"`
	OutputTokens  int    `json:"outputTokens"`
	CachedTokens  int    `json:"cachedTokens"`
	NanoAiu       int64  `json:"copilotUsageNanoAiu"`
}

// DebugSessionShutdownData contains per-model usage from a session.shutdown event.
type DebugSessionShutdownData struct {
	MainModel    string                              `json:"mainModel"`
	CurrentModel string                              `json:"currentModel"`
	ModelMetrics map[string]DebugShutdownMetrics     `json:"modelMetrics"`
	TotalNanoAiu float64                             `json:"totalNanoAiu"`
}

// DebugShutdownMetrics holds per-model token usage from a shutdown event.
type DebugShutdownMetrics struct {
	Usage struct {
		InputTokens      int `json:"inputTokens"`
		OutputTokens     int `json:"outputTokens"`
		CacheReadTokens  int `json:"cacheReadTokens"`
		CacheWriteTokens int `json:"cacheWriteTokens"`
	} `json:"usage"`
}

// DebugLogResult holds aggregated token usage from a debug log.
type DebugLogResult struct {
	InputTokens      int
	OutputTokens     int
	CachedTokens     int
	ModelTurns       int
	ModelBreakdown   map[string]ModelUsage
	TotalNanoAiu     float64
}

// ParseDebugLog reads a debug log JSONL file and extracts token usage.
func ParseDebugLog(path string) (*DebugLogResult, error) {
	f, err := os.Open(expandHome(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := &DebugLogResult{
		ModelBreakdown: make(map[string]ModelUsage),
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)

	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}

		var event DebugLogEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		switch event.Type {
		case "llm_request":
			var attrs DebugLLMRequestAttrs
			if err := json.Unmarshal(event.Attrs, &attrs); err != nil {
				continue
			}
			result.InputTokens += attrs.InputTokens
			result.OutputTokens += attrs.OutputTokens
			result.CachedTokens += attrs.CachedTokens
			result.ModelTurns++

			if attrs.Model != "" {
				u := result.ModelBreakdown[attrs.Model]
				u.Model = attrs.Model
				u.InputTokens += attrs.InputTokens
				u.OutputTokens += attrs.OutputTokens
				result.ModelBreakdown[attrs.Model] = u
			}

		case "session.shutdown":
			var data DebugSessionShutdownData
			if err := json.Unmarshal(event.Data, &data); err != nil {
				continue
			}
			result.TotalNanoAiu += data.TotalNanoAiu

			mainModel := data.MainModel
			if mainModel == "" {
				mainModel = data.CurrentModel
			}
			for modelName, metrics := range data.ModelMetrics {
				isSub := mainModel != "" && modelName != mainModel
				u := result.ModelBreakdown[modelName]
				u.Model = modelName
				u.IsSubagent = isSub
				u.InputTokens += metrics.Usage.InputTokens
				u.OutputTokens += metrics.Usage.OutputTokens
				u.CacheReadTokens += metrics.Usage.CacheReadTokens
				u.CacheCreationTokens += metrics.Usage.CacheWriteTokens
				result.ModelBreakdown[modelName] = u
			}
		}
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// DiscoverDebugLogPath returns the debug log path for a given session ID and workspaceStorage root.
func DiscoverDebugLogPath(workspaceStoragePath string, sessionID string) string {
	return filepath.Join(workspaceStoragePath, "debug-logs", sessionID, "main.jsonl")
}
