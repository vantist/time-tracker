## Why

`tt record` 以 Claude Code hook 的 `session_id`（UUID）識別 session，但 `/clear` 每次產生新 UUID，導致同一次工作被切割成多個 session，造成成本與時間估算失真。

## What Changes

- `sessions` 表新增 `process_pid`、`process_start`、`conversation_id` 欄位
- `tt record prompt` 透過 env var `$PPID` 接收穩定的工作 session 識別符
- Recorder 的 upsert 邏輯改以 `(process_pid, process_start)` 作為 session key，`session_id` UUID 降為對話段落識別符
- DB migration：現有 sessions 的新欄位預設 NULL（標記為舊資料）
- session 工作時間計算跨所有對話段落（所有 conversation_id）

## Capabilities

### New Capabilities

- `stable-session-key`: 以 `(process_pid, process_start)` 組合作為跨 `/clear` 穩定的工作 session 識別符，避免 PID reuse

### Modified Capabilities

(none)

## Impact

- Affected code:
  - Modified: `tt` (CLI binary，新增 env var 讀取邏輯)
  - Modified: `internal/recorder/recorder.go` (upsert 邏輯改用新 session key)
  - Modified: `internal/db/schema.go` 或對應 migration 檔案（新增欄位）
  - Modified: `.claude/settings.json` 或 hook 設定（UserPromptSubmit hook 傳入 `$PPID`）

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-18-brainstorm-session-identity.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
