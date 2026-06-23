package recorder

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/user/tt/internal/pricing"
	"github.com/user/tt/internal/transcript"
)

type tokenPayload struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	CacheReadTokens     int `json:"cache_read_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens"`
	CacheCreate5m       int `json:"cache_creation_5m_tokens"`
	CacheCreate1h       int `json:"cache_creation_1h_tokens"`
}

func RecordResponse(conn *sql.DB, sessionID, tokensJSON, model string, subagentTokensJSON string) error {
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
		cost = pricing.Calculate(sessionModel, tok.InputTokens, tok.OutputTokens, tok.CacheReadTokens, tok.CacheCreationTokens, tok.CacheCreate5m, tok.CacheCreate1h)
	}

	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var turnID int64
	err = tx.QueryRow("SELECT id FROM turns WHERE session_id=? AND response_at IS NULL ORDER BY id DESC LIMIT 1", stableID).Scan(&turnID)
	if err == sql.ErrNoRows {
		return nil // silently skip
	} else if err != nil {
		return err
	}

	// Update the turn
	_, err = tx.Exec(`
		UPDATE turns SET
			response_at                = ?,
			input_tokens               = CASE WHEN ? > 0 THEN ? ELSE input_tokens END,
			output_tokens              = CASE WHEN ? > 0 THEN ? ELSE output_tokens END,
			cache_read_tokens          = ?,
			cache_creation_tokens      = ?,
			cache_creation_5m_tokens   = ?,
			cache_creation_1h_tokens   = ?,
			estimated_cost_usd         = ?,
			subagent_tokens_settled    = 0
		WHERE id = ?`,
		now.Format(time.RFC3339),
		tok.InputTokens, tok.InputTokens,
		tok.OutputTokens, tok.OutputTokens,
		tok.CacheReadTokens,
		tok.CacheCreationTokens,
		tok.CacheCreate5m,
		tok.CacheCreate1h,
		cost,
		turnID,
	)
	if err != nil {
		return err
	}

	// Write main agent usage to turn_model_usages
	var costVal float64
	if cost != nil {
		costVal = *cost
	}
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO turn_model_usages (
			turn_id, model, is_subagent,
			input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens,
			cache_creation_5m_tokens, cache_creation_1h_tokens, estimated_cost_usd
		) VALUES (?, ?, 0, ?, ?, ?, ?, ?, ?, ?)`,
		turnID,
		sessionModel,
		tok.InputTokens,
		tok.OutputTokens,
		tok.CacheReadTokens,
		tok.CacheCreationTokens,
		tok.CacheCreate5m,
		tok.CacheCreate1h,
		costVal,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// parseSubagentTokensJSON parses an opencode --subagent-tokens JSON array into
// ModelUsage entries marked as subagents. Missing optional token fields
// default to 0 (schema is NOT NULL DEFAULT 0). Returns (nil, nil) on empty
// input so callers can skip the DB step entirely.
func parseSubagentTokensJSON(s string) ([]transcript.ModelUsage, error) {
	if s == "" {
		return nil, nil
	}
	var entries []struct {
		Model           string `json:"model"`
		Agent           string `json:"agent"`
		InputTokens     int    `json:"input_tokens"`
		OutputTokens    int    `json:"output_tokens"`
		CacheReadTokens int    `json:"cache_read_tokens"`
		CacheCreation   int    `json:"cache_creation_tokens"`
		ReasoningTokens int    `json:"reasoning_tokens"`
	}
	if err := json.Unmarshal([]byte(s), &entries); err != nil {
		return nil, err
	}
	out := make([]transcript.ModelUsage, 0, len(entries))
	for _, e := range entries {
		out = append(out, transcript.ModelUsage{
			Model:               e.Model,
			IsSubagent:          true,
			InputTokens:         e.InputTokens,
			OutputTokens:        e.OutputTokens,
			CacheReadTokens:     e.CacheReadTokens,
			CacheCreationTokens: e.CacheCreation,
			CacheCreation5m:     0,
			CacheCreation1h:     0,
		})
	}
	return out, nil
}
