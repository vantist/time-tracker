package transcript

import (
	"encoding/json"
	"os"
)

// ChatSession represents a VS Code Copilot Chat session file (chatSessions/*.json).
type ChatSession struct {
	Version           int              `json:"version"`
	RequesterUsername string           `json:"requesterUsername"`
	ResponderUsername string           `json:"responderUsername"`
	InitialLocation   string           `json:"initialLocation"`
	Requests          []ChatRequest    `json:"requests"`
}

// ChatRequest represents a single request in a chat session.
type ChatRequest struct {
	RequestID string         `json:"requestId"`
	ModelID   string         `json:"modelId"`
	Result    *ChatResult    `json:"result"`
	Response  []ChatResponse `json:"response"`
}

// ChatResult contains model information from a request result.
type ChatResult struct {
	ModelID string `json:"modelId"`
	Details string `json:"details"`
}

// ChatResponse represents a response item in a chat session.
type ChatResponse struct {
	Kind   string `json:"kind"`
	Text   string `json:"text"`
	Tokens int    `json:"tokens"`
}

// ParseChatSession reads a chatSessions JSON file and extracts session metadata.
func ParseChatSession(path string) (*ChatSession, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session ChatSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	// Copy modelId from result to request level for convenience
	for i := range session.Requests {
		req := &session.Requests[i]
		if req.Result != nil && req.Result.ModelID != "" && req.ModelID == "" {
			req.ModelID = req.Result.ModelID
		}
	}

	return &session, nil
}
