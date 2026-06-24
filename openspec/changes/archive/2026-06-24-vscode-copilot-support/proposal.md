## Why

tt 目前支援 Claude Code 和 Copilot CLI 的時間追蹤，但不支援 VS Code 內建的 GitHub Copilot Chat。大量使用者透過 VS Code Copilot Chat 進行 AI 輔助開發，卻無法自動記錄 AI 工作時間與 token 費用。VS Code 公開 API 無法直接觀察 Copilot 的 chat interactions，但 Copilot 會將 session 檔案寫入本地 workspaceStorage，可透過解析這些檔案來實現被動追蹤。

## What Changes

- 新增 VS Code Copilot session 檔案解析器，支援三種資料來源：`transcripts/*.jsonl`（對話事件）、`chatSessions/*.json`（session metadata）、`debug-logs/*/main.jsonl`（實際 token 使用量）
- 新增 token 估算策略：優先使用 debug-log 的實際 token 數，回退到 character-to-token ratio 估算，再回退到 ratio-based input:output 估算
- 新增 `tt setup --vscode-copilot` 命令，自動安裝輕量 VS Code Extension（TypeScript bridge）
- VS Code Extension 監聽 workspaceStorage 檔案變更，自動呼叫 `tt record prompt/response --tool vscode-copilot`
- `tt report` 顯示 `vscode-copilot` 作為 tool 類型，包含 session 時間、model 使用、token 使用量、費用估算

## Capabilities

### New Capabilities

- `vscode-copilot-session-parsing`: 解析 VS Code Copilot 的三種 session 檔案格式（transcripts JSONL、chatSessions JSON、debug-logs JSONL），提取 timestamps、model 資訊、tool calls、token 使用量
- `vscode-copilot-token-estimation`: 多層級 token 估算策略，從實際 token 數到 character-to-token ratio 到 ratio-based input:output 估算
- `vscode-copilot-bridge`: 輕量 VS Code Extension，監聽 workspaceStorage 檔案變更並呼叫 tt CLI 記錄事件

### Modified Capabilities

- `event-recording`: 新增 `vscode-copilot` tool 類型支援
- `setup-improvements`: 新增 `--vscode-copilot` flag 到 `tt setup` 命令

## Impact

- Affected code:
  - New: `internal/transcript/vscode_copilot.go`, `internal/transcript/vscode_copilot_token.go`, `internal/setup/vscode_copilot.go`, `vscode-extension/` (TypeScript bridge)
  - Modified: `cmd/tt/setup_cmd.go`, `internal/report/report.go`
  - Removed: (none)
- Affected specs: `vscode-copilot-session-parsing` (new), `vscode-copilot-token-estimation` (new), `vscode-copilot-bridge` (new), `event-recording` (modified), `setup-improvements` (modified)
- Dependencies: `ai-engineering-fluency` 專案的 session 檔案格式研究（參考，非直接依賴）

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-23-brainstorm-vscode-copilot-support.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
