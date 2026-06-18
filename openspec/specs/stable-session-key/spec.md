# stable-session-key Specification

## Purpose
TBD - created by archiving change session-identity. Update Purpose after archive.
## Requirements
### Requirement: 穩定工作 session 識別

系統 SHALL 以 `(process_pid, process_start)` 組合作為工作 session 的唯一識別符。同一 Claude Code 進程內，無論執行多少次 `/clear`，皆 SHALL 對應到同一個 session 記錄。

#### Scenario: 首次建立工作 session

- **WHEN** `tt record prompt` 收到 `$PROCESS_PID` 與 `$PROCESS_START`，且 DB 中不存在相同 `(process_pid, process_start)` 的 session
- **THEN** 建立新 session 記錄，`process_pid` 與 `process_start` 設為收到的值，`conversation_id` 設為 stdin 的 `session_id` UUID

#### Scenario: /clear 後繼續記錄同一工作 session

- **WHEN** `tt record prompt` 收到相同的 `$PROCESS_PID` 與 `$PROCESS_START`，但 `session_id` UUID 與現有 session 的 `conversation_id` 不同（代表發生過 `/clear`）
- **THEN** 更新現有 session 的 `conversation_id` 為新 UUID，`last_seen` 更新為當前時間，不建立新 session

#### Scenario: 舊資料（無 process_pid）不受影響

- **WHEN** DB 中存在 `process_pid = NULL` 的 session 記錄
- **THEN** 這些記錄 SHALL 繼續可讀，不被修改或刪除

### Requirement: Hook 環境傳遞 process 識別資訊

Claude Code `UserPromptSubmit` hook SHALL 在執行 `tt record prompt` 前，將 `$PPID`（Claude Code PID）設為 `PROCESS_PID` env var，並將估算的進程啟動 Unix epoch 時間戳設為 `PROCESS_START` env var。

#### Scenario: Hook 正確傳遞 env var

- **WHEN** Claude Code 觸發 `UserPromptSubmit` hook
- **THEN** `tt record prompt` 可透過 `$PROCESS_PID` 讀到非零整數（= `$PPID`），並透過 `$PROCESS_START` 讀到 > 0 的 Unix epoch 整數

#### Scenario: Hook 取不到 process_start 時的降級處理

- **WHEN** `ps` 指令失敗導致 `$PROCESS_START` 為空或 0
- **THEN** `tt record prompt` SHALL 降級為僅用 `process_pid` 進行 upsert（允許同 PID 合併），並在 stderr 輸出警告

### Requirement: session 工作時間跨對話段落計算

session 工作時間 SHALL 以同一 `(process_pid, process_start)` 下所有 prompt/response 記錄的時間範圍計算，而非以單一 `conversation_id` 計算。

#### Scenario: 跨 /clear 的工作時間累計

- **WHEN** 查詢某 session 的工作時間，該 session 在同一 Claude Code 進程中發生過 2 次 `/clear`（共 3 段 conversation_id）
- **THEN** 回報的工作時間 = 第一段第一個 prompt 到最後一段最後一個 response 的時間差（含中間間隔的最大閾值截斷邏輯）

### Requirement: DB 欄位新增

`sessions` 表 SHALL 新增以下欄位：

| 欄位 | 型別 | 說明 |
|-----|------|------|
| `process_pid` | INTEGER | Claude Code 進程 PID（`$PPID`） |
| `process_start` | INTEGER | 進程啟動 Unix epoch 秒 |
| `conversation_id` | TEXT | 當前對話段落 UUID（原 session_id） |

現有 sessions 的這三欄 SHALL 預設為 NULL。

#### Scenario: Migration 成功執行

- **WHEN** 執行 DB migration
- **THEN** `sessions` 表包含 `process_pid`、`process_start`、`conversation_id` 三欄，且現有資料的這三欄值為 NULL

