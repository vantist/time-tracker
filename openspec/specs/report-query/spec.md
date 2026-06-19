# report-query Specification

## Purpose
TBD - created by archiving change ai-tool-time-tracker. Update Purpose after archive.
## Requirements
### Requirement: 基本報表輸出

系統 SHALL 透過 `tt report` 命令輸出過去 7 天（預設）的聚合統計，格式為純文字，包含：
- Sessions 數量
- Agent 時間（h m 格式）
- User 主動時間（h m 格式，含使用的 idle threshold）
- Token 總量（input、output、cache hit 比例）
- 預估成本（USD）
- Sessions 表格中包含 `Agent` 欄位（對齊欄寬 `%-12s`）
- 新增 `─── By Agent ───` 統計表格，顯示各 Agent 獨立計算的 User 主動時間與成本。

#### Scenario: 無資料時顯示空報表而非錯誤
- **WHEN** `tt report` 被呼叫，且資料庫中 7 天內無任何 turns
- **THEN** 輸出 "No data for the selected period."
- **THEN** exit code 0

#### Scenario: 有資料時輸出格式正確
- **WHEN** 7 天內有資料，`tt report` 被呼叫
- **THEN** stdout 輸出包含 "Sessions:", "Agent time:", "User active:", "Tokens in:", "Est. cost:" 等欄位
- **THEN** Agent time 格式為 `Xh Ym`（如 `2h 34m`）
- **THEN** stdout 輸出包含 "─── By Agent ───" 區塊，列出各 Agent 名稱、User time、Cost
- **THEN** stdout 的 Sessions 表格中包含 "Agent" 欄位標題，且對應列處顯示該 session 正規化後的 Agent 名稱

### Requirement: 篩選條件

系統 SHALL 支援以下篩選選項：

| 選項 | 說明 |
|------|------|
| `--project <name>` | 依 `sessions.project` 路徑末段或完整路徑篩選 |
| `--since <duration\|date>` | 時間範圍：`7d`、`30d`、`2026-06-01` |
| `--format json` | 輸出 JSON 格式（預設 text） |

#### Scenario: --project 篩選只顯示指定專案

- **WHEN** `tt report --project time-tracker` 被呼叫
- **THEN** 只包含 `sessions.project` 路徑含 "time-tracker" 的 sessions

#### Scenario: --since 7d 篩選過去 7 天

- **WHEN** `tt report --since 7d` 被呼叫（今日為 2026-06-17）
- **THEN** 只包含 `prompt_at >= 2026-06-10 00:00:00 UTC` 的 turns

#### Scenario: --since 指定日期篩選

- **WHEN** `tt report --since 2026-06-01` 被呼叫
- **THEN** 只包含 `prompt_at >= 2026-06-01 00:00:00 UTC` 的 turns

#### Scenario: --format json 輸出合法 JSON

- **WHEN** `tt report --format json` 被呼叫
- **THEN** stdout 為合法 JSON，可被 `jq` 解析
- **THEN** JSON 包含 `sessions_count`, `agent_time_sec`, `user_active_time_sec`, `input_tokens`, `output_tokens`, `estimated_cost_usd` 欄位

### Requirement: 依工作項目分組報表

系統 SHALL 支援 `tt report --by-work-item`，將資料依 `work_item ?? branch ?? "untagged"` 分組，每組顯示時間與成本。

#### Scenario: 有工作項目標記時優先顯示 work_item

- **WHEN** session 的 `work_item = "login-redesign"`, `branch = "feature/auth"`
- **THEN** 報表分組使用 "login-redesign"（不用 branch）

#### Scenario: 無 work_item 時使用 branch

- **WHEN** session 的 `work_item = NULL`, `branch = "feature/auth"`
- **THEN** 報表分組使用 "feature/auth"

#### Scenario: 無 work_item 且無 branch 時顯示 untagged

- **WHEN** session 的 `work_item = NULL`, `branch = NULL`
- **THEN** 報表分組使用 "untagged"

