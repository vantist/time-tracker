# tt — Design Notes

## Hook Integration: Confirmed Interface

### Data Delivery
Claude Code hooks receive data via **stdin as JSON**, not environment variables.
The hook command string is run as a shell command with stdin piped from Claude Code.

### UserPromptSubmit stdin fields
```json
{
  "session_id": "...",
  "cwd": "/path/to/project",
  "hook_event_name": "UserPromptSubmit",
  "permission_mode": "...",
  "model": "claude-sonnet-4-6",
  "prompt": "..."
}
```

### Stop stdin fields
```json
{
  "session_id": "...",
  "cwd": "/path/to/project",
  "hook_event_name": "Stop",
  "permission_mode": "..."
}
```
Note: token usage data is NOT documented in Stop hook payload. May appear as undocumented field.

### CLI Design Consequence
Original spec assumed env-var flags:
```
tt record prompt --session $CLAUDE_SESSION_ID --project $CLAUDE_PROJECT_PATH --tool claude-code --model $CLAUDE_MODEL
tt record response --session $CLAUDE_SESSION_ID --tokens "$CLAUDE_USAGE_JSON"
```

**Actual design**: both commands read stdin JSON:
```
"command": "tt record prompt"
"command": "tt record response"
```

`tt record prompt` reads from stdin: extracts `session_id`, `cwd` (→ project), `model`.
`tt record response` reads from stdin: extracts `session_id`, token fields if present (graceful NULL if absent).

Flags (`--session`, `--project`, `--tool`, `--model`, `--tokens`) retained for manual/testing use,
stdin takes precedence when available (non-empty stdin detected via `os.Stdin.Stat()`).

### Hook settings.json format
```json
{
  "hooks": {
    "UserPromptSubmit": [{"hooks": [{"type": "command", "command": "tt record prompt"}]}],
    "Stop": [{"hooks": [{"type": "command", "command": "tt record response"}]}]
  }
}
```

## Copilot CLI Hook Integration: Confirmed Interface

### agentStop stdin fields
```json
{
  "sessionId": "...",
  "timestamp": 1234567890000,
  "cwd": "/path/to/project",
  "transcriptPath": "/path/to/transcript",
  "stopReason": "end_turn"
}
```
- Registration: `~/.copilot/settings.json` under `hooks` key (user-level)
- **No token usage, no model name** in payload
- Fallback strategy: token fields written as NULL for Copilot sessions; `tt record response` must handle missing token data gracefully (already required by spec)

### userPromptSubmitted stdin fields
- Similar shape: `sessionId`, `cwd`, `timestamp`
- No model field — write model as NULL or "copilot-cli"

### Hook command format
```
"command": "tt record prompt --tool copilot-cli"
"command": "tt record response --tool copilot-cli"
```
`--tool` flag needed so recorder can set the correct tool name (stdin won't provide it).

## Open Questions

- [x] Stop hook token fields — test empirically whether Claude Code Stop stdin includes usage data.
  **Resolution**: Checked empirically; Claude Code Stop hook stdin does not include usage data. Instead, `tt record response` parses the local transcript JSONL file.

