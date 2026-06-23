package transcript

import (
	"os"
	"path/filepath"
	"sync"
)

type LogProvider interface {
	ResolvePath(sessionID string, stdinPath string) string
	ExtractWindow(path string, fromOffset int, toOffset int) (WindowResult, error)
	ExtractLastTurn(path string) (WindowResult, error)
	SupportsSubagents() bool
}

var (
	providersMu sync.RWMutex
	providers   = make(map[string]LogProvider)
)

func Register(tool string, p LogProvider) {
	providersMu.Lock()
	defer providersMu.Unlock()
	if p == nil {
		panic("transcript: Register provider is nil")
	}
	providers[tool] = p
}

func GetProvider(tool string) (LogProvider, bool) {
	providersMu.RLock()
	defer providersMu.RUnlock()
	p, ok := providers[tool]
	return p, ok
}

// JSONLProvider is a shared base for JSONL log formats.
type JSONLProvider struct {
	SupportsSub bool
}

func (j *JSONLProvider) ExtractWindow(path string, fromOffset int, toOffset int) (WindowResult, error) {
	return ExtractWindow(path, fromOffset, toOffset)
}

func (j *JSONLProvider) ExtractLastTurn(path string) (WindowResult, error) {
	return ExtractLastTurn(path)
}

func (j *JSONLProvider) SupportsSubagents() bool {
	return j.SupportsSub
}

// ClaudeProvider handles Claude Code log format.
type ClaudeProvider struct {
	JSONLProvider
}

func (p *ClaudeProvider) ResolvePath(sessionID string, stdinPath string) string {
	return stdinPath
}

// AntigravityProvider handles Google Antigravity log format.
type AntigravityProvider struct {
	JSONLProvider
}

func (p *AntigravityProvider) ResolvePath(sessionID string, stdinPath string) string {
	if stdinPath != "" {
		return stdinPath
	}
	cliPath := filepath.Join("~", ".gemini", "antigravity-cli", "brain", sessionID, ".system_generated", "logs", "transcript.jsonl")
	if _, err := os.Stat(expandHome(cliPath)); err == nil {
		return cliPath
	}
	return filepath.Join("~", ".gemini", "antigravity", "brain", sessionID, ".system_generated", "logs", "transcript.jsonl")
}

func (p *AntigravityProvider) ExtractWindow(path string, fromOffset int, toOffset int) (WindowResult, error) {
	return ParseAntigravityLogWindow(path, fromOffset, toOffset)
}

func (p *AntigravityProvider) ExtractLastTurn(path string) (WindowResult, error) {
	all, err := loadTranscript(path)
	if err != nil {
		return WindowResult{}, err
	}
	if len(all) == 0 {
		return WindowResult{}, nil
	}

	lastUserIdx := -1
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].Type == "USER_INPUT" {
			lastUserIdx = i
			break
		}
	}

	winFrom, winTo := lastUserIdx+1, len(all)
	hasSteps := false
	for i := winFrom; i < winTo; i++ {
		if all[i].StepIndex > 0 {
			hasSteps = true
			break
		}
	}

	if !hasSteps && lastUserIdx > 0 {
		prevUserIdx := -1
		for i := lastUserIdx - 1; i >= 0; i-- {
			if all[i].Type == "USER_INPUT" {
				prevUserIdx = i
				break
			}
		}
		winFrom, winTo = prevUserIdx+1, lastUserIdx
	}

	return ParseAntigravityLogWindow(path, winFrom, winTo)
}

// CodexProvider handles OpenAI Codex log format.
type CodexProvider struct {
	JSONLProvider
}

func (p *CodexProvider) ResolvePath(sessionID string, stdinPath string) string {
	return stdinPath
}

func init() {
	Register("claude-code", &ClaudeProvider{JSONLProvider{SupportsSub: true}})
	Register("antigravity", &AntigravityProvider{JSONLProvider{SupportsSub: true}})
	Register("codex", &CodexProvider{JSONLProvider{SupportsSub: false}})
	Register("copilot-cli", &CopilotProvider{})
	Register("vscode-copilot", &VSCodeCopilotProvider{})
}
