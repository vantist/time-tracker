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
}
