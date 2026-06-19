## 1. DB Schema 擴充

- [x] 1.1 在 `internal/db/schema.go` 的 `addTurnColumns` 新增四個欄位：`model TEXT`、`cache_creation_5m_tokens INTEGER`、`cache_creation_1h_tokens INTEGER`、`subagent_tokens_settled BOOLEAN DEFAULT 0`（使用 PRAGMA table_info 檢查再 ALTER TABLE）
- [x] 1.2 寫失敗測試：舊 DB 無這四個欄位 → `db.Open()` 後 PRAGMA table_info 驗證欄位存在；欄位已存在 → 不回傳錯誤

## 2. transcript.WindowResult typed struct + ExtractWindow 改簽章

- [x] 2.1 在 `internal/transcript/extract.go` 定義 `WindowResult` struct（InputTokens、OutputTokens、CacheReadTokens、CacheCreationTokens、CacheCreate5m、CacheCreate1h、Model）
- [x] 2.2 擴充 `usageFields` struct 加入 `CacheCreation` 子物件（`Ephemeral5m`、`Ephemeral1h`），對應 `usage.cache_creation.ephemeral_5m_input_tokens` 與 `ephemeral_1h_input_tokens`
- [x] 2.3 將 `ExtractWindow` 回傳型別改為 `(WindowResult, error)`，移除 `tokensJSON string` 回傳值；在函數內填入 `WindowResult` 的所有欄位（含 5m/1h、Model）
- [x] 2.4 寫失敗測試：建立 fixture transcript（含 `cache_creation.ephemeral_5m_input_tokens`）→ `ExtractWindow` 回傳 `WindowResult.CacheCreate5m` 正確值；無 cache_creation 欄位 → 回傳 0 不報錯

## 3. Bug A：extractSubagentTokens 加 to 邊界

- [x] 3.1 修改 `internal/transcript/extract.go` 中 `extractSubagentTokens` 簽章改為 `(path string, entries []entry, from, to int) usageFields`，掃描範圍改為 `[from, min(to, len(entries)))`
- [x] 3.2 更新 `ExtractWindow` 中對 `extractSubagentTokens` 的呼叫，傳入 `end`（已裁切的上界）
- [x] 3.3 寫失敗測試：兩個連續 turn，Turn 1 range=[0,10)、Turn 2 range=[10,20)，各有一個 Agent tool_use → `ExtractWindow(path, 0, 10)` 不含 Turn 2 的 subagent token；`ExtractWindow(path, 10, 20)` 不含 Turn 1 的 subagent token

## 4. Bug E：新增 ExtractLastTurn，移除 record.go 重複 code

- [ ] 4.1 在 `internal/transcript/extract.go` 新增 `ExtractLastTurn(path string) (WindowResult, error)`，實作「找最後 user entry + /clear fallback + subagent（to=-1）」邏輯（搬移自 `record.go:extractFromTranscript`）
- [ ] 4.2 修改 `cmd/tt/record.go`：`extractFromTranscript(path)` 改為呼叫 `transcript.ExtractLastTurn(path)` 並 marshal 回 JSON string
- [ ] 4.3 修改 `cmd/tt/record.go`：`extractFromTranscriptAtOffset(path, offset)` 改為呼叫 `transcript.ExtractWindow(path, offset, -1)` 並 marshal 回 JSON string
- [ ] 4.4 刪除 `cmd/tt/record.go` 中所有重複 type：`transcriptUsageFields`、`contentBlock`、`transcriptEntry`、`subagentMeta`、`loadTranscript`、`sumWindow`、`extractSubagentTokens`、`sumSubagentWindow`
- [ ] 4.5 寫失敗測試：`ExtractLastTurn` 正確處理 /clear 競態（lastUserIdx 為最後一筆 user entry，後無 assistant entries → fallback 至前一個 window）

## 5. Bug F：countLines 改 bufio.Scanner

- [ ] 5.1 修改 `internal/recorder/recorder.go` 的 `countLines`：改用 `bufio.NewScanner`、設定 1MB buffer（`scanner.Buffer(make([]byte, 64*1024), 1024*1024)`）、逐行計數；移除 `io.ReadAll` 與 `bytes.Count`
- [ ] 5.2 寫失敗測試：建立 2MB 假 transcript（每行 JSON）→ `countLines` 回傳正確行數；單行超過 64KB → 不 panic 不回傳錯誤

## 6. Bug D：subagent_tokens_settled 結算旗標

- [ ] 6.1 修改 `internal/recorder/response.go` 的 `RecordResponse` UPDATE：加入 `subagent_tokens_settled = 0`
- [ ] 6.2 修改 `internal/reconcile/reconcile.go` 的 WHERE 條件：改為 `(t.response_at IS NULL OR t.input_tokens IS NULL OR t.subagent_tokens_settled = 0)`
- [ ] 6.3 修改 `reconcile.go` 的兩個 UPDATE（有/無 response_at 分支）：加入 `subagent_tokens_settled = 1`
- [ ] 6.4 寫失敗測試：Stop hook 已寫 response_at 但 subagent_tokens_settled=0 → reconcile 掃到此 turn 並重算；重算後 subagent_tokens_settled=1 → 下次 reconcile 跳過

## 7. reconcile 改用 WindowResult

- [ ] 7.1 修改 `internal/reconcile/reconcile.go`：`transcript.ExtractWindow` 改接 `(WindowResult, error)` 回傳；直接使用 `result.InputTokens` 等欄位，移除 `parseTokensJSON` 函數與 `tokenCounts` struct
- [ ] 7.2 修改兩個 UPDATE 語句：加入 `cache_creation_5m_tokens`、`cache_creation_1h_tokens`、`model`（來自 `result.CacheCreate5m`、`result.CacheCreate1h`、`result.Model`）
- [ ] 7.3 修改空值判斷：`tokensJSON == ""` → `result.InputTokens == 0 && result.OutputTokens == 0`
- [ ] 7.4 寫失敗測試：transcript 含 5m/1h cache → reconcile 後 DB 的 `cache_creation_5m_tokens` 與 `cache_creation_1h_tokens` 填入正確值

## 8. pricing 費率拆分（選用，若本 change 涵蓋）

- [ ] 8.1 修改 `internal/pricing/pricing.go` 的 `Calculate` 簽章：加入 `cacheCreate5m, cacheCreate1h int` 參數（或傳 struct），分別套用對應費率；更新所有呼叫方（reconcile、response.go）
- [ ] 8.2 寫失敗測試：已知定價 + 已知 5m/1h token 數 → `Calculate` 回傳正確成本金額

## 9. 整合驗證

- [ ] 9.1 執行 `go test ./...`，確認所有測試通過，無 regression
- [ ] 9.2 手動模擬 Stop hook 競態：啟動 session，Stop hook 在 subagent JSONL 未完成時執行 `record response` → 確認 `subagent_tokens_settled=0`；process 結束後觸發 reconcile → 確認 `subagent_tokens_settled=1`，token 值正確
- [ ] 9.3 確認 `record.go` 編譯後無未使用 import，移除 `io`/`bytes` import（已被 bufio 取代於 recorder）
