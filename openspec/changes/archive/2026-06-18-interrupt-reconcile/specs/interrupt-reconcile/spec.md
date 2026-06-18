## ADDED Requirements

### Requirement: 補算懸空 turn 的 token 與結束時間

系統 SHALL 提供 `MaybeReconcile(conn *sql.DB)` 函式，掃描所有 `response_at IS NULL` 且具備 `transcript_path` 與 `prompt_line_offset` 的 turn，從 transcript JSONL 提取 token 資料，估算 `response_at`，並寫回 DB。

#### Scenario: 中間懸空 turn 補算成功

- **WHEN** DB 中存在 `response_at IS NULL` 的 turn，且該 turn 存在後繼 turn（`next_prompt_at` 不為 NULL）
- **THEN** 系統從 transcript 提取 from=`prompt_line_offset` 到 to=`next_offset` 的 token 窗口，將 `response_at` 設為 `next_prompt_at - 1ms`，並 UPDATE turn row（input_tokens、output_tokens、cache_read_tokens、cache_creation_tokens、estimated_cost_usd、response_at）

#### Scenario: 最後一個懸空 turn（process 已死）補算成功

- **WHEN** DB 中存在 `response_at IS NULL` 的 turn，且該 turn 為 session 內最後一個（無後繼 turn），且對應 process 已不存活
- **THEN** 系統從 transcript 提取 from=`prompt_line_offset` 到 EOF 的 token 窗口，將 `response_at` 設為 transcript 檔案的 mtime，並 UPDATE turn row

#### Scenario: 進行中的 turn 不被誤算

- **WHEN** DB 中存在 `response_at IS NULL` 的 turn，且該 turn 為 session 內最後一個，且對應 process 仍存活（`process.IsAlive` 回傳 true）
- **THEN** 系統 skip 該 turn，不做任何 UPDATE

#### Scenario: Idempotency — Stop hook 先寫入時 reconcile 為 no-op

- **WHEN** `MaybeReconcile` 執行時，某 turn 的 `response_at` 已被 Stop hook 寫入（非 NULL）
- **THEN** UPDATE 條件 `WHERE id = ? AND response_at IS NULL` 不匹配任何 row，不修改該 turn 的資料

### Requirement: 並發安全

`MaybeReconcile` SHALL 同時使用 in-process mutex 與 cross-process flock 防止並發重入。

#### Scenario: in-process 並發呼叫被跳過

- **WHEN** `MaybeReconcile` 正在執行中，同一 process 內另一個 goroutine 呼叫 `MaybeReconcile`
- **THEN** 後者立即 return，不等待，不執行 reconcile 邏輯

#### Scenario: cross-process 並發呼叫被跳過

- **WHEN** `tt serve` 正在執行 `MaybeReconcile`，使用者同時執行 `tt report`
- **THEN** `tt report` 的 `MaybeReconcile` 嘗試 `flock(LOCK_NB)` 失敗，立即 return，不等待

### Requirement: 觸發點整合

系統 SHALL 在以下三個觸發點呼叫 `MaybeReconcile`：

1. `tt serve` 啟動時（無條件）
2. `tt report` 執行前（無條件）
3. `/api/report` 每次 refresh 時，若 `hasActiveSession` 為 false 則呼叫；若有 active session 則 skip

#### Scenario: tt serve 啟動觸發補算

- **WHEN** 使用者執行 `tt serve`
- **THEN** 系統在開始 HTTP server 前呼叫 `MaybeReconcile`，補算所有歷史懸空 turn

#### Scenario: tt report 觸發補算

- **WHEN** 使用者執行 `tt report`
- **THEN** 系統在輸出報告前呼叫 `MaybeReconcile`，確保當次報告包含已補算的 token

#### Scenario: /api/report 無 active session 時觸發補算

- **WHEN** 瀏覽器呼叫 `/api/report`，且所有 session 的 process 均已結束
- **THEN** handler 呼叫 `MaybeReconcile`，回傳補算後的報告資料

#### Scenario: /api/report 有 active session 時跳過補算

- **WHEN** 瀏覽器呼叫 `/api/report`，且至少一個 session 的 process 仍存活
- **THEN** handler skip `MaybeReconcile`，直接回傳目前 DB 資料，不嘗試取鎖

### Requirement: Transcript 提取邏輯共用化

系統 SHALL 將 `extractFromTranscriptAtOffset`、`extractSubagentTokens` 等提取函式移至 `internal/transcript` package，供 `cmd/tt/record.go` 與 `internal/reconcile/reconcile.go` 共用。

#### Scenario: record.go 使用共用提取函式

- **WHEN** `cmd/tt/record.go` 在 Stop hook 觸發時計算 token
- **THEN** 呼叫 `internal/transcript.ExtractWindow`（或等效函式），行為與重構前一致，現有測試全數通過

#### Scenario: reconcile 使用共用提取函式

- **WHEN** `internal/reconcile/reconcile.go` 補算懸空 turn
- **THEN** 呼叫 `internal/transcript.ExtractWindow`，不依賴 cmd 層的任何 context
