package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)


// WindowResult holds extracted token usage and model for a transcript window.
type WindowResult struct {
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int // total cache creation (5m + 1h)
	CacheCreate5m       int // ephemeral_5m_input_tokens
	CacheCreate1h       int // ephemeral_1h_input_tokens
	Model               string
}

type cacheCreationFields struct {
	Ephemeral5m int `json:"ephemeral_5m_input_tokens"`
	Ephemeral1h int `json:"ephemeral_1h_input_tokens"`
}

type usageFields struct {
	InputTokens              int                `json:"input_tokens"`
	OutputTokens             int                `json:"output_tokens"`
	CacheReadInputTokens     int                `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int                `json:"cache_creation_input_tokens"`
	CacheCreation            cacheCreationFields `json:"cache_creation"`
}

type contentBlock struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type entry struct {
	Type        string `json:"type"`
	IsSidechain bool   `json:"isSidechain"`
	Message     struct {
		Model   string         `json:"model"`
		Usage   usageFields    `json:"usage"`
		Content []contentBlock `json:"content"`
	} `json:"message"`
}

type subagentMeta struct {
	ToolUseID string `json:"toolUseId"`
}

// ExtractWindow extracts token usage and model from a transcript JSONL file
// for entries in the range [from, to). to=-1 means read to EOF.
// Returns error if the file cannot be opened.
func ExtractWindow(path string, from, to int) (WindowResult, error) {
	all, err := loadTranscript(path)
	if err != nil {
		return WindowResult{}, err
	}

	var result WindowResult

	// Model from last non-sidechain assistant entry (whole transcript).
	for i := len(all) - 1; i >= 0; i-- {
		e := all[i]
		if e.Type == "assistant" && !e.IsSidechain && e.Message.Model != "" {
			result.Model = e.Message.Model
			break
		}
	}

	end := len(all)
	if to != -1 && to < end {
		end = to
	}
	if from > end {
		from = end
	}

	acc := sumWindow(all, from, end)

	sub := extractSubagentTokens(path, all, from, end)
	acc.InputTokens += sub.InputTokens
	acc.OutputTokens += sub.OutputTokens
	acc.CacheReadInputTokens += sub.CacheReadInputTokens
	acc.CacheCreationInputTokens += sub.CacheCreationInputTokens
	acc.CacheCreation.Ephemeral5m += sub.CacheCreation.Ephemeral5m
	acc.CacheCreation.Ephemeral1h += sub.CacheCreation.Ephemeral1h

	result.InputTokens = acc.InputTokens
	result.OutputTokens = acc.OutputTokens
	result.CacheReadTokens = acc.CacheReadInputTokens
	result.CacheCreationTokens = acc.CacheCreationInputTokens
	result.CacheCreate5m = acc.CacheCreation.Ephemeral5m
	result.CacheCreate1h = acc.CacheCreation.Ephemeral1h

	return result, nil
}

// ExtractLastTurn extracts token usage for the last user turn in the transcript.
// Handles /clear race: if the last user entry has no following assistant entries,
// falls back to the previous turn window.
func ExtractLastTurn(path string) (WindowResult, error) {
	all, err := loadTranscript(path)
	if err != nil {
		return WindowResult{}, err
	}
	if len(all) == 0 {
		return WindowResult{}, nil
	}

	var result WindowResult

	// Model: search entire transcript for last non-sidechain assistant entry.
	for i := len(all) - 1; i >= 0; i-- {
		e := all[i]
		if e.Type == "assistant" && !e.IsSidechain && e.Message.Model != "" {
			result.Model = e.Message.Model
			break
		}
	}

	lastUserIdx := -1
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].Type == "user" && !all[i].IsSidechain {
			lastUserIdx = i
			break
		}
	}

	acc := sumWindow(all, lastUserIdx+1, len(all))

	if acc.InputTokens == 0 && acc.OutputTokens == 0 && lastUserIdx > 0 {
		// /clear race: fallback to previous turn window.
		prevUserIdx := -1
		for i := lastUserIdx - 1; i >= 0; i-- {
			if all[i].Type == "user" && !all[i].IsSidechain {
				prevUserIdx = i
				break
			}
		}
		acc = sumWindow(all, prevUserIdx+1, lastUserIdx)
	}

	sub := extractSubagentTokens(path, all, lastUserIdx+1, len(all))
	acc.InputTokens += sub.InputTokens
	acc.OutputTokens += sub.OutputTokens
	acc.CacheReadInputTokens += sub.CacheReadInputTokens
	acc.CacheCreationInputTokens += sub.CacheCreationInputTokens
	acc.CacheCreation.Ephemeral5m += sub.CacheCreation.Ephemeral5m
	acc.CacheCreation.Ephemeral1h += sub.CacheCreation.Ephemeral1h

	result.InputTokens = acc.InputTokens
	result.OutputTokens = acc.OutputTokens
	result.CacheReadTokens = acc.CacheReadInputTokens
	result.CacheCreationTokens = acc.CacheCreationInputTokens
	result.CacheCreate5m = acc.CacheCreation.Ephemeral5m
	result.CacheCreate1h = acc.CacheCreation.Ephemeral1h

	return result, nil
}

func loadTranscript(path string) ([]entry, error) {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var all []entry
	dec := json.NewDecoder(f)
	for dec.More() {
		var e entry
		if err := dec.Decode(&e); err != nil {
			continue
		}
		all = append(all, e)
	}
	return all, nil
}

func sumWindow(all []entry, from, to int) usageFields {
	type usageKey struct{ in, out, read, create int }
	seen := make(map[usageKey]bool)
	var acc usageFields
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
		acc.CacheCreation.Ephemeral5m += u.CacheCreation.Ephemeral5m
		acc.CacheCreation.Ephemeral1h += u.CacheCreation.Ephemeral1h
	}
	return acc
}

// extractSubagentTokens scans entries[from:min(to,len(entries))] for Agent tool_use IDs.
func extractSubagentTokens(transcriptPath string, entries []entry, from, to int) usageFields {
	if to > len(entries) {
		to = len(entries)
	}
	agentIDs := make(map[string]bool)
	for i := from; i < to; i++ {
		e := entries[i]
		if e.Type != "assistant" {
			continue
		}
		for _, blk := range e.Message.Content {
			if blk.Type == "tool_use" && blk.Name == "Agent" && blk.ID != "" {
				agentIDs[blk.ID] = true
			}
		}
	}
	if len(agentIDs) == 0 {
		return usageFields{}
	}

	subagentsDir := filepath.Join(strings.TrimSuffix(transcriptPath, ".jsonl"), "subagents")
	metas, err := filepath.Glob(filepath.Join(subagentsDir, "*.meta.json"))
	if err != nil || len(metas) == 0 {
		return usageFields{}
	}

	var acc usageFields
	for _, metaPath := range metas {
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta subagentMeta
		if err := json.Unmarshal(data, &meta); err != nil || !agentIDs[meta.ToolUseID] {
			continue
		}
		base := strings.TrimSuffix(metaPath, ".meta.json")
		agentEntries, err := loadTranscript(base + ".jsonl")
		if err != nil {
			continue
		}
		sub := sumSubagentWindow(agentEntries)
		acc.InputTokens += sub.InputTokens
		acc.OutputTokens += sub.OutputTokens
		acc.CacheReadInputTokens += sub.CacheReadInputTokens
		acc.CacheCreationInputTokens += sub.CacheCreationInputTokens
	}
	return acc
}

func sumSubagentWindow(entries []entry) usageFields {
	type usageKey struct{ in, out, read, create int }
	seen := make(map[usageKey]bool)
	var acc usageFields
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
		acc.CacheCreation.Ephemeral5m += u.CacheCreation.Ephemeral5m
		acc.CacheCreation.Ephemeral1h += u.CacheCreation.Ephemeral1h
	}
	return acc
}
