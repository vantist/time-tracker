package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

var binPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "tt-test-*")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}

	binPath = filepath.Join(tmpDir, "tt")

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("failed to compile tt binary: %v\nOutput: %s", err, string(output))
	}

	code := m.Run()

	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func TestIntegration_BinaryExists(t *testing.T) {
	if binPath == "" {
		t.Fatal("binPath is not set")
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("compiled binary does not exist at %s: %v", binPath, err)
	}
}

func runTT(t *testing.T, home, dbPath, stdin string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), "HOME="+home, "TT_DB_PATH="+dbPath)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func TestIntegration_RunTTHelper(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	stdout, stderr, err := runTT(t, home, dbPath, "", "version")
	if err != nil {
		t.Fatalf("runTT failed: %v, stderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "dev") {
		t.Errorf("expected version output 'dev', got: %s", stdout)
	}
}

type dbSession struct {
	ID        string
	Project   string
	Tool      string
	Model     string
	Branch    *string
	WorkItem  *string
	StartedAt string
	EndedAt   *string
}

type dbTurn struct {
	ID                  int64
	SessionID           string
	PromptAt            string
	ResponseAt          *string
	InputTokens         *int64
	OutputTokens        *int64
	CacheReadTokens     *int64
	CacheCreationTokens *int64
	EstimatedCostUSD    *float64
}

type dbTurnModelUsage struct {
	ID                  int64
	TurnID              int64
	Model               string
	IsSubagent          bool
	InputTokens         int64
	OutputTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	EstimatedCostUSD    float64
}

func openDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()
	dbConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database %s: %v", dbPath, err)
	}
	return dbConn
}

func getSession(t *testing.T, dbPath, sessionID string) *dbSession {
	t.Helper()
	dbConn := openDB(t, dbPath)
	defer dbConn.Close()

	var s dbSession
	err := dbConn.QueryRow("SELECT id, project, tool, model, branch, work_item, started_at, ended_at FROM sessions WHERE id = ?", sessionID).
		Scan(&s.ID, &s.Project, &s.Tool, &s.Model, &s.Branch, &s.WorkItem, &s.StartedAt, &s.EndedAt)
	if err != nil {
		t.Fatalf("failed to get session %s: %v", sessionID, err)
	}
	return &s
}

func getTurns(t *testing.T, dbPath, sessionID string) []dbTurn {
	t.Helper()
	dbConn := openDB(t, dbPath)
	defer dbConn.Close()

	rows, err := dbConn.Query("SELECT id, session_id, prompt_at, response_at, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, estimated_cost_usd FROM turns WHERE session_id = ? ORDER BY id ASC", sessionID)
	if err != nil {
		t.Fatalf("failed to query turns for %s: %v", sessionID, err)
	}
	defer rows.Close()

	var turns []dbTurn
	for rows.Next() {
		var r dbTurn
		err := rows.Scan(&r.ID, &r.SessionID, &r.PromptAt, &r.ResponseAt, &r.InputTokens, &r.OutputTokens, &r.CacheReadTokens, &r.CacheCreationTokens, &r.EstimatedCostUSD)
		if err != nil {
			t.Fatalf("failed to scan turn: %v", err)
		}
		turns = append(turns, r)
	}
	return turns
}

func getTurnModelUsages(t *testing.T, dbPath string, turnID int64) []dbTurnModelUsage {
	t.Helper()
	dbConn := openDB(t, dbPath)
	defer dbConn.Close()

	rows, err := dbConn.Query("SELECT id, turn_id, model, is_subagent, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, estimated_cost_usd FROM turn_model_usages WHERE turn_id = ? ORDER BY id ASC", turnID)
	if err != nil {
		t.Fatalf("failed to query turn model usages for turn %d: %v", turnID, err)
	}
	defer rows.Close()

	var usages []dbTurnModelUsage
	for rows.Next() {
		var u dbTurnModelUsage
		err := rows.Scan(&u.ID, &u.TurnID, &u.Model, &u.IsSubagent, &u.InputTokens, &u.OutputTokens, &u.CacheReadTokens, &u.CacheCreationTokens, &u.EstimatedCostUSD)
		if err != nil {
			t.Fatalf("failed to scan model usage: %v", err)
		}
		usages = append(usages, u)
	}
	return usages
}

func TestIntegration_DBAssertHelpers(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_assert.db")
	dbConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer dbConn.Close()

	_, err = dbConn.Exec(`
		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			project TEXT,
			tool TEXT,
			model TEXT,
			branch TEXT,
			work_item TEXT,
			started_at DATETIME,
			ended_at DATETIME
		);
		CREATE TABLE turns (
			id INTEGER PRIMARY KEY,
			session_id TEXT,
			prompt_at DATETIME,
			response_at DATETIME,
			input_tokens INTEGER,
			output_tokens INTEGER,
			cache_read_tokens INTEGER,
			cache_creation_tokens INTEGER,
			estimated_cost_usd REAL
		);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	_, err = dbConn.Exec(`
		INSERT INTO sessions (id, project, tool, model, branch, work_item, started_at)
		VALUES ('sess-1', '/proj', 'claude-code', 'claude-3-5', 'main', 'wi-1', '2026-06-22T00:00:00Z');
		INSERT INTO turns (id, session_id, prompt_at, response_at, input_tokens, output_tokens)
		VALUES (1, 'sess-1', '2026-06-22T00:00:05Z', '2026-06-22T00:00:10Z', 10, 20);
	`)
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	sess := getSession(t, dbPath, "sess-1")
	if sess.ID != "sess-1" {
		t.Errorf("expected session ID 'sess-1', got %q", sess.ID)
	}

	turns := getTurns(t, dbPath, "sess-1")
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
}

func initGitRepo(t *testing.T, dir, branch string) {
	t.Helper()
	runCmd := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to run git %v: %v, output: %s", args, err, string(out))
		}
	}
	runCmd("init")
	runCmd("config", "user.name", "Test User")
	runCmd("config", "user.email", "test@example.com")

	dummyFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(dummyFile, []byte("# Mock Repo"), 0644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}
	runCmd("add", "README.md")
	runCmd("commit", "-m", "initial commit")
	runCmd("checkout", "-B", branch)
}

func TestIntegration_GitBranchRepair(t *testing.T) {
	projDir := t.TempDir()
	initGitRepo(t, projDir, "feature-abc")

	home := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	_, _, err := runTT(t, home, dbPath, "", "report")
	if err != nil {
		t.Fatalf("failed to initialize db: %v", err)
	}

	dbConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer dbConn.Close()

	_, err = dbConn.Exec(`
		INSERT INTO sessions (id, project, tool, model, started_at, branch)
		VALUES ('sess-git-repair', ?, 'claude-code', 'claude-3-5', '2026-06-22T00:00:00Z', '')
	`, projDir)
	if err != nil {
		t.Fatalf("failed to insert session: %v", err)
	}

	_, _, err = runTT(t, home, dbPath, "", "report")
	if err != nil {
		t.Logf("report run finished (might be empty): %v", err)
	}

	sess := getSession(t, dbPath, "sess-git-repair")

	if sess.Branch == nil {
		t.Fatal("session branch is nil")
	}
	if *sess.Branch != "feature-abc" {
		t.Fatalf("expected branch %q, got %q", "feature-abc", *sess.Branch)
	}
}

func TestIntegration_ActiveTurnPreemption(t *testing.T) {
	projDir := t.TempDir()
	home := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	_, _, err := runTT(t, home, dbPath, "", "record", "prompt",
		"--session", "sess-preempt",
		"--tool", "antigravity",
		"--model", "gemini-3.5-flash",
		"--project", projDir,
	)
	if err != nil {
		t.Fatalf("first record prompt failed: %v", err)
	}

	dbConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	oldTime := time.Now().UTC().Add(-20 * time.Minute).Format(time.RFC3339)
	_, err = dbConn.Exec("UPDATE turns SET prompt_at = ? WHERE session_id = 'sess-preempt'", oldTime)
	if err != nil {
		t.Fatalf("failed to age turn: %v", err)
	}
	dbConn.Close()

	_, _, err = runTT(t, home, dbPath, "", "record", "prompt",
		"--session", "sess-preempt",
		"--tool", "antigravity",
		"--model", "gemini-3.5-flash",
		"--project", projDir,
	)
	if err != nil {
		t.Fatalf("second record prompt failed: %v", err)
	}

	turns := getTurns(t, dbPath, "sess-preempt")
	if len(turns) != 2 {
		t.Fatalf("expected 2 total turns, got %d", len(turns))
	}

	if turns[0].ResponseAt == nil {
		t.Fatal("expected first turn response_at to be non-nil (preempted)")
	}
}

func TestIntegration_IdleThresholdReconcile(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	_, _, err := runTT(t, home, dbPath, "", "report")
	if err != nil {
		t.Fatalf("failed to initialize db: %v", err)
	}

	transDir := t.TempDir()
	transPath := filepath.Join(transDir, "transcript.jsonl")
	content := `{"type":"user","isSidechain":false}
{"type":"assistant","isSidechain":false,"message":{"model":"claude-3-5-sonnet","usage":{"input_tokens":100,"output_tokens":50}}}
`
	if err := os.WriteFile(transPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write transcript: %v", err)
	}

	dbConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer dbConn.Close()

	parentPID := os.Getpid()

	_, err = dbConn.Exec(`
		INSERT INTO sessions (id, project, tool, model, started_at, process_pid, process_start)
		VALUES ('sess-idle-recent', '/proj', 'claude-code', 'claude-3-5', '2026-06-22T00:00:00Z', ?, 0)
	`, parentPID)
	if err != nil {
		t.Fatalf("failed to insert session recent: %v", err)
	}

	timeRecent := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
	_, err = dbConn.Exec(`
		INSERT INTO turns (session_id, prompt_at, transcript_path, prompt_line_offset)
		VALUES ('sess-idle-recent', ?, ?, 0)
	`, timeRecent, transPath)
	if err != nil {
		t.Fatalf("failed to insert recent turn: %v", err)
	}

	_, err = dbConn.Exec(`
		INSERT INTO sessions (id, project, tool, model, started_at, process_pid, process_start)
		VALUES ('sess-idle-old', '/proj', 'claude-code', 'claude-3-5', '2026-06-22T00:00:00Z', ?, 0)
	`, parentPID)
	if err != nil {
		t.Fatalf("failed to insert session old: %v", err)
	}

	timeOld := time.Now().UTC().Add(-20 * time.Minute).Format(time.RFC3339)
	_, err = dbConn.Exec(`
		INSERT INTO turns (session_id, prompt_at, transcript_path, prompt_line_offset)
		VALUES ('sess-idle-old', ?, ?, 0)
	`, timeOld, transPath)
	if err != nil {
		t.Fatalf("failed to insert old turn: %v", err)
	}

	dbConn.Close()

	_, _, err = runTT(t, home, dbPath, "", "report")
	if err != nil {
		t.Logf("report run finished: %v", err)
	}

	turnsRecent := getTurns(t, dbPath, "sess-idle-recent")
	if len(turnsRecent) != 1 {
		t.Fatalf("expected 1 recent turn, got %d", len(turnsRecent))
	}
	if turnsRecent[0].ResponseAt != nil {
		t.Errorf("expected recent active turn NOT to be reconciled, but got response_at: %s", *turnsRecent[0].ResponseAt)
	}

	turnsOld := getTurns(t, dbPath, "sess-idle-old")
	if len(turnsOld) != 1 {
		t.Fatalf("expected 1 old turn, got %d", len(turnsOld))
	}

	if turnsOld[0].ResponseAt == nil {
		t.Fatal("expected old active turn response_at to be non-nil (reconciled)")
	}
	if turnsOld[0].InputTokens == nil || *turnsOld[0].InputTokens != 100 {
		t.Errorf("expected input_tokens to be 100, got %v", turnsOld[0].InputTokens)
	}
	if turnsOld[0].OutputTokens == nil || *turnsOld[0].OutputTokens != 50 {
		t.Errorf("expected output_tokens to be 50, got %v", turnsOld[0].OutputTokens)
	}
}

func TestIntegration_FallbackDefaultModel(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	expectedWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	_, _, err = runTT(t, home, dbPath, "", "record", "prompt",
		"--session", "sess-fallback",
		"--tool", "antigravity",
	)
	if err != nil {
		t.Fatalf("record prompt failed: %v", err)
	}

	sess := getSession(t, dbPath, "sess-fallback")

	if sess.Project != expectedWD {
		t.Errorf("expected project %q, got %q", expectedWD, sess.Project)
	}

	if sess.Model != "gemini-3.5-flash" {
		t.Errorf("expected model %q, got %q", "gemini-3.5-flash", sess.Model)
	}
}

func TestIntegration_MultiToolIntegration(t *testing.T) {
	projDir := t.TempDir()
	home := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	_, _, err := runTT(t, home, dbPath, "", "report")
	if err != nil {
		t.Fatalf("failed to initialize db: %v", err)
	}

	testCases := []struct {
		name          string
		tool          string
		pid           string
		start         string
		sessID        string
		stdinPattern  string
		initialData   string
		finalData     string
		expectedModel string
		expectedInput int64
	}{
		{
			name:          "Claude Code",
			tool:          "claude-code",
			pid:           "10001",
			start:         "1700000001",
			sessID:        "sess-claude",
			stdinPattern:  `{"session_id":"%s","cwd":"%s","model":"claude-3-5-sonnet","transcript_path":"%s"}`,
			initialData:   `{"type":"user","isSidechain":false}` + "\n",
			finalData:     `{"type":"assistant","isSidechain":false,"message":{"model":"claude-3-5-sonnet","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":10,"cache_creation_input_tokens":5}}}` + "\n",
			expectedModel: "claude-3-5-sonnet",
			expectedInput: 100,
		},
		{
			name:          "Copilot CLI",
			tool:          "copilot-cli",
			pid:           "10002",
			start:         "1700000002",
			sessID:        "sess-copilot",
			stdinPattern:  `{"sessionId":"%s","cwd":"%s","model":"gpt-4o","transcriptPath":"%s"}`,
			initialData:   "",
			finalData:     `{"type":"session.shutdown","data":{"mainModel":"gpt-4o","modelMetrics":{"gpt-4o":{"usage":{"inputTokens":200,"outputTokens":100,"cacheReadTokens":50,"cacheWriteTokens":25}}}}}` + "\n",
			expectedModel: "gpt-4o",
			expectedInput: 200,
		},
		{
			name:          "Google Antigravity",
			tool:          "antigravity",
			pid:           "10003",
			start:         "1700000003",
			sessID:        "sess-antigravity",
			stdinPattern:  `{"conversationId":"%s","cwd":"%s","model":"gemini-1.5-pro","transcriptPath":"%s"}`,
			initialData:   `{"type":"user","isSidechain":false}` + "\n",
			finalData:     `{"type":"assistant","isSidechain":false,"message":{"model":"gemini-1.5-pro","usage":{"input_tokens":300,"output_tokens":150,"cache_read_input_tokens":80,"cache_creation_input_tokens":40}}}` + "\n",
			expectedModel: "gemini-1.5-pro",
			expectedInput: 300,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PROCESS_PID", tc.pid)
			t.Setenv("PROCESS_START", tc.start)

			transPath := filepath.Join(t.TempDir(), "transcript.jsonl")
			if err := os.WriteFile(transPath, []byte(tc.initialData), 0644); err != nil {
				t.Fatalf("failed to write transcript: %v", err)
			}

			stdin := fmt.Sprintf(tc.stdinPattern, tc.sessID, projDir, transPath)
			stdout, stderr, err := runTT(t, home, dbPath, stdin, "record", "prompt", "--tool", tc.tool)
			if err != nil {
				t.Fatalf("prompt record failed: %v, stdout: %s, stderr: %s", err, stdout, stderr)
			}
			if stderr != "" {
				t.Logf("prompt record stderr: %s", stderr)
			}

			f, err := os.OpenFile(transPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				t.Fatalf("failed to open transcript: %v", err)
			}
			if _, err := f.WriteString(tc.finalData); err != nil {
				t.Fatalf("failed to append response to transcript: %v", err)
			}
			f.Close()

			stdout, stderr, err = runTT(t, home, dbPath, stdin, "record", "response", "--tool", tc.tool)
			if err != nil {
				t.Fatalf("response record failed: %v, stdout: %s, stderr: %s", err, stdout, stderr)
			}
			if stderr != "" {
				t.Logf("response record stderr: %s", stderr)
			}

			turns := getTurns(t, dbPath, tc.sessID)
			if len(turns) != 1 {
				t.Fatalf("expected 1 turn, got %d", len(turns))
			}
			trn := turns[0]
			if trn.InputTokens == nil || *trn.InputTokens != tc.expectedInput {
				t.Errorf("expected input tokens %d, got %v", tc.expectedInput, trn.InputTokens)
			}

			usages := getTurnModelUsages(t, dbPath, trn.ID)
			if len(usages) != 1 {
				t.Fatalf("expected 1 model usage, got %d", len(usages))
			}
			if usages[0].Model != tc.expectedModel {
				t.Errorf("expected model %q, got %q", tc.expectedModel, usages[0].Model)
			}
		})
	}
}

// Integration test: tt setup --opencode produces valid TS file, and the
// opencode record flow (prompt + response with subagent-tokens) writes
// sessions / turns / turn_model_usages correctly.
func TestIntegration_OpenCode_FullFlow(t *testing.T) {
	home := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	// 1. tt setup --opencode creates plugin file
	stdout, stderr, err := runTT(t, home, dbPath, "", "setup", "--opencode")
	if err != nil {
		t.Fatalf("tt setup --opencode: %v, stderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "OpenCode plugin configured in") {
		t.Errorf("unexpected setup output: %s", stdout)
	}

	pluginPath := filepath.Join(home, ".config", "opencode", "plugins", "tt-bridge.ts")
	if _, err := os.Stat(pluginPath); err != nil {
		t.Fatalf("plugin file not created: %v", err)
	}

	// 2. tt record prompt --tool opencode creates session + turn
	_, stderr, err = runTT(t, home, dbPath, "", "record", "prompt",
		"--tool", "opencode", "--session", "sess-oc-int", "--project", "/repo", "--model", "")
	if err != nil {
		t.Fatalf("tt record prompt: %v, stderr: %s", err, stderr)
	}

	sess := getSession(t, dbPath, "sess-oc-int")
	if sess.Tool != "opencode" {
		t.Errorf("session tool = %q, want opencode", sess.Tool)
	}

	turns := getTurns(t, dbPath, "sess-oc-int")
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}

	// 3. tt record response with --subagent-tokens writes main + subagent usages
	mainTokens := `{"input_tokens":500,"output_tokens":100,"cache_read_tokens":0,"cache_creation_tokens":0}`
	subTokens := `[{"model":"claude-haiku","agent":"build","input_tokens":100,"output_tokens":50,"cache_read_tokens":20}]`
	_, stderr, err = runTT(t, home, dbPath, "", "record", "response",
		"--tool", "opencode", "--session", "sess-oc-int", "--model", "claude-sonnet-4-6",
		"--tokens", mainTokens, "--subagent-tokens", subTokens)
	if err != nil {
		t.Fatalf("tt record response: %v, stderr: %s", err, stderr)
	}

	// Verify turn tokens
	turns = getTurns(t, dbPath, "sess-oc-int")
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn after response, got %d", len(turns))
	}
	trn := turns[0]
	if trn.ResponseAt == nil {
		t.Error("response_at not set")
	}
	if trn.InputTokens == nil || *trn.InputTokens != 500 {
		t.Errorf("input_tokens = %v, want 500", trn.InputTokens)
	}
	if trn.OutputTokens == nil || *trn.OutputTokens != 100 {
		t.Errorf("output_tokens = %v, want 100", trn.OutputTokens)
	}

	// Verify turn_model_usages: 1 main (is_subagent=0) + 1 subagent (is_subagent=1)
	usages := getTurnModelUsages(t, dbPath, trn.ID)
	if len(usages) != 2 {
		t.Fatalf("expected 2 model usages, got %d", len(usages))
	}

	var mainUsage, subUsage *dbTurnModelUsage
	for i := range usages {
		if usages[i].IsSubagent {
			subUsage = &usages[i]
		} else {
			mainUsage = &usages[i]
		}
	}

	if mainUsage == nil {
		t.Fatal("no main agent usage found")
	}
	if mainUsage.Model != "claude-sonnet-4-6" {
		t.Errorf("main model = %q, want claude-sonnet-4-6", mainUsage.Model)
	}

	if subUsage == nil {
		t.Fatal("no subagent usage found")
	}
	if subUsage.Model != "claude-haiku" {
		t.Errorf("subagent model = %q, want claude-haiku", subUsage.Model)
	}
	if subUsage.InputTokens != 100 || subUsage.OutputTokens != 50 {
		t.Errorf("subagent tokens = in=%d out=%d, want 100/50", subUsage.InputTokens, subUsage.OutputTokens)
	}
}
