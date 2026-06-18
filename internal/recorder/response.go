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

func RecordResponse(conn *sql.DB, sessionID, tokensJSON, model string) error {
	now := time.Now().UTC()

	// Resolve to the stable sessions.id (may differ from sessionID when process key is in use).
	stableID := resolveStableSessionID(conn, sessionID)
	if stableID == "" {
		stableID = sessionID // no matching session found; fall back to given ID
	}

	var tok tokenPayload
	if tokensJSON != "" {
		// Try flat format; fall back to nested {"usage": {...}}
		if err := json.Unmarshal([]byte(tokensJSON), &tok); err != nil || (tok.InputTokens == 0 && tok.OutputTokens == 0 && tok.CacheReadTokens == 0 && tok.CacheCreationTokens == 0) {
			var nested struct {
				Usage tokenPayload `json:"usage"`
			}
			if err2 := json.Unmarshal([]byte(tokensJSON), &nested); err2 == nil && (nested.Usage.InputTokens > 0 || nested.Usage.OutputTokens > 0 || nested.Usage.CacheReadTokens > 0 || nested.Usage.CacheCreationTokens > 0) {
				tok = nested.Usage
			}
		}
	}

	// Backfill model on session if not yet set
	if model != "" {
		conn.Exec(`UPDATE sessions SET model=? WHERE id=? AND (model='' OR model IS NULL)`, model, stableID)
	}

	// Look up model for this session to calculate cost (may have just been written)
	var sessionModel string
	conn.QueryRow("SELECT model FROM sessions WHERE id=?", stableID).Scan(&sessionModel)

	var cost *float64
	if tok.InputTokens > 0 || tok.OutputTokens > 0 {
		cost = pricing.Calculate(sessionModel, tok.InputTokens, tok.OutputTokens, tok.CacheReadTokens, tok.CacheCreationTokens)
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
		stableID,
	)
	return err
}
