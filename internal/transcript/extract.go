package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type usageFields struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
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
		Model string      `json:"model"`
		Usage usageFields `json:"usage"`
	} `json:"message"`
	Content []contentBlock `json:"content"`
}

type subagentMeta struct {
	ToolUseID string `json:"toolUseId"`
}

// ExtractWindow extracts token usage and model from a transcript JSONL file
// for entries in the range [from, to). to=-1 means read to EOF.
// Returns error if the file cannot be opened.
func ExtractWindow(path string, from, to int) (tokensJSON string, model string, err error) {
	all, err := loadTranscript(path)
	if err != nil {
		return "", "", err
	}

	// Model from last non-sidechain assistant entry (whole transcript).
	for i := len(all) - 1; i >= 0; i-- {
		e := all[i]
		if e.Type == "assistant" && !e.IsSidechain && e.Message.Model != "" {
			model = e.Message.Model
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

	sub := extractSubagentTokens(path, all, from)
	acc.InputTokens += sub.InputTokens
	acc.OutputTokens += sub.OutputTokens
	acc.CacheReadInputTokens += sub.CacheReadInputTokens
	acc.CacheCreationInputTokens += sub.CacheCreationInputTokens

	if acc.InputTokens == 0 && acc.OutputTokens == 0 {
		return "", model, nil
	}

	out, err := json.Marshal(map[string]int{
		"input_tokens":          acc.InputTokens,
		"output_tokens":         acc.OutputTokens,
		"cache_read_tokens":     acc.CacheReadInputTokens,
		"cache_creation_tokens": acc.CacheCreationInputTokens,
	})
	if err != nil {
		return "", model, nil
	}
	return string(out), model, nil
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
	}
	return acc
}

func extractSubagentTokens(transcriptPath string, entries []entry, offset int) usageFields {
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
	}
	return acc
}
