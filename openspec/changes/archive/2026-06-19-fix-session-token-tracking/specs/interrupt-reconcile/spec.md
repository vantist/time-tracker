## MODIFIED Requirements

### Requirement: 補算懸空 turn 的 token 與結束時間

系統 SHALL 提供 `MaybeReconcile(conn *sql.DB)` 函式，掃描所有符合下列任一條件的 turn，並在 process 結束後從 transcript 重算 token（含 subagent）寫回 DB：

1. `response_at IS NULL`（Stop hook 未執行）
2. `input_tokens IS NULL`（token 未寫入）
3. `subagent_tokens_settled = 0`（subagent token 待重算）

且該 turn 具備 `transcript_path` 與 `prompt_line_offset`。

#### Scenario: 中間懸空 turn 補算成功

- **WHEN** DB 中存在 `response_at IS NULL` 的 turn，且該 turn 存在後繼 turn（`next_prompt_at` 不為 NULL）
- **THEN** 系統從 transcript 提取 `[prompt_line_offset, next_offset)` 的 token 窗口（`WindowResult`），將 `response_at` 設為 `next_prompt_at - 1ms`，並 UPDATE turn row（input_tokens、output_tokens、cache_read_tokens、cache_creation_tokens、cache_creation_5m_tokens、cache_creation_1h_tokens、model、estimated_cost_usd、response_at、subagent_tokens_settled=1）

#### Scenario: 最後一個懸空 turn（process 已死）補算成功

- **WHEN** DB 中存在 `response_at IS NULL` 的 turn，且該 turn 為 session 內最後一個（無後繼 turn），且對應 process 已不存活
- **THEN** 系統從 transcript 提取 `[prompt_line_offset, EOF)` 的 token 窗口，將 `response_at` 設為 transcript 檔案的 mtime，並 UPDATE turn row（含 `subagent_tokens_settled=1`）

#### Scenario: 進行中的 turn 不被誤算

- **WHEN** DB 中存在 `response_at IS NULL` 的 turn，且該 turn 為 session 內最後一個，且對應 process 仍存活（`process.IsAlive` 回傳 true）
- **THEN** 系統 skip 該 turn，不做任何 UPDATE

#### Scenario: Stop hook 已寫 response_at 但 subagent_tokens_settled=0 時重算 token

- **WHEN** `MaybeReconcile` 執行時，某 turn 的 `response_at` 已被 Stop hook 寫入（非 NULL），`input_tokens IS NOT NULL`，但 `subagent_tokens_settled = 0`，且 process 已不存活
- **THEN** reconcile 重新執行 `ExtractWindow`，覆蓋 token 欄位（包含正確的 subagent token），並將 `subagent_tokens_settled` 設為 1

#### Scenario: subagent_tokens_settled=1 的 turn 不被重算

- **WHEN** `MaybeReconcile` 執行時，某 turn 的 `response_at IS NOT NULL`、`input_tokens IS NOT NULL`、`subagent_tokens_settled = 1`
- **THEN** reconcile WHERE 條件不匹配該 turn，不做任何 UPDATE（no-op）

#### Scenario: Idempotency — 同一 turn 多次重算結果一致

- **WHEN** 相同 transcript 的同一 turn 被 `MaybeReconcile` 重算兩次
- **THEN** 第二次 UPDATE 產生相同結果，不累加或重複計算

### Requirement: 補算時使用 WindowResult typed struct

系統 SHALL 在 `reconcile.go` 中直接使用 `transcript.WindowResult` struct 的欄位存取 token 值，不使用 JSON 字串 parse。

#### Scenario: reconcile 直接存取 WindowResult 欄位

- **WHEN** `transcript.ExtractWindow` 回傳 `WindowResult`
- **THEN** `reconcile.go` 直接讀取 `result.InputTokens`、`result.CacheCreate5m` 等欄位，不需要呼叫 `parseTokensJSON`

#### Scenario: ExtractWindow 回傳空 WindowResult 時 reconcile 跳過

- **WHEN** `transcript.ExtractWindow` 回傳 `WindowResult{}` 零值（InputTokens=0, OutputTokens=0）
- **THEN** reconcile 不更新該 turn（`tokensJSON == ""` 的等效判斷 → 改為 `result.InputTokens == 0 && result.OutputTokens == 0`），跳過此 turn
