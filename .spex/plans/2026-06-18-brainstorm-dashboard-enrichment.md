# Dashboard 資訊補齊：Model、Cost、User Time、Work Item

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). -->

## Context

現有 `tt serve` web dashboard 有四個問題：

1. **Model 沒記錄** — DB `sessions.model` 全空。`UpsertSession` 用 `INSERT OR IGNORE`，model 欄只在第一次 insert 時寫入，但 `UserPromptSubmit` stdin 的 `model` 欄位很可能為空（或帶 `vertex_ai/` 前綴），造成永遠寫不進去。
2. **Cost = N/A** — `pricing.Calculate` 用 model 當 key，model 空 → pricing table 查無 → 回傳 nil → N/A。
3. **Work item 沒在 web 顯示** — `Options.ByWorkItem` 在 `/api/report` 固定是 `false`，Groups 永遠空。Session 明細表也沒 work_item 欄。
4. **By Project / Session 缺 user active time** — `UserActiveTimeSec` 只在 summary 卡片顯示，ProjectSummary 和 SessionRow struct 沒這欄。

## Decision

**Model 修法**：從 transcript JSONL 抽取 model（Stop hook 已讀 transcript），在 `RecordResponse` 補 `UPDATE sessions SET model=? WHERE (model IS NULL OR model='')` 更新。同時在 `pricing.Calculate` 加 normalize 去掉 `vertex_ai/`、`us.` 前綴，以支援 Vertex AI model ID。

**Work item**：session table 加 Work Item 欄（有值才顯示），同時加 By Work Item 分組 section（類似 By Project）。`/api/report` 永遠計算 Groups（不依 ByWorkItem flag）。

**User time**：ProjectSummary 和 SessionRow 加 UserActiveTimeSec。By Project table 加欄，Sessions table 加欄。

## Rationale

- Model 用 transcript 抽取：Stop hook 已有 `transcript_path`，`extractTokensFromTranscript` 已讀 JSONL，改動最小，不需要新 hook 或新 DB column。
- normalize 前綴：比改 pricing table 更有彈性（支援未來的 Vertex 部署）。
- Work item 兩處都加：session 欄讓使用者看到每個 session 的標記，分組 section 讓使用者看到時間/成本彙總。
- `ByWorkItem` 永遠計算：Web 沒有 flag 控制，永遠算比加 query param 更簡單。

## Approach

### 1. `internal/recorder/response.go`

`extractTokensFromTranscript` 改名為 `extractFromTranscript`（或拆成兩個），新增 model 抽取：

```go
// assistant entry 的 message 加 Model 欄位
type transcriptEntry struct {
    ...
    Message struct {
        Model string      `json:"model"`
        Usage usageFields `json:"usage"`
    } `json:"message"`
}
// 從最後一個 assistant entry 取 model（所有 API call 同 model）
```

`RecordResponse` 抽完 tokens 後，若 model 非空，`UPDATE sessions SET model=? WHERE id=? AND (model='' OR model IS NULL)`。

### 2. `internal/pricing/pricing.go`

Model ID 只有一種變形（gateway 層加工）：
- 前綴：`vertex_ai/claude-sonnet-4-6` → 取最後一個 `/` 之後
- haiku 不帶日期後綴（gateway 送 `claude-haiku-4-5`，不是 `claude-haiku-4-5@20251001`）

因此 pricing table key 需去掉日期，`claude-haiku-4-5-20251001` → `claude-haiku-4-5`：

struct 欄位順序：`{input, output, cacheRead, cacheCreation}`。`cacheCreation` 用 **5m Cache Writes** 價（DB 無從分辨 TTL，低估保守值）。

```go
var table = map[string]modelPricing{
    // {input, output, cacheRead, cacheCreation-5m} USD / MTok
    "claude-fable-5":    {10.00, 50.00, 1.00, 12.50},
    "claude-opus-4-8":   {5.00,  25.00, 0.50, 6.25},
    "claude-opus-4-7":   {5.00,  25.00, 0.50, 6.25},
    "claude-opus-4-6":   {5.00,  25.00, 0.50, 6.25},
    "claude-opus-4-5":   {5.00,  25.00, 0.50, 6.25},
    "claude-sonnet-4-6": {3.00,  15.00, 0.30, 3.75},
    "claude-sonnet-4-5": {3.00,  15.00, 0.30, 3.75},
    "claude-haiku-4-5":  {1.00,  5.00,  0.10, 1.25},
    "claude-haiku-3-5":  {0.80,  4.00,  0.08, 1.00},
}

func normalize(model string) string {
    // strip provider prefix: "vertex_ai/claude-sonnet-4-6" → "claude-sonnet-4-6"
    if idx := strings.LastIndex(model, "/"); idx >= 0 {
        model = model[idx+1:]
    }
    return model
}

func Calculate(model string, ...) *float64 {
    p, ok := table[normalize(model)]
    ...
}
```

`strings.LastIndex` 涵蓋所有已知前綴格式。既有錯誤修正：
- 舊 `claude-opus-4-8` 用 $15/$75（Opus 4.1 deprecated 舊價） → 修正為 $5/$25
- 舊 `claude-haiku-4-5-20251001` key 帶日期後綴 → 改 `claude-haiku-4-5`，價格從 $0.80 改 $1.00（Haiku 4.5 新價）

Open: `claude-fable-5` gateway model ID 為猜測，需驗證。

### 3. `internal/report/report.go`

```go
type ProjectSummary struct {
    ...
    UserActiveTimeSec int64  `json:"user_active_time_sec"` // 新增
    Tokens            int64  `json:"tokens"`                // 新增（input+output 合計）
}

type SessionRow struct {
    ...
    UserTimeSec int64   `json:"user_time_sec"` // 新增
    WorkItem    string  `json:"work_item"`      // 新增
}
```

`Query()` 永遠計算 Groups（`groupByWorkItem` 不依 opts.ByWorkItem）：
```go
res.Groups = groupByWorkItem(allRows, sessTurns, idleThreshold)
```

per-project user time：`projMap` 改存 `sessTurns`，最後 `aggregator.UserActiveTime(ps.turns, threshold)`。

per-session user time：`sessMap` 已有 turns，`aggregator.UserActiveTime(sessTurns[sid], threshold)` 寫進 SessionRow。

work_item：`sessMap.workItem` 新增欄位，從 `r.workItem` 取。

### 4. `internal/report/html.go`

- By Project table：`Agent time` 欄後加 `User time`
- Sessions table：加 `User time` + `Work item` 欄（model 欄已有，資料修好後自動顯示）
- 加 By Work Item section（`d.groups` array → table，欄：Label, Sessions, Agent time, User time, Cost）
- `fmtCost` 函式：由原本 `'$'+c.toFixed(4)` 改成合理精度（`$0.0234` 格式）

## Design Notes

- `extractFromTranscript` 需向後相容：若 model 欄為空（舊版 transcript），不更新 session，不影響 tokens 記錄。
- By Work Item 的 `groups` 永遠計算，但若全部 sessions 都沒有 work_item 且 branch 全是 main，會聚合成單一 "main" group（等同 no-op）。前端可判斷 groups 長度 ≤ 1 時隱藏該 section。
- `Tokens` 欄位在 ProjectSummary：`input_tokens + output_tokens`（不含 cache）作為簡易摘要，符合 By Project table 現有 "Tokens" 欄。

## Insights to Capture

- `design.md`：model 從 transcript 抽取，normalize vertex_ai/ 前綴
- `tasks.md`：response.go 改動、pricing normalize、report struct 補欄、html.go 更新

## Open Questions

- Transcript JSONL 的 `message.model` 確切欄位名稱尚未確認（需執行一次 debug 驗證）。建議在 implement 前先 `cat` 一個 transcript 確認。
- `groups` 的排序：目前 `groupByWorkItem` 回傳無序，前端或後端需排序（建議後端按 AgentTimeSec desc）。
