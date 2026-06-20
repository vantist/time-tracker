## Why

目前 `tt` 時間追蹤器僅支援 Claude Code 的本地日誌解析。為了全面支援多種 AI 開發工具（GitHub Copilot CLI 與 Antigravity），我們需要擴充 Token 計算與模型費用追蹤機制，改為直接解析其各自的本地日誌檔案。

## What Changes

- 當 `tool == "copilot-cli"` 時，`tt` 將自動尋找並解析本地的 `~/.copilot/session-state/<sessionId>/events.jsonl`，從 `session.shutdown` 事件的 `modelMetrics` 中讀取 Token 與模型數據。
- 當 `tool == "antigravity"` 時，`tt` 將自動尋找並解析本地的 `~/.gemini/antigravity/brain/<sessionId>/.system_generated/logs/transcript.jsonl`，統計主 Agent 的 model usage。
- 更新定價模組，支援新加入的模型如 `gpt-5.4`、`gpt-5-mini` 與 `claude-sonnet-4.6` 等。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `model-cost-tracking`: 擴充對 GitHub Copilot CLI 與 Antigravity 本地日誌格式（events.jsonl 與 transcript.jsonl）的解析，並整合其 Token 費用至現有的模型費用追蹤系統中。

## Impact

- Affected specs: `model-cost-tracking`
- Affected code:
  - New:
    - `internal/transcript/copilot_transcript.go`
    - `internal/transcript/antigravity_transcript.go`
  - Modified:
    - `cmd/tt/record.go`
    - `internal/pricing/pricing.go`
  - Removed:
    (none)

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-20-brainstorm-token-calculation-research.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
