## Context

`tt` 透過三條路徑記錄 token：
1. **Prompt hook** → `record prompt` → 寫 turn 列（無 token）
2. **Stop hook** → `record response` → `RecordResponse` 寫 `response_at` + token
3. **Reconcile** → 掃描 `response_at IS NULL OR input_tokens IS NULL` 的 turn → 補算

現有問題：

- Stop hook 在 subagent JSONL 未完全 flush 前執行，寫入部分（或僅 main）token → `input_tokens` 非 NULL → reconcile WHERE 跳過 → subagent token 永遠遺漏（Bug D）
- `extractSubagentTokens` 掃描範圍為 `[offset, EOF]`，Turn 1 會把 Turn 2+ 的 Agent tool_use ID 全部吸收（Bug A）
- `record.go` 與 `internal/transcript` 分別定義相同 struct 與邏輯，兩者不同步（Bug E）
- `countLines` 用 `io.ReadAll` 讀整個 transcript，大 session 每次 prompt hook 都耗費大量 I/O（Bug F）
- cache 費率 5m/1h 不同但統一存入 `cache_creation_tokens`，定價無法區分（Data Gap）
- turns 未記錄 per-turn model，切換模型時計費取 session model 造成誤差（Data Gap）

## Goals / Non-Goals

**Goals:**
- 修正 subagent token 競態與重複計算（Bug A、Bug D）
- 統一 transcript 解析邏輯，消除 `record.go` 重複 code（Bug E）
- 修正 `countLines` 效能（Bug F）
- 記錄 `cache_creation_5m_tokens`、`cache_creation_1h_tokens`（階段 2）
- 記錄 per-turn model（階段 2）
- `ExtractWindow` 回傳 typed struct（架構清理）

**Non-Goals:**
- User prompt 文字儲存
- Subagent 開始/結束時間統計
- `server_tool_use`（web_search/web_fetch）計費

## Decisions

### Decision 1：Bug D — 以 `subagent_tokens_settled` 欄位追蹤結算狀態

**選項 A（採用）**：新增 `subagent_tokens_settled BOOL DEFAULT 0`。Stop hook 寫 token 時設為 0；reconcile 確認 process 結束後重新從 transcript 提取（含 subagent），成功後設為 1。reconcile WHERE 加 `OR subagent_tokens_settled=0`。

**選項 B（棄）**：Stop hook 完全不寫 subagent token，只寫 main window。需另設 flag 或修改 reconcile 邏輯判斷「是否需要補 subagent」，反而讓 Stop hook 與 reconcile 的責任邊界更模糊。

**理由**：選項 A 讓 Stop hook 保留 fast path（快速寫入已知的 main token），reconcile 負責最終正確性，責任分離清楚。`subagent_tokens_settled=0` 是明確的「待重算」訊號，不需要複雜的條件推導。

### Decision 2：Bug A — `extractSubagentTokens` 加 `to` 參數

`ExtractWindow` 已計算 `end = min(to, len(all))`；將此 `end` 作為 `extractSubagentTokens` 的上界，限制 Agent tool_use ID 的收集範圍至 `[from, end)`。

不需要修改 `sumSubagentWindow`（subagent 自身的 JSONL 掃全 file 是正確的，subagent JSONL 只含該 agent 的 entries）。

### Decision 3：Bug E — 新增 `transcript.ExtractLastTurn`，刪 `record.go` 重複 code

`internal/transcript/extract.go` 新增 exported `ExtractLastTurn(path string) (WindowResult, error)`，封裝「找最後 user entry + /clear fallback + subagent 合計」邏輯（即現行 `record.go:extractFromTranscript` 的行為）。

`record.go` 的 `extractFromTranscript` 直接呼叫 `transcript.ExtractLastTurn`，並刪除所有重複 type 定義。`extractFromTranscriptAtOffset` 改呼叫 `transcript.ExtractWindow`（已存在）。

### Decision 4：typed struct `WindowResult`

```go
type WindowResult struct {
    InputTokens         int
    OutputTokens        int
    CacheReadTokens     int
    CacheCreationTokens int    // 總 cache creation（向後相容）
    CacheCreate5m       int    // ephemeral_5m_input_tokens
    CacheCreate1h       int    // ephemeral_1h_input_tokens
    Model               string
}
```

`ExtractWindow` 與 `ExtractLastTurn` 均回傳 `(WindowResult, error)`。`reconcile.go` 直接存取欄位，移除 `parseTokensJSON`。`record.go` 在需要傳 JSON 給 `RecordResponse` 時自行 marshal（或同步修改 `RecordResponse` 接 struct）。

**選項 B（棄）**：沿用 JSON string。維護 `parseTokensJSON`，新欄位需改 marshal/unmarshal key，跨 package 邊界傳字串是不必要的間接層。

### Decision 5：cache 5m/1h DB 欄位

```sql
cache_creation_5m_tokens  INTEGER
cache_creation_1h_tokens  INTEGER
```

透過 `addTurnColumns` 既有模式（PRAGMA table_info + ALTER TABLE）新增。`pricing.Calculate` 簽章改為接收 `cacheCreate5m, cacheCreate1h int`，分別套用對應費率；`cacheCreationTokens`（總計）仍保留作報表加總用。

### Decision 6：per-turn model DB 欄位

```sql
model TEXT  -- turns 表
```

`RecordResponse` 的 UPDATE 加 `model=?`（從 `sessionModel` 取值，或從 Stop hook 傳入的 `model` 參數取值）。Reconcile 在 `transcript.ExtractWindow` 回傳 `WindowResult.Model` 時一併寫入。

## Risks / Trade-offs

- **現有 turn 的 `subagent_tokens_settled`** → `ALTER TABLE` 加欄位 DEFAULT 0，所有舊 turn 視為「未結算」，下次 reconcile 會嘗試重算。若 process 已不在、transcript 仍存在，可以重算（正確）；若 transcript 不在，reconcile 跳過（維持現狀）。風險低。
- **`ExtractWindow` 改回傳 struct** → `reconcile.go` 與 `record.go` 兩個呼叫方都要同步改，為多檔案修改。但是一次性改完，之後介面更清晰。
- **`pricing.Calculate` 簽章改變** → 只有 `reconcile.go` 與 `recorder/response.go` 呼叫，影響有限。

## Migration Plan

1. `addTurnColumns` 加 `subagent_tokens_settled`、`cache_creation_5m_tokens`、`cache_creation_1h_tokens`、`model`（turns 表）
2. 所有欄位 nullable / DEFAULT 0 — 向後相容，舊資料不需要 backfill
3. 下次 `tt report` 或任意 reconcile 觸發，舊 turn 的 `subagent_tokens_settled=0` 列若 process 已死會被重算；新欄位為 NULL 直到下次重算

## Open Questions

- `RecordResponse` 目前接 `tokensJSON string`；若改接 `WindowResult` struct，需修改 `recorder/response.go` 與所有呼叫方（`record.go`）。是否在此 change 一次改完，或維持 JSON 傳遞？**決定：維持 JSON string，`record.go` 呼叫方自行 marshal；减少 recorder package 介面破壞。**
- `pricing.Calculate` 若加 `cacheCreate5m/1h` 參數，舊呼叫方（測試）需同步更新。**可接受，影響範圍明確。**
