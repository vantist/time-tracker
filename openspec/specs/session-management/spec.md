# session-management Specification

## Purpose
TBD - created by archiving change ai-tool-time-tracker. Update Purpose after archive.
## Requirements
### Requirement: Session 資料模型

系統 SHALL 維護 `sessions` 表，欄位如下：

| 欄位 | 類型 | 說明 |
|------|------|------|
| `id` | TEXT PRIMARY KEY | AI 工具提供的 session ID |
| `project` | TEXT | git root 路徑，fallback 為 cwd |
| `tool` | TEXT | `"claude-code"` 或 `"copilot-cli"` |
| `started_at` | INTEGER | unix milliseconds，第一個 prompt 時間 |
| `ended_at` | INTEGER | unix milliseconds，最後一個 response 時間，可 NULL |
| `branch` | TEXT | git branch 名稱，可 NULL |
| `work_item` | TEXT | 手動標記的工作項目，可 NULL |

#### Scenario: Session upsert 保留 started_at

- **WHEN** `sessions` 表已存在 `id = "abc123"` 的 session（`started_at = T1`），再次 INSERT 相同 ID
- **THEN** `started_at` 保持 T1，不被覆蓋
- **THEN** `ended_at` 更新為最新值

### Requirement: Turn 資料模型

系統 SHALL 維護 `turns` 表，欄位如下：

| 欄位 | 類型 | 說明 |
|------|------|------|
| `id` | INTEGER PRIMARY KEY AUTOINCREMENT | |
| `session_id` | TEXT REFERENCES sessions | |
| `model` | TEXT | 模型名稱 |
| `prompt_at` | INTEGER | unix milliseconds |
| `response_at` | INTEGER | unix milliseconds，可 NULL（等待 response） |
| `input_tokens` | INTEGER | 可 NULL |
| `output_tokens` | INTEGER | 可 NULL |
| `cache_read_tokens` | INTEGER | 可 NULL |
| `cache_creation_tokens` | INTEGER | 可 NULL |
| `estimated_cost_usd` | REAL | 可 NULL（未知 model 時） |

#### Scenario: 資料庫初始化建立 schema

- **WHEN** `tt` 首次執行任何命令，且 `~/.tt/data.db` 不存在
- **THEN** 系統自動建立 `~/.tt/data.db`
- **THEN** 建立 `sessions` 表與 `turns` 表（包含所有欄位與外鍵約束）
- **THEN** 命令正常繼續執行

#### Scenario: 已存在的資料庫不重建 schema

- **WHEN** `~/.tt/data.db` 已存在且 schema 正確
- **THEN** 系統不刪除或重建任何表

### Requirement: 資料庫路徑可透過環境變數覆蓋

系統 SHALL 在環境變數 `TT_DB_PATH` 設定時，使用該路徑作為 SQLite 資料庫路徑（取代預設的 `~/.tt/data.db`）。

#### Scenario: 測試時使用臨時資料庫

- **WHEN** 環境變數 `TT_DB_PATH=/tmp/tt-test.db` 設定
- **THEN** 所有讀寫操作使用 `/tmp/tt-test.db`，不碰 `~/.tt/data.db`

