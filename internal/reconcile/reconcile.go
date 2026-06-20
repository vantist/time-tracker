package reconcile

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	id                 int64
	sessionID          string
	transcriptPath     string
	promptLineOffset   int
	promptAt           time.Time
	responseAt         *time.Time // non-nil when Stop hook already set it
	processPID         int64
	processStart       int64
	tool               string
	nextOffset         *int
	nextTranscriptPath *string
	nextPromptAt       *time.Time
}

func reconcile(conn *sql.DB) {
	repairSessions(conn)

	rows, err := conn.Query(`
		SELECT
			t.id, t.session_id, t.transcript_path, t.prompt_line_offset, t.prompt_at,
			t.response_at,
			COALESCE(s.process_pid, 0), COALESCE(s.process_start, 0), COALESCE(s.tool, ''),
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
	defer rows.Close()

	var turns []danglingTurn
	for rows.Next() {
		var dt danglingTurn
		err := rows.Scan(
			&dt.id, &dt.sessionID, &dt.transcriptPath, &dt.promptLineOffset, &dt.promptAt,
			&dt.responseAt,
			&dt.processPID, &dt.processStart, &dt.tool,
			&dt.nextOffset, &dt.nextTranscriptPath, &dt.nextPromptAt,
		)
		if err != nil {
			continue
		}
		turns = append(turns, dt)
	}
	rows.Close()

	for _, dt := range turns {
		_ = reconcileTurn(conn, dt)
	}
}

func reconcileTurn(conn *sql.DB, dt danglingTurn) error {
	isLatest := dt.nextOffset == nil
	if isLatest && process.IsAlive(dt.processPID, dt.processStart) {
		if time.Since(dt.promptAt) <= 15*time.Minute {
			return nil
		}
	}

	to := -1
	if dt.nextOffset != nil && dt.nextTranscriptPath != nil && *dt.nextTranscriptPath == dt.transcriptPath {
		to = *dt.nextOffset
	}

	tool := dt.tool
	if tool == "" {
		tool = "claude-code"
	}
	p, ok := transcript.GetProvider(tool)
	if !ok {
		return fmt.Errorf("reconcile: unknown provider for tool %q", tool)
	}

	result, err := p.ExtractWindow(dt.transcriptPath, dt.promptLineOffset, to)
	if err != nil {
		return err
	}
	if result.InputTokens() == 0 && result.OutputTokens() == 0 && tool != "antigravity" {
		return nil
	}

	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete old turn model usages
	if _, err = tx.Exec("DELETE FROM turn_model_usages WHERE turn_id=?", dt.id); err != nil {
		return err
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
			return err
		}
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
			if err != nil || info.ModTime().IsZero() || info.ModTime().Unix() == 0 {
				responseAt = time.Now().Add(-time.Millisecond)
			} else {
				responseAt = info.ModTime()
			}
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
		return err
	}

	if result.Model() != "" {
		_, err = tx.Exec(
			"UPDATE sessions SET model=? WHERE id=? AND (model IS NULL OR model='')",
			result.Model(),
			dt.sessionID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func repairSessions(db *sql.DB) {
	rows, err := db.Query(`
		SELECT id, COALESCE(tool, ''), COALESCE(model, ''), COALESCE(project, '') FROM sessions
		WHERE project IS NULL OR project = '' OR model IS NULL OR model = ''
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	type sessInfo struct {
		id      string
		tool    string
		model   string
		project string
	}
	var sessList []sessInfo
	for rows.Next() {
		var s sessInfo
		if err := rows.Scan(&s.id, &s.tool, &s.model, &s.project); err == nil {
			sessList = append(sessList, s)
		}
	}
	rows.Close()

	if len(sessList) == 0 {
		return
	}

	type updateInfo struct {
		id      string
		project string
		model   string
	}
	var updates []updateInfo

	for _, s := range sessList {
		pathToRead, found := findExistingTranscriptPath(db, s.id)
		if !found {
			continue
		}

		newProject := s.project
		newModel := s.model

		if s.project == "" {
			if homeDir, err := os.UserHomeDir(); err == nil {
				newProject = resolveProjectPath(pathToRead, homeDir)
			}
		}

		if s.model == "" {
			newModel = resolveModel(pathToRead, s.tool)
		}

		if newProject != s.project || newModel != s.model {
			updates = append(updates, updateInfo{
				id:      s.id,
				project: newProject,
				model:   newModel,
			})
		}
	}

	if len(updates) == 0 {
		return
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	for _, up := range updates {
		_, err = tx.Exec(`
			UPDATE sessions
			SET project = ?, model = ?
			WHERE id = ?
		`, up.project, up.model, up.id)
		if err != nil {
			return
		}
	}

	_ = tx.Commit()
}

func resolveProjectPath(path string, homeDir string) string {
	f, err := os.Open(path)
	if err != nil {
		if wd, err := os.Getwd(); err == nil {
			return wd
		}
		return ""
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)

	for sc.Scan() {
		for _, p := range extractPathsFromLine(sc.Text(), homeDir) {
			if !isPathExcluded(p) {
				if root, ok := findProjectRoot(p); ok {
					return root
				}
			}
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return ""
}

func findExistingTranscriptPath(db *sql.DB, sessionID string) (string, bool) {
	rows, err := db.Query(`
		SELECT transcript_path FROM turns
		WHERE session_id = ? AND transcript_path IS NOT NULL AND transcript_path != ''
		ORDER BY id ASC
	`, sessionID)
	if err != nil {
		return "", false
	}
	defer rows.Close()

	for rows.Next() {
		var rawPath string
		if err := rows.Scan(&rawPath); err != nil {
			continue
		}

		resolvedPath := expandHomePath(rawPath)
		fullPath := strings.Replace(resolvedPath, "transcript.jsonl", "transcript_full.jsonl", 1)
		for _, p := range []string{fullPath, resolvedPath} {
			if _, err := os.Stat(p); err == nil {
				return p, true
			}
		}
	}
	return "", false
}

func expandHomePath(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func extractPathsFromLine(line string, homeDir string) []string {
	var paths []string
	idx := 0
	for {
		i := strings.Index(line[idx:], homeDir)
		if i == -1 {
			break
		}
		start := idx + i
		end := start + len(homeDir)
		for end < len(line) && !strings.ContainsRune(" '\"`\\,}]\n\r\t", rune(line[end])) {
			end++
		}
		paths = append(paths, line[start:end])
		idx = end
	}
	return paths
}

func isPathExcluded(path string) bool {
	excludes := []string{
		".gemini",
		".claude",
		".copilot",
		"Library",
		"Downloads",
		"Desktop",
		"Applications",
	}
	for _, excl := range excludes {
		if strings.Contains(path, excl) {
			return true
		}
	}
	return false
}

func findProjectRoot(path string) (string, bool) {
	dir := filepath.Clean(path)
	info, err := os.Stat(dir)
	if err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func resolveModel(path string, tool string) string {
	if tool == "antigravity" {
		if res, err := transcript.ParseAntigravityLog(path); err == nil && res.Model() != "" {
			return res.Model()
		}
	} else if res, err := transcript.ExtractWindow(path, 0, -1); err == nil && res.Model() != "" && res.Model() != "unknown" {
		return res.Model()
	}

	// Fallback to settings.json
	if home, err := os.UserHomeDir(); err == nil {
		for _, name := range []string{"antigravity-cli", "antigravity"} {
			p := filepath.Join(home, ".gemini", name, "settings.json")
			if data, err := os.ReadFile(p); err == nil {
				var cfg struct {
					Model string `json:"model"`
				}
				if err := json.Unmarshal(data, &cfg); err == nil && cfg.Model != "" {
					return cleanModelName(cfg.Model)
				}
			}
		}
	}
	return "gemini-3.5-flash"
}

func cleanModelName(name string) string {
	name = strings.ToLower(name)
	if i := strings.Index(name, "("); i >= 0 {
		name = name[:i]
	}
	name = strings.TrimSpace(name)
	var result []rune
	lastIsDash := false
	for _, r := range name {
		if r == ' ' || r == '-' {
			if !lastIsDash {
				result = append(result, '-')
				lastIsDash = true
			}
		} else if r == '.' {
			result = append(result, r)
			lastIsDash = false
		} else if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result = append(result, r)
			lastIsDash = false
		}
	}
	name = string(result)
	name = strings.Trim(name, "-")
	if name == "" {
		return "gemini-3.5-flash"
	}
	return name
}

func gitBranch(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}




