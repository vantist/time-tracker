## Why

AI 工具（Claude Code、Copilot CLI）使用時間與 token 成本缺乏可見性，開發者無法量化 AI 工具的實際效益。透過 hook 系統自動記錄每次互動，提供精確的時間與成本報表。

## What Changes

- 新增 Go CLI binary `tt`，透過 hook 系統自動記錄 Claude Code 與 Copilot CLI 互動事件
- 資料存入本機 SQLite（`~/.tt/data.db`），保留原始事件供聚合查詢
- 提供 `tt report` 報表介面，支援依專案、時間範圍、工作項目篩選
- 支援手動 `tt work` 標記工作項目，覆蓋自動偵測的 git branch

## Capabilities

### New Capabilities

- `event-recording`: 接收 hook 呼叫並寫入 SQLite，記錄 session、turn、token 使用量與預估成本
- `session-management`: 管理 session 生命週期（建立、更新、推算結束時間）
- `time-aggregation`: 計算 agent 時間（精確）與 user 主動時間（idle threshold 近似）
- `cost-estimation`: 根據 model 欄位查詢定價表，計算每次 turn 的預估 USD 成本
- `report-query`: 聚合查詢介面，支援多維度篩選與輸出格式
- `work-item-tagging`: 手動標記工作項目，優先於 git branch 自動偵測
- `hook-integration`: Claude Code 與 Copilot CLI 的 hook 設定與事件欄位對照

- `transcript-anchored-token-capture`: 在 `UserPromptSubmit` 時記錄 transcript 的當前行數（`prompt_line_offset`），Stop hook 讀取時以此 offset 為 anchor 切割 transcript，避免 `/clear` 後立即下指令造成的 lastUserIdx 計算錯誤
- `subagent-token-capture`: Stop hook 讀取 transcript 時，掃描主 transcript 中 `tool_use[name=="Agent"]` 的 toolUseIds，對應 `<session>/subagents/*.meta.json` 找到 subagent jsonl 路徑，累加各 subagent 的 assistant token 至現有欄位（input/output/cache），合計後寫入 turn；已知限制：不遞迴處理 subagent 再呼叫 agent 的巢狀情境

### Modified Capabilities

- `event-recording`: 加入 `prompt_line_offset` 欄位至 `turns` 表；`RecordPrompt` 記錄 transcript path 與行數；`RecordResponse` 改用 stored offset 切割 transcript，並合計 subagent token

## Impact

- Affected specs: `event-recording`, `session-management`, `time-aggregation`, `cost-estimation`, `report-query`, `work-item-tagging`, `hook-integration`
- Affected code:
  - New: `cmd/tt/main.go`
  - New: `internal/db/schema.go`
  - New: `internal/db/migrations.go`
  - New: `internal/recorder/recorder.go`
  - New: `internal/session/session.go`
  - New: `internal/aggregator/aggregator.go`
  - New: `internal/pricing/pricing.go`
  - New: `internal/report/report.go`
  - New: `internal/hooks/claude.go`
  - New: `internal/hooks/copilot.go`
  - New: `go.mod`
  - New: `go.sum`
  - Modified: `internal/db/schema.go` — `turns` 表加入 `transcript_path TEXT`, `prompt_line_offset INTEGER`
  - Modified: `internal/recorder/recorder.go` — `RecordPrompt` 記錄 offset；`RecordResponse` 用 offset 切割 transcript
  - Modified: `cmd/tt/record.go` — `UserPromptSubmit` hook 傳入 `transcript_path`；`extractFromTranscriptAtOffset` 加入 `extractSubagentTokens` 並合計
  - Modified: `internal/recorder/response.go` — `RecordResponse` 呼叫 subagent token 合計邏輯

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-17-brainstorm-time-tracker.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
