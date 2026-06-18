package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

// transcriptUsageFields mirrors the usage fields in a Claude Code transcript entry.
type transcriptUsageFields struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// contentBlock is one element in a transcript entry's content array.
type contentBlock struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// transcriptEntry is one JSONL line in a Claude Code transcript file.
type transcriptEntry struct {
	Type        string `json:"type"`
	IsSidechain bool   `json:"isSidechain"`
	Message     struct {
		Model string                `json:"model"`
		Usage transcriptUsageFields `json:"usage"`
	} `json:"message"`
	Content []contentBlock `json:"content"`
}

// subagentMeta is the content of agent-<id>.meta.json in the subagents directory.
type subagentMeta struct {
	ToolUseID   string `json:"toolUseId"`
	AgentType   string `json:"agentType"`
	Description string `json:"description"`
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
		transcriptPath := ""
		if stdin != nil {
			transcriptPath = stdin.TranscriptPath
		}
		tokensJSON, model = resolveTokensFromTranscript(conn, sessionID, transcriptPath)
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
	tokensJSON, model, _ = transcript.ExtractWindow(path, offset, -1)
	return
}

// loadTranscript opens and decodes all entries from a JSONL transcript file.
// Returns nil on error. Expands a leading "~/" to the user home directory.
func loadTranscript(path string) []transcriptEntry {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var all []transcriptEntry
	dec := json.NewDecoder(f)
	for dec.More() {
		var entry transcriptEntry
		if err := dec.Decode(&entry); err != nil {
			continue
		}
		all = append(all, entry)
	}
	return all
}

// sumWindow deduplicates assistant entries by usage tuple in all[from:to] and
// returns their sum. Each unique (input, output, cacheRead, cacheCreate) tuple
// represents one API call; duplicate content blocks are skipped.
func sumWindow(all []transcriptEntry, from, to int) transcriptUsageFields {
	type usageKey struct{ in, out, read, create int }
	seen := make(map[usageKey]bool)
	var acc transcriptUsageFields
	for i := from; i < to; i++ {
		e := all[i]
		if e.Type != "assistant" || e.IsSidechain {
			continue
		}
		u := e.Message.Usage
		k := usageKey{u.InputTokens, u.OutputTokens, u.CacheReadInputTokens, u.CacheCreationInputTokens}
		if seen[k] {
			continue
		}
		seen[k] = true
		acc.InputTokens += u.InputTokens
		acc.OutputTokens += u.OutputTokens
		acc.CacheReadInputTokens += u.CacheReadInputTokens
		acc.CacheCreationInputTokens += u.CacheCreationInputTokens
	}
	return acc
}

// extractSubagentTokens scans entries[offset:] for Agent tool_use IDs, then reads
// the corresponding subagent jsonl files and sums their token usage.
// All errors are silently ignored; zero is returned when nothing can be read.
func extractSubagentTokens(transcriptPath string, entries []transcriptEntry, offset int) transcriptUsageFields {
	// Collect Agent tool_use IDs from entries at or after offset.
	agentIDs := make(map[string]bool)
	for i := offset; i < len(entries); i++ {
		e := entries[i]
		if e.Type != "assistant" {
			continue
		}
		for _, blk := range e.Content {
			if blk.Type == "tool_use" && blk.Name == "Agent" && blk.ID != "" {
				agentIDs[blk.ID] = true
			}
		}
	}
	if len(agentIDs) == 0 {
		return transcriptUsageFields{}
	}

	subagentsDir := filepath.Join(strings.TrimSuffix(transcriptPath, ".jsonl"), "subagents")
	metas, err := filepath.Glob(filepath.Join(subagentsDir, "*.meta.json"))
	if err != nil || len(metas) == 0 {
		return transcriptUsageFields{}
	}

	var acc transcriptUsageFields
	for _, metaPath := range metas {
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta subagentMeta
		if err := json.Unmarshal(data, &meta); err != nil || !agentIDs[meta.ToolUseID] {
			continue
		}
		// Derive agent jsonl path from meta path.
		base := strings.TrimSuffix(metaPath, ".meta.json")
		agentEntries := loadTranscript(base + ".jsonl")
		sub := sumSubagentWindow(agentEntries)
		acc.InputTokens += sub.InputTokens
		acc.OutputTokens += sub.OutputTokens
		acc.CacheReadInputTokens += sub.CacheReadInputTokens
		acc.CacheCreationInputTokens += sub.CacheCreationInputTokens
	}
	return acc
}

// sumSubagentWindow deduplicates and sums all assistant entries in a subagent transcript.
// Unlike sumWindow, it does not filter out isSidechain entries — subagent entries are all sidechain.
func sumSubagentWindow(entries []transcriptEntry) transcriptUsageFields {
	type usageKey struct{ in, out, read, create int }
	seen := make(map[usageKey]bool)
	var acc transcriptUsageFields
	for _, e := range entries {
		if e.Type != "assistant" {
			continue
		}
		u := e.Message.Usage
		k := usageKey{u.InputTokens, u.OutputTokens, u.CacheReadInputTokens, u.CacheCreationInputTokens}
		if seen[k] {
			continue
		}
		seen[k] = true
		acc.InputTokens += u.InputTokens
		acc.OutputTokens += u.OutputTokens
		acc.CacheReadInputTokens += u.CacheReadInputTokens
		acc.CacheCreationInputTokens += u.CacheCreationInputTokens
	}
	return acc
}

// extractFromTranscript returns summed token usage for the last user turn plus
// the model from the last non-sidechain assistant entry.
//
// Claude Code writes multiple assistant entries per API call (one per content
// block: thinking/text/tool_use), all sharing identical usage stats. We
// deduplicate by (input, output, cache_read, cache_creation) before summing
// so each API call is counted exactly once.
func extractFromTranscript(path string) (tokensJSON, model string) {
	all := loadTranscript(path)
	if len(all) == 0 {
		return "", ""
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

	total := sumWindow(all, lastUserIdx+1, len(all))

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
			total = sumWindow(all, prevUserIdx+1, lastUserIdx)
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
