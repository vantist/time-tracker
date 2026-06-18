## Why

現有 `tt report` 只顯示基本 session 統計，缺少 output tokens、cache tokens、cost 及 by-project 分組，資訊不完整。長時間使用需要可持續閱讀的網頁介面，純 CLI 輸出不適合。此外，model 欄位未正確記錄（gateway 層 model ID 帶前綴，`UpsertSession` 不更新已存在 session），導致 cost 全部顯示 N/A；web dashboard 亦缺少 user active time per-project/session 及 work item 資訊。

## What Changes

- `tt report` 文字輸出補齊：output tokens、cache read tokens、cache creation tokens、依 project 分組（含 sessions、agent time、cost）
- `tt report --json` 補齊上述欄位，輸出完整 JSON
- 新增 `tt serve` subcommand：啟動本機 HTTP server，自動開啟瀏覽器
- 網頁 dashboard 顯示：Summary 卡片、7 天 daily timeline bar chart（純 CSS）、By Project table、Session 明細 table
- 前端無外部 JS 框架，每 60 秒自動刷新
- **Model 記錄修正**：`RecordResponse` 從 transcript JSONL 同時抽取 model，補寫至 `sessions` 表（僅在 model 為空時更新）
- **Pricing normalize**：`pricing.Calculate` 加 normalize 函式，去除 gateway 前綴（`vertex_ai/` 等），統一 pricing table key 格式；更新 pricing table 至最新定價
- **Dashboard 補齊**：By Project / Session 加 user active time 欄；Session 加 work item 欄；新增 By Work Item 分組 section

## Capabilities

### New Capabilities

- `report-text-tokens`: `tt report` 文字輸出增加 output tokens、cache read、cache creation、by-project 分組
- `web-dashboard`: `tt serve` 啟動 HTTP server 提供互動式網頁 dashboard，含 Summary、daily timeline、by-project、session 明細
- `model-cost-tracking`: 從 transcript JSONL 抽取 model 並補寫至 session；pricing normalize 支援 gateway model ID 前綴；pricing table 更新至最新定價
- `dashboard-enrichment`: web dashboard By Project / Session 加 user active time；Session 加 work item；新增 By Work Item 分組 section

### Modified Capabilities

(none)

## Impact

- Affected specs: report-text-tokens、web-dashboard（已有）、model-cost-tracking（新增）、dashboard-enrichment（新增）
- Affected code:
  - Modified: `internal/report/report.go`（Result struct 補欄位、FormatText 改版、FormatJSON 補欄位、daily grouping 後處理；ProjectSummary / SessionRow 加 UserActiveTimeSec / WorkItem；Groups 永遠計算）
  - Modified: `internal/report/html.go`（By Project / Session table 加 user time、work item 欄；加 By Work Item section）
  - Modified: `internal/recorder/response.go`（extractFromTranscript 同時抽 model；RecordResponse UPDATE sessions.model）
  - Modified: `internal/pricing/pricing.go`（加 normalize 函式；更新 pricing table）
  - New: `internal/report/html.go`（已建立）
  - New: `cmd/tt/serve_cmd.go`（已建立）

## Source

Derived from brainstorm plans:
- `.spex/plans/2026-06-18-brainstorm-report-web.md`（原始）
- `.spex/plans/2026-06-18-brainstorm-dashboard-enrichment.md`（新增範圍）

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
