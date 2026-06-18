package main

import (
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
)

func init() {
	rootCmd.AddCommand(recordCmd)
	recordCmd.AddCommand(recordPromptCmd, recordResponseCmd)

	recordPromptCmd.Flags().String("session", "", "session ID (overrides stdin)")
	recordPromptCmd.Flags().String("project", "", "project path (overrides stdin)")
	recordPromptCmd.Flags().String("tool", "claude-code", "tool name")
	recordPromptCmd.Flags().String("model", "", "model name (overrides stdin)")

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
		sessionID, tokensJSON, model, err := resolveResponseInput(cmd)
		if err != nil {
			return err
		}

		conn, err := db.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "tt: db open error: %v\n", err)
			return nil
		}
		defer conn.Close()

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
		var out recorder.PromptInput
		if pid, err := strconv.ParseInt(pidEnv, 10, 64); err == nil {
			out.ProcessPID = pid
		}
		if start, err := strconv.ParseInt(startEnv, 10, 64); err != nil || start == 0 {
			fmt.Fprintln(os.Stderr, "tt: PROCESS_START empty or invalid, session key may be unstable")
		} else {
			out.ProcessStart = start
		}
		return out, nil
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
	}

	envInput, _ := resolvePromptInputFromEnv()

	return recorder.PromptInput{
		SessionID:    sessionID,
		Project:      project,
		Tool:         tool,
		Model:        model,
		ProcessPID:   envInput.ProcessPID,
		ProcessStart: envInput.ProcessStart,
	}, nil
}

func resolveResponseInput(cmd *cobra.Command) (sessionID, tokensJSON, model string, err error) {
	stdin, _ := readStdinJSON()

	sessionID, _ = cmd.Flags().GetString("session")
	tokensJSON, _ = cmd.Flags().GetString("tokens")

	if stdin != nil {
		if sessionID == "" {
			sessionID = stdin.SessionID
		}
		if tokensJSON == "" && stdin.TranscriptPath != "" {
			tokensJSON, model = extractFromTranscript(stdin.TranscriptPath)
		}
	}
	return sessionID, tokensJSON, model, nil
}

// extractFromTranscript reads the transcript JSONL and returns the summed
// token usage across all API calls in the last user turn as a flat JSON string,
// plus the model from the last non-sidechain assistant entry.
//
// Claude Code writes multiple assistant entries per API call (one per content
// block: thinking/text/tool_use), all sharing identical usage stats. We
// deduplicate by (input, output, cache_read, cache_creation) before summing
// so each API call is counted exactly once.
func extractFromTranscript(path string) (tokensJSON, model string) {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	type usageFields struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	}
	type transcriptEntry struct {
		Type        string `json:"type"`
		IsSidechain bool   `json:"isSidechain"`
		Message     struct {
			Model string      `json:"model"`
			Usage usageFields `json:"usage"`
		} `json:"message"`
	}

	// Collect all entries, then work backwards from the end.
	var all []transcriptEntry
	dec := json.NewDecoder(f)
	for dec.More() {
		var entry transcriptEntry
		if err := dec.Decode(&entry); err != nil {
			continue
		}
		all = append(all, entry)
	}

	// Find the index of the last non-sidechain user entry; assistant entries
	// after it belong to the current turn.
	lastUserIdx := -1
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].Type == "user" && !all[i].IsSidechain {
			lastUserIdx = i
			break
		}
	}

	// Extract model from the last non-sidechain assistant entry (search whole
	// transcript — if Stop fires after /clear, lastUserIdx may be at the end
	// leaving no assistant entries in range).
	for i := len(all) - 1; i >= 0; i-- {
		e := all[i]
		if e.Type == "assistant" && !e.IsSidechain && e.Message.Model != "" {
			model = e.Message.Model
			break
		}
	}

	// Deduplicate assistant entries by usage tuple (same tuple = same API call),
	// then sum across unique API calls.
	type usageKey struct{ in, out, read, create int }
	seen := make(map[usageKey]bool)
	var total usageFields
	for i := lastUserIdx + 1; i < len(all); i++ {
		e := all[i]
		if e.Type != "assistant" || e.IsSidechain {
			continue
		}
		u := e.Message.Usage
		key := usageKey{u.InputTokens, u.OutputTokens, u.CacheReadInputTokens, u.CacheCreationInputTokens}
		if seen[key] {
			continue
		}
		seen[key] = true
		total.InputTokens += u.InputTokens
		total.OutputTokens += u.OutputTokens
		total.CacheReadInputTokens += u.CacheReadInputTokens
		total.CacheCreationInputTokens += u.CacheCreationInputTokens
	}

	if total.InputTokens == 0 && total.OutputTokens == 0 {
		// /clear race: lastUserIdx is the /clear entry; no assistant entries follow it yet.
		// Fall back to the previous turn window [prevUserIdx+1, lastUserIdx).
		if lastUserIdx > 0 {
			prevUserIdx := -1
			for i := lastUserIdx - 1; i >= 0; i-- {
				if all[i].Type == "user" && !all[i].IsSidechain {
					prevUserIdx = i
					break
				}
			}
			seen = make(map[usageKey]bool)
			for i := prevUserIdx + 1; i < lastUserIdx; i++ {
				e := all[i]
				if e.Type != "assistant" || e.IsSidechain {
					continue
				}
				u := e.Message.Usage
				key := usageKey{u.InputTokens, u.OutputTokens, u.CacheReadInputTokens, u.CacheCreationInputTokens}
				if seen[key] {
					continue
				}
				seen[key] = true
				total.InputTokens += u.InputTokens
				total.OutputTokens += u.OutputTokens
				total.CacheReadInputTokens += u.CacheReadInputTokens
				total.CacheCreationInputTokens += u.CacheCreationInputTokens
			}
		}
		if total.InputTokens == 0 && total.OutputTokens == 0 {
			return "", model
		}
	}

	out, err := json.Marshal(map[string]int{
		"input_tokens":          total.InputTokens,
		"output_tokens":         total.OutputTokens,
		"cache_read_tokens":     total.CacheReadInputTokens,
		"cache_creation_tokens": total.CacheCreationInputTokens,
	})
	if err != nil {
		return "", model
	}
	return string(out), model
}
