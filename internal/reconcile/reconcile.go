package reconcile

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/user/tt/internal/pricing"
	"github.com/user/tt/internal/process"
	"github.com/user/tt/internal/transcript"
)

var mu sync.Mutex

// MaybeReconcile acquires in-process and cross-process locks then runs reconcile.
// Returns immediately if either lock is unavailable (another reconcile is running).
func MaybeReconcile(conn *sql.DB) {
	if !mu.TryLock() {
		return
	}
	defer mu.Unlock()

	path := lockPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	unlock, ok := tryLock(path)
	if !ok {
		return
	}
	defer unlock()

	reconcile(conn)
}

// HasActiveSession returns true if any session has a currently running process.
func HasActiveSession(conn *sql.DB) bool {
	return hasActiveSession(conn)
}

// hasActiveSession returns true if any session has a currently running process.
func hasActiveSession(conn *sql.DB) bool {
	rows, err := conn.Query("SELECT process_pid, process_start FROM sessions WHERE ended_at IS NULL")
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var pid, start int64
		if err := rows.Scan(&pid, &start); err != nil {
			continue
		}
		if process.IsAlive(pid, start) {
			return true
		}
	}
	return false
}

type danglingTurn struct {
	id               int64
	sessionID        string
	transcriptPath   string
	promptLineOffset int
	promptAt         time.Time
	processPID       int64
	processStart     int64
	nextOffset       *int
	nextPromptAt     *time.Time
}

func reconcile(conn *sql.DB) {
	rows, err := conn.Query(`
		SELECT
			t.id, t.session_id, t.transcript_path, t.prompt_line_offset, t.prompt_at,
			s.process_pid, s.process_start,
			(SELECT prompt_line_offset FROM turns t2
			 WHERE t2.session_id = t.session_id AND t2.id > t.id
			 ORDER BY t2.id LIMIT 1) AS next_offset,
			(SELECT prompt_at FROM turns t2
			 WHERE t2.session_id = t.session_id AND t2.id > t.id
			 ORDER BY t2.id LIMIT 1) AS next_prompt_at
		FROM turns t
		JOIN sessions s ON s.id = t.session_id
		WHERE t.response_at IS NULL
		  AND t.transcript_path IS NOT NULL
		  AND t.prompt_line_offset IS NOT NULL
	`)
	if err != nil {
		return
	}

	var turns []danglingTurn
	for rows.Next() {
		var dt danglingTurn
		var nextOffset sql.NullInt64
		var nextPromptAtStr sql.NullString
		var promptAtStr string
		err := rows.Scan(
			&dt.id, &dt.sessionID, &dt.transcriptPath, &dt.promptLineOffset, &promptAtStr,
			&dt.processPID, &dt.processStart,
			&nextOffset, &nextPromptAtStr,
		)
		if err != nil {
			continue
		}
		dt.promptAt, _ = time.Parse(time.RFC3339Nano, promptAtStr)
		if nextOffset.Valid {
			v := int(nextOffset.Int64)
			dt.nextOffset = &v
		}
		if nextPromptAtStr.Valid {
			t, err := time.Parse(time.RFC3339Nano, nextPromptAtStr.String)
			if err == nil {
				dt.nextPromptAt = &t
			}
		}
		turns = append(turns, dt)
	}
	rows.Close()

	for _, dt := range turns {
		isLatest := dt.nextOffset == nil
		if isLatest && process.IsAlive(dt.processPID, dt.processStart) {
			continue
		}

		to := -1
		if dt.nextOffset != nil {
			to = *dt.nextOffset
		}

		tokensJSON, model, err := transcript.ExtractWindow(dt.transcriptPath, dt.promptLineOffset, to)
		if err != nil || tokensJSON == "" {
			continue
		}

		var responseAt time.Time
		if dt.nextPromptAt != nil {
			responseAt = dt.nextPromptAt.Add(-time.Millisecond)
		} else {
			info, err := os.Stat(dt.transcriptPath)
			if err != nil {
				continue
			}
			responseAt = info.ModTime()
		}

		tokens := parseTokensJSON(tokensJSON)
		var cost *float64
		if model != "" {
			cost = pricing.Calculate(model, tokens.input, tokens.output, tokens.cacheRead, tokens.cacheCreate)
		}

		conn.Exec(
			`UPDATE turns SET response_at=?, input_tokens=?, output_tokens=?, cache_read_tokens=?, cache_creation_tokens=?, estimated_cost_usd=?
			 WHERE id=? AND response_at IS NULL`,
			responseAt.UTC().Format(time.RFC3339Nano),
			tokens.input, tokens.output, tokens.cacheRead, tokens.cacheCreate,
			cost,
			dt.id,
		)
	}
}

type tokenCounts struct {
	input, output, cacheRead, cacheCreate int
}

func parseTokensJSON(tokensJSON string) tokenCounts {
	var m map[string]int
	if err := json.Unmarshal([]byte(tokensJSON), &m); err != nil {
		return tokenCounts{}
	}
	return tokenCounts{
		input:       m["input_tokens"],
		output:      m["output_tokens"],
		cacheRead:   m["cache_read_tokens"],
		cacheCreate: m["cache_creation_tokens"],
	}
}
