package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ParseAntigravityLog parses the transcript.jsonl file and returns main agent model usage.
func ParseAntigravityLog(path string) (WindowResult, error) {
	all, err := loadTranscript(path)
	if err != nil {
		return WindowResult{}, err
	}

	mainModel := getAntigravityModel(all)
	acc := sumWindow(all, 0, len(all))

	var result WindowResult
	result.Usages = append(result.Usages, makeMainUsage(mainModel, acc))

	subUsages := extractSubagentModelUsages(path, all, 0, len(all))
	result.Usages = append(result.Usages, subUsages...)

	return result, nil
}

// getAntigravityModel reads and resolves the model name from ~/.gemini/antigravity-cli/settings.json,
// falling back to ~/.gemini/antigravity/settings.json, and normalizes it.
func getAntigravityModel(all []entry) string {
	if logModel := findMainModel(all); logModel != "unknown" {
		return logModel
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "gemini-3.5-flash"
	}

	paths := []string{
		filepath.Join(home, ".gemini", "antigravity-cli", "settings.json"),
		filepath.Join(home, ".gemini", "antigravity", "settings.json"),
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var cfg struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(data, &cfg); err == nil && cfg.Model != "" {
			return cleanAntigravityModel(cfg.Model)
		}
	}

	return "gemini-3.5-flash"
}

func cleanAntigravityModel(name string) string {
	name = strings.ToLower(name)
	// Strip parentheses e.g. (medium) or (large)
	if i := strings.Index(name, "("); i >= 0 {
		name = name[:i]
	}
	name = strings.TrimSpace(name)
	// Replace spaces/hyphens with a single hyphen, keeping alphanumeric characters and dots
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
