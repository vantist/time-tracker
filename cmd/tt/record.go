package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/user/tt/internal/db"
	"github.com/user/tt/internal/process"
	"github.com/user/tt/internal/recorder"
	"github.com/user/tt/internal/transcript"
)

func init() {
	rootCmd.AddCommand(recordCmd)
	recordCmd.AddCommand(recordPromptCmd, recordResponseCmd)

	recordPromptCmd.Flags().String("session", "", "session ID (overrides stdin)")
	recordPromptCmd.Flags().String("project", "", "project path (overrides stdin)")
	recordPromptCmd.Flags().String("tool", "claude-code", "tool name")
	recordPromptCmd.Flags().String("model", "", "model name (overrides stdin)")
	recordPromptCmd.Flags().String("transcript-path", "", "transcript JSONL path (overrides stdin)")

	recordResponseCmd.Flags().String("session", "", "session ID (overrides stdin)")
	recordResponseCmd.Flags().String("tokens", "", "tokens JSON string (overrides stdin)")
	recordResponseCmd.Flags().String("tool", "claude-code", "tool name")
}

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "Record AI tool events (called by hooks)",
}

var recordPromptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Record a user prompt event",
	RunE: func(cmd *cobra.Command, args []string) error {
		input, err := resolvePromptInput(cmd)
		if err != nil {
			return err
		}

		conn, err := db.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "tt: db open error: %v\n", err)
			return nil
		}
		defer conn.Close()

		return recorder.RecordPromptSilent(conn, input)
	},
}

var recordResponseCmd = &cobra.Command{
	Use:   "response",
	Short: "Record a response/stop event",
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := db.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "tt: db open error: %v\n", err)
			return nil
		}
		defer conn.Close()

		sessionID, tokensJSON, model, err := resolveResponseInput(cmd, conn)
		if err != nil {
			return err
		}

		return recorder.RecordResponseSilent(conn, sessionID, tokensJSON, model)
	},
}


// hookPayload covers both Claude Code and Copilot CLI stdin formats.
type hookPayload struct {
	// Claude Code fields
	SessionID      string `json:"session_id"`
	Cwd            string `json:"cwd"`
	Model          string `json:"model"`
	TranscriptPath string `json:"transcript_path"`
	// Copilot CLI fields
	CopilotSessionID string `json:"sessionId"`
}

func readStdinJSON() (*hookPayload, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, nil // interactive terminal, no stdin
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil || len(data) == 0 {
		return nil, err
	}
	var p hookPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, nil // malformed, ignore
	}
	// normalise Copilot sessionId → session_id
	if p.SessionID == "" && p.CopilotSessionID != "" {
		p.SessionID = p.CopilotSessionID
	}
	return &p, nil
}

// resolvePromptInputFromEnv resolves ProcessPID and ProcessStart.
// Priority: PROCESS_PID + PROCESS_START env vars (both non-empty) → override.
// Otherwise: os.Getppid() + process.StartTime(ppid).
func resolvePromptInputFromEnv() (recorder.PromptInput, error) {
	pidEnv := os.Getenv("PROCESS_PID")
	startEnv := os.Getenv("PROCESS_START")

	if pidEnv != "" && startEnv != "" {
		pid, pidErr := strconv.ParseInt(pidEnv, 10, 64)
		start, startErr := strconv.ParseInt(startEnv, 10, 64)
		if pidErr == nil && startErr == nil && start != 0 {
			return recorder.PromptInput{ProcessPID: pid, ProcessStart: start}, nil
		}
		if pidErr != nil {
			fmt.Fprintln(os.Stderr, "tt: PROCESS_PID invalid, session key may be unstable")
		} else {
			fmt.Fprintln(os.Stderr, "tt: PROCESS_START empty or invalid, session key may be unstable")
		}
		// fall through to ppid + process.StartTime
	}

	ppid := os.Getppid()
	start, err := process.StartTime(ppid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tt: process.StartTime: %v, session key may be unstable\n", err)
	}
	return recorder.PromptInput{
		ProcessPID:   int64(ppid),
		ProcessStart: start,
	}, nil
}

func resolvePromptInput(cmd *cobra.Command) (recorder.PromptInput, error) {
	stdin, _ := readStdinJSON()

	sessionID, _ := cmd.Flags().GetString("session")
	project, _ := cmd.Flags().GetString("project")
	tool, _ := cmd.Flags().GetString("tool")
	model, _ := cmd.Flags().GetString("model")
	transcriptPath, _ := cmd.Flags().GetString("transcript-path")

	if stdin != nil {
		if sessionID == "" {
			sessionID = stdin.SessionID
		}
		if project == "" {
			project = stdin.Cwd
		}
		if model == "" {
			model = stdin.Model
		}
		if transcriptPath == "" {
			transcriptPath = stdin.TranscriptPath
		}
	}

	envInput, _ := resolvePromptInputFromEnv()

	return recorder.PromptInput{
		SessionID:      sessionID,
		Project:        project,
		Tool:           tool,
		Model:          model,
		ProcessPID:     envInput.ProcessPID,
		ProcessStart:   envInput.ProcessStart,
		TranscriptPath: transcriptPath,
	}, nil
}

func resolveResponseInput(cmd *cobra.Command, conn *sql.DB) (sessionID, tokensJSON, model string, err error) {
	stdin, _ := readStdinJSON()

	sessionID, _ = cmd.Flags().GetString("session")
	tokensJSON, _ = cmd.Flags().GetString("tokens")

	if stdin != nil {
		if sessionID == "" {
			sessionID = stdin.SessionID
		}
	}

	// If tokensJSON was not provided via flag, extract from transcript.
	if tokensJSON == "" {
		tool, _ := cmd.Flags().GetString("tool")
		var stableID, dbTool string
		conn.QueryRow("SELECT id, tool FROM sessions WHERE id=?", sessionID).Scan(&stableID, &dbTool)
		if stableID == "" {
			conn.QueryRow("SELECT id, tool FROM sessions WHERE conversation_id=?", sessionID).Scan(&stableID, &dbTool)
		}
		if (tool == "" || tool == "claude-code") && dbTool != "" {
			tool = dbTool
		}

		if tool == "copilot-cli" || tool == "antigravity" {
			var res transcript.WindowResult
			var err error
			if tool == "copilot-cli" {
				path := filepath.Join("~", ".copilot", "session-state", sessionID, "events.jsonl")
				res, err = transcript.ParseCopilotLog(path)
			} else {
				path := filepath.Join("~", ".gemini", "antigravity", "brain", sessionID, ".system_generated", "logs", "transcript.jsonl")
				res, err = transcript.ParseAntigravityLog(path)
			}
			if err == nil {
				tokensJSON = marshalWindowResult(res)
				model = res.Model()
			} else {
				fmt.Fprintf(os.Stderr, "tt: failed to parse %s log: %v\n", tool, err)
			}
		} else {
			transcriptPath := ""
			if stdin != nil {
				transcriptPath = stdin.TranscriptPath
			}
			tokensJSON, model = resolveTokensFromTranscript(conn, sessionID, transcriptPath)
		}
	}
	return sessionID, tokensJSON, model, nil
}

// resolveTokensFromTranscript selects the extraction strategy based on whether
// a stored prompt_line_offset exists for the latest turn of this session.
// Falls back to full-transcript extraction when offset is absent.
func resolveTokensFromTranscript(conn *sql.DB, sessionID, transcriptPath string) (tokensJSON, model string) {
	if transcriptPath == "" {
		return "", ""
	}

	// Resolve stable session ID (same logic as RecordResponse).
	var stableID string
	conn.QueryRow("SELECT id FROM sessions WHERE id=?", sessionID).Scan(&stableID)
	if stableID == "" {
		conn.QueryRow("SELECT id FROM sessions WHERE conversation_id=?", sessionID).Scan(&stableID)
	}
	if stableID == "" {
		stableID = sessionID
	}

	var storedPath string
	var offset *int
	conn.QueryRow(
		"SELECT transcript_path, prompt_line_offset FROM turns WHERE session_id=? ORDER BY id DESC LIMIT 1",
		stableID,
	).Scan(&storedPath, &offset)

	if offset != nil && storedPath != "" {
		return extractFromTranscriptAtOffset(storedPath, *offset)
	}
	return extractFromTranscript(transcriptPath)
}

// extractFromTranscriptAtOffset extracts token usage from offset to EOF.
func extractFromTranscriptAtOffset(path string, offset int) (tokensJSON, model string) {
	result, err := transcript.ExtractWindow(path, offset, -1)
	if err != nil || (result.InputTokens() == 0 && result.OutputTokens() == 0) {
		return "", result.Model()
	}
	return marshalWindowResult(result), result.Model()
}

// extractFromTranscript extracts token usage for the last user turn.
func extractFromTranscript(path string) (tokensJSON, model string) {
	result, err := transcript.ExtractLastTurn(path)
	if err != nil || (result.InputTokens() == 0 && result.OutputTokens() == 0) {
		return "", result.Model()
	}
	return marshalWindowResult(result), result.Model()
}

// marshalWindowResult converts a WindowResult to the JSON string format expected by RecordResponse.
func marshalWindowResult(r transcript.WindowResult) string {
	out, err := json.Marshal(map[string]int{
		"input_tokens":              r.InputTokens(),
		"output_tokens":             r.OutputTokens(),
		"cache_read_tokens":         r.CacheReadTokens(),
		"cache_creation_tokens":     r.CacheCreationTokens(),
		"cache_creation_5m_tokens":  r.CacheCreate5m(),
		"cache_creation_1h_tokens":  r.CacheCreate1h(),
	})
	if err != nil {
		return ""
	}
	return string(out)
}
