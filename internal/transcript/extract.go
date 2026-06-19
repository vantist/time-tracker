package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)


// ModelUsage details token usage for a specific model and role.
type ModelUsage struct {
	Model               string
	IsSubagent          bool
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheCreationTokens int
	CacheCreation5m     int
	CacheCreation1h     int
}

// WindowResult holds extracted token usage and model for a transcript window.
type WindowResult struct {
	Usages []ModelUsage
}

func (r WindowResult) Model() string {
	for _, u := range r.Usages {
		if !u.IsSubagent {
			return u.Model
		}
	}
	return ""
}

func (r WindowResult) InputTokens() int {
	var total int
	for _, u := range r.Usages {
		total += u.InputTokens
	}
	return total
}

func (r WindowResult) OutputTokens() int {
	var total int
	for _, u := range r.Usages {
		total += u.OutputTokens
	}
	return total
}

func (r WindowResult) CacheReadTokens() int {
	var total int
	for _, u := range r.Usages {
		total += u.CacheReadTokens
	}
	return total
}

func (r WindowResult) CacheCreationTokens() int {
	var total int
	for _, u := range r.Usages {
		total += u.CacheCreationTokens
	}
	return total
}

func (r WindowResult) CacheCreate5m() int {
	var total int
	for _, u := range r.Usages {
		total += u.CacheCreation5m
	}
	return total
}

func (r WindowResult) CacheCreate1h() int {
	var total int
	for _, u := range r.Usages {
		total += u.CacheCreation1h
	}
	return total
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

	mainModel := findMainModel(all)

	end := len(all)
	if to != -1 && to < end {
		end = to
	}
	if from > end {
		from = end
	}

	acc := sumWindow(all, from, end)

	var result WindowResult
	if acc.InputTokens > 0 || acc.OutputTokens > 0 || mainModel != "unknown" {
		result.Usages = append(result.Usages, makeMainUsage(mainModel, acc))
	}

	subUsages := extractSubagentModelUsages(path, all, from, end)
	result.Usages = append(result.Usages, subUsages...)

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

	mainModel := findMainModel(all)

	lastUserIdx := -1
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].Type == "user" && !all[i].IsSidechain {
			lastUserIdx = i
			break
		}
	}

	winFrom, winTo := lastUserIdx+1, len(all)
	acc := sumWindow(all, winFrom, winTo)

	if acc.InputTokens == 0 && acc.OutputTokens == 0 && lastUserIdx > 0 {
		// /clear race: fallback to previous turn window.
		prevUserIdx := -1
		for i := lastUserIdx - 1; i >= 0; i-- {
			if all[i].Type == "user" && !all[i].IsSidechain {
				prevUserIdx = i
				break
			}
		}
		winFrom, winTo = prevUserIdx+1, lastUserIdx
		acc = sumWindow(all, winFrom, winTo)
	}

	var result WindowResult
	if acc.InputTokens > 0 || acc.OutputTokens > 0 || mainModel != "unknown" {
		result.Usages = append(result.Usages, makeMainUsage(mainModel, acc))
	}

	subUsages := extractSubagentModelUsages(path, all, winFrom, winTo)
	result.Usages = append(result.Usages, subUsages...)

	return result, nil
}

func findMainModel(all []entry) string {
	for i := len(all) - 1; i >= 0; i-- {
		e := all[i]
		if e.Type == "assistant" && !e.IsSidechain && e.Message.Model != "" {
			return e.Message.Model
		}
	}
	return "unknown"
}

func makeMainUsage(model string, acc usageFields) ModelUsage {
	return ModelUsage{
		Model:               model,
		IsSubagent:          false,
		InputTokens:         acc.InputTokens,
		OutputTokens:        acc.OutputTokens,
		CacheReadTokens:     acc.CacheReadInputTokens,
		CacheCreationTokens: acc.CacheCreationInputTokens,
		CacheCreation5m:     acc.CacheCreation.Ephemeral5m,
		CacheCreation1h:     acc.CacheCreation.Ephemeral1h,
	}
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

// extractSubagentModelUsages scans entries[from:min(to,len(entries))] for Agent tool_use IDs,
// and extracts token usage grouped by model.
func extractSubagentModelUsages(transcriptPath string, entries []entry, from, to int) []ModelUsage {
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
		return nil
	}

	subagentsDir := filepath.Join(strings.TrimSuffix(transcriptPath, ".jsonl"), "subagents")
	metas, err := filepath.Glob(filepath.Join(subagentsDir, "*.meta.json"))
	if err != nil || len(metas) == 0 {
		return nil
	}

	subUsages := make(map[string]*ModelUsage)
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

		var model string
		for i := len(agentEntries) - 1; i >= 0; i-- {
			e := agentEntries[i]
			if e.Type == "assistant" && e.Message.Model != "" {
				model = e.Message.Model
				break
			}
		}
		if model == "" {
			model = "unknown"
		}

		sub := sumSubagentWindow(agentEntries)

		u, exists := subUsages[model]
		if !exists {
			u = &ModelUsage{
				Model:      model,
				IsSubagent: true,
			}
			subUsages[model] = u
		}
		u.InputTokens += sub.InputTokens
		u.OutputTokens += sub.OutputTokens
		u.CacheReadTokens += sub.CacheReadInputTokens
		u.CacheCreationTokens += sub.CacheCreationInputTokens
		u.CacheCreation5m += sub.CacheCreation.Ephemeral5m
		u.CacheCreation1h += sub.CacheCreation.Ephemeral1h
	}

	var results []ModelUsage
	for _, u := range subUsages {
		results = append(results, *u)
	}
	return results
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
