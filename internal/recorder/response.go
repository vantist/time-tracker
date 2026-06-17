package recorder

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/user/tt/internal/pricing"
)

type tokenPayload struct {
	InputTokens          int `json:"input_tokens"`
	OutputTokens         int `json:"output_tokens"`
	CacheReadTokens      int `json:"cache_read_tokens"`
	CacheCreationTokens  int `json:"cache_creation_tokens"`
}

func RecordResponse(conn *sql.DB, sessionID, tokensJSON string) error {
	now := time.Now().UTC()

	var tok tokenPayload
	if tokensJSON != "" {
		// Try flat format first, then nested {"usage": {...}}
		if err := json.Unmarshal([]byte(tokensJSON), &tok); err == nil && tok.InputTokens == 0 {
			var nested struct {
				Usage tokenPayload `json:"usage"`
			}
			if err2 := json.Unmarshal([]byte(tokensJSON), &nested); err2 == nil && nested.Usage.InputTokens > 0 {
				tok = nested.Usage
			}
		} else if err != nil {
			var nested struct {
				Usage tokenPayload `json:"usage"`
			}
			_ = json.Unmarshal([]byte(tokensJSON), &nested)
			tok = nested.Usage
		}
	}

	// Look up model for this session to calculate cost
	var model string
	conn.QueryRow("SELECT model FROM sessions WHERE id=?", sessionID).Scan(&model)

	var cost *float64
	if tok.InputTokens > 0 || tok.OutputTokens > 0 {
		cost = pricing.Calculate(model, tok.InputTokens, tok.OutputTokens, tok.CacheReadTokens, tok.CacheCreationTokens)
	}

	// Update the latest turn for this session (highest rowid)
	_, err := conn.Exec(`
		UPDATE turns SET
			response_at          = ?,
			input_tokens         = CASE WHEN ? > 0 THEN ? ELSE input_tokens END,
			output_tokens        = CASE WHEN ? > 0 THEN ? ELSE output_tokens END,
			cache_read_tokens    = ?,
			cache_creation_tokens = ?,
			estimated_cost_usd   = ?
		WHERE id = (
			SELECT id FROM turns WHERE session_id=? ORDER BY id DESC LIMIT 1
		)`,
		now.Format(time.RFC3339),
		tok.InputTokens, tok.InputTokens,
		tok.OutputTokens, tok.OutputTokens,
		tok.CacheReadTokens,
		tok.CacheCreationTokens,
		cost,
		sessionID,
	)
	return err
}
