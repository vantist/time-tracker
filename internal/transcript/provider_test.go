package transcript

import (
	"testing"
)

type mockProvider struct{}

func (m *mockProvider) ResolvePath(sessionID string, stdinPath string) string {
	return "mock-path"
}

func (m *mockProvider) ExtractWindow(path string, fromOffset int, toOffset int) (WindowResult, error) {
	return WindowResult{}, nil
}

func (m *mockProvider) ExtractLastTurn(path string) (WindowResult, error) {
	return WindowResult{}, nil
}

func (m *mockProvider) SupportsSubagents() bool {
	return true
}

func TestRegistry(t *testing.T) {
	mock := &mockProvider{}
	Register("mock-tool", mock)

	got, ok := GetProvider("mock-tool")
	if !ok {
		t.Fatal("expected mock-tool to be registered")
	}
	if got != mock {
		t.Errorf("got %v, want %v", got, mock)
	}

	_, ok = GetProvider("non-existent")
	if ok {
		t.Error("expected non-existent to not be registered")
	}
}
