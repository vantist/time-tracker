package transcript

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

// VSCodeCopilotEvent represents a single event in a VS Code Copilot Chat transcript JSONL file.
type VSCodeCopilotEvent struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// VSCodeCopilotSessionStart is the data payload of a session.start event.
type VSCodeCopilotSessionStart struct {
	SessionID      string `json:"sessionId"`
	StartTime      string `json:"startTime"`
	CopilotVersion string `json:"copilotVersion"`
	VscodeVersion  string `json:"vscodeVersion"`
}

// VSCodeCopilotUserMessage is the data payload of a user.message event.
type VSCodeCopilotUserMessage struct {
	Content string `json:"content"`
}

// VSCodeCopilotAssistantMessage is the data payload of an assistant.message event.
type VSCodeCopilotAssistantMessage struct {
	Content       string          `json:"content"`
	ToolRequests  json.RawMessage `json:"toolRequests"`
	ReasoningText string          `json:"reasoningText"`
	OutputTokens  int             `json:"outputTokens"`
}

// VSCodeCopilotToolExecution is the data payload of tool.execution_start/complete events.
type VSCodeCopilotToolExecution struct {
	ToolCallID string `json:"toolCallId"`
	ToolName   string `json:"toolName"`
	Success    bool   `json:"success"`
}

// VSCodeCopilotSessionShutdown is the data payload of a session.shutdown event.
type VSCodeCopilotSessionShutdown struct {
	MainModel    string                             `json:"mainModel"`
	CurrentModel string                             `json:"currentModel"`
	ModelMetrics map[string]VSCodeCopilotMetrics    `json:"modelMetrics"`
	TotalNanoAiu float64                            `json:"totalNanoAiu"`
}

// VSCodeCopilotMetrics holds per-model token usage from a session.shutdown event.
type VSCodeCopilotMetrics struct {
	Usage struct {
		InputTokens      int `json:"inputTokens"`
		OutputTokens     int `json:"outputTokens"`
		CacheReadTokens  int `json:"cacheReadTokens"`
		CacheWriteTokens int `json:"cacheWriteTokens"`
	} `json:"usage"`
}

// VSCodeCopilotProvider implements LogProvider for VS Code Copilot Chat transcript files.
type VSCodeCopilotProvider struct{}

func (p *VSCodeCopilotProvider) ResolvePath(sessionID string, stdinPath string) string {
	if stdinPath != "" {
		return stdinPath
	}
	return filepath.Join("~", ".vscode", "extensions", "github.copilot-chat", "transcripts", sessionID+".jsonl")
}

func (p *VSCodeCopilotProvider) ExtractWindow(path string, fromOffset int, toOffset int) (WindowResult, error) {
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

		var event VSCodeCopilotEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		switch event.Type {
		case "session.shutdown":
			var shutdown VSCodeCopilotSessionShutdown
			if err := json.Unmarshal(event.Data, &shutdown); err != nil {
				continue
			}
			mainModel := shutdown.MainModel
			if mainModel == "" {
				mainModel = shutdown.CurrentModel
			}
			for modelName, metrics := range shutdown.ModelMetrics {
				isSub := mainModel != "" && modelName != mainModel
				k := usageKey{model: modelName, isSubagent: isSub}
				u, exists := modelUsages[k]
				if !exists {
					u = ModelUsage{
						Model:      modelName,
						IsSubagent: isSub,
					}
				}
				u.InputTokens += metrics.Usage.InputTokens
				u.OutputTokens += metrics.Usage.OutputTokens
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

func (p *VSCodeCopilotProvider) ExtractLastTurn(path string) (WindowResult, error) {
	return p.ExtractWindow(path, 0, -1)
}

func (p *VSCodeCopilotProvider) SupportsSubagents() bool {
	return false
}
