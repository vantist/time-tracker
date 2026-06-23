package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseChatSession_ExtractModelInfo(t *testing.T) {
	content := `{
		"version": 3,
		"requesterUsername": "testuser",
		"responderUsername": "GitHub Copilot",
		"initialLocation": "editor",
		"requests": [
			{
				"requestId": "req-1",
				"result": {
					"modelId": "copilot/gpt-5-codex",
					"details": "GPT-5-Codex (Preview) • 1x"
				}
			}
		]
	}`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "session.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	session, err := ParseChatSession(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.RequesterUsername != "testuser" {
		t.Errorf("expected requesterUsername testuser, got %s", session.RequesterUsername)
	}
	if session.ResponderUsername != "GitHub Copilot" {
		t.Errorf("expected responderUsername GitHub Copilot, got %s", session.ResponderUsername)
	}
	if session.InitialLocation != "editor" {
		t.Errorf("expected initialLocation editor, got %s", session.InitialLocation)
	}

	if len(session.Requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(session.Requests))
	}

	req := session.Requests[0]
	if req.ModelID != "copilot/gpt-5-codex" {
		t.Errorf("expected modelId copilot/gpt-5-codex, got %s", req.ModelID)
	}
	if req.Result == nil {
		t.Fatal("expected result to be non-nil")
	}
	if req.Result.Details != "GPT-5-Codex (Preview) • 1x" {
		t.Errorf("expected details GPT-5-Codex (Preview) • 1x, got %s", req.Result.Details)
	}
}

func TestParseChatSession_ExtractThinkingTokens(t *testing.T) {
	content := `{
		"version": 3,
		"requests": [
			{
				"requestId": "req-1",
				"response": [
					{
						"kind": "thinking",
						"text": "thinking content",
						"tokens": 1792
					}
				]
			}
		]
	}`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "session.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	session, err := ParseChatSession(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(session.Requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(session.Requests))
	}

	req := session.Requests[0]
	if len(req.Response) != 1 {
		t.Fatalf("expected 1 response item, got %d", len(req.Response))
	}

	item := req.Response[0]
	if item.Tokens != 1792 {
		t.Errorf("expected 1792 thinking tokens, got %d", item.Tokens)
	}
}

func TestParseChatSession_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.json")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	session, err := ParseChatSession(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(session.Requests) != 0 {
		t.Errorf("expected 0 requests, got %d", len(session.Requests))
	}
}

func TestParseChatSession_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ParseChatSession(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
