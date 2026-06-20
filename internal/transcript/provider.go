package transcript

import (
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
