package reconcile

import (
	"database/sql"
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
	id                    int64
	sessionID             string
	transcriptPath        string
	promptLineOffset      int
	promptAt              time.Time
	responseAt            *time.Time // non-nil when Stop hook already set it
	processPID            int64
	processStart          int64
	nextOffset            *int
	nextTranscriptPath    string
	nextPromptAt          *time.Time
}

func reconcile(conn *sql.DB) {
	rows, err := conn.Query(`
		SELECT
			t.id, t.session_id, t.transcript_path, t.prompt_line_offset, t.prompt_at,
			t.response_at,
			s.process_pid, s.process_start,
			(SELECT prompt_line_offset FROM turns t2
			 WHERE t2.session_id = t.session_id AND t2.id > t.id
			 ORDER BY t2.id LIMIT 1) AS next_offset,
			(SELECT transcript_path FROM turns t2
			 WHERE t2.session_id = t.session_id AND t2.id > t.id
			 ORDER BY t2.id LIMIT 1) AS next_transcript_path,
			(SELECT prompt_at FROM turns t2
			 WHERE t2.session_id = t.session_id AND t2.id > t.id
			 ORDER BY t2.id LIMIT 1) AS next_prompt_at
		FROM turns t
		JOIN sessions s ON s.id = t.session_id
		WHERE (t.response_at IS NULL OR t.input_tokens IS NULL OR t.subagent_tokens_settled = 0)
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
		var nextTranscriptPath sql.NullString
		var nextPromptAtStr sql.NullString
		var promptAtStr string
		var responseAtStr sql.NullString
		err := rows.Scan(
			&dt.id, &dt.sessionID, &dt.transcriptPath, &dt.promptLineOffset, &promptAtStr,
			&responseAtStr,
			&dt.processPID, &dt.processStart,
			&nextOffset, &nextTranscriptPath, &nextPromptAtStr,
		)
		if err != nil {
			continue
		}
		dt.promptAt, _ = time.Parse(time.RFC3339Nano, promptAtStr)
		if responseAtStr.Valid {
			t, err := time.Parse(time.RFC3339Nano, responseAtStr.String)
			if err == nil {
				dt.responseAt = &t
			}
		}
		if nextOffset.Valid {
			v := int(nextOffset.Int64)
			dt.nextOffset = &v
		}
		if nextTranscriptPath.Valid {
			dt.nextTranscriptPath = nextTranscriptPath.String
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
		if dt.nextOffset != nil && dt.nextTranscriptPath == dt.transcriptPath {
			to = *dt.nextOffset
		}

		result, err := transcript.ExtractWindow(dt.transcriptPath, dt.promptLineOffset, to)
		if err != nil || (result.InputTokens() == 0 && result.OutputTokens() == 0) {
			continue
		}

		tx, err := conn.Begin()
		if err != nil {
			continue
		}

		// Delete old turn model usages
		_, err = tx.Exec("DELETE FROM turn_model_usages WHERE turn_id=?", dt.id)
		if err != nil {
			tx.Rollback()
			continue
		}

		// Insert new usages
		var totalCostVal float64
		var hasAnyCost bool

		for _, u := range result.Usages {
			costPtr := pricing.CalculateForUsage(u)
			var costVal float64
			if costPtr != nil {
				costVal = *costPtr
				totalCostVal += costVal
				hasAnyCost = true
			}

			_, err = tx.Exec(`
				INSERT INTO turn_model_usages (
					turn_id, model, is_subagent,
					input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens,
					cache_creation_5m_tokens, cache_creation_1h_tokens, estimated_cost_usd
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				dt.id,
				u.Model,
				u.IsSubagent,
				u.InputTokens,
				u.OutputTokens,
				u.CacheReadTokens,
				u.CacheCreationTokens,
				u.CacheCreation5m,
				u.CacheCreation1h,
				costVal,
			)
			if err != nil {
				tx.Rollback()
				break
			}
		}
		if err != nil {
			continue
		}

		var totalCost *float64
		if hasAnyCost {
			totalCost = &totalCostVal
		}

		if dt.responseAt != nil {
			// Stop hook already wrote response_at — overwrite tokens (subagent may be incomplete).
			_, err = tx.Exec(
				`UPDATE turns SET input_tokens=?, output_tokens=?, cache_read_tokens=?, cache_creation_tokens=?,
				 cache_creation_5m_tokens=?, cache_creation_1h_tokens=?, model=?,
				 estimated_cost_usd=?, subagent_tokens_settled=1
				 WHERE id=?`,
				result.InputTokens(), result.OutputTokens(), result.CacheReadTokens(), result.CacheCreationTokens(),
				result.CacheCreate5m(), result.CacheCreate1h(), result.Model(),
				totalCost,
				dt.id,
			)
		} else {
			var responseAt time.Time
			if dt.nextPromptAt != nil {
				responseAt = dt.nextPromptAt.Add(-time.Millisecond)
			} else {
				info, err := os.Stat(dt.transcriptPath)
				if err != nil {
					tx.Rollback()
					continue
				}
				responseAt = info.ModTime()
			}
			_, err = tx.Exec(
				`UPDATE turns SET response_at=?, input_tokens=?, output_tokens=?, cache_read_tokens=?, cache_creation_tokens=?,
				 cache_creation_5m_tokens=?, cache_creation_1h_tokens=?, model=?,
				 estimated_cost_usd=?, subagent_tokens_settled=1
				 WHERE id=? AND response_at IS NULL`,
				responseAt.UTC().Format(time.RFC3339Nano),
				result.InputTokens(), result.OutputTokens(), result.CacheReadTokens(), result.CacheCreationTokens(),
				result.CacheCreate5m(), result.CacheCreate1h(), result.Model(),
				totalCost,
				dt.id,
			)
		}

		if err != nil {
			tx.Rollback()
			continue
		}

		tx.Commit()
	}
}
