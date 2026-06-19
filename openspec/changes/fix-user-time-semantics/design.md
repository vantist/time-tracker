## Context

`internal/aggregator/aggregator.go` 目前以 `prompt[i] - prompt[i-1]` 計算 user active time，此方式包含 agent 處理時間，語義不正確。`internal/report/report.go` 三個聚合點（總計、ByProject、groupByWorkItem）直接加總各 session 的 user duration，多 session 並行時會重複計算重疊時段。

使用者平常同時開多個 session 多工，因此 overlap 問題為現實情境，不是邊際案例。

## Goals / Non-Goals

**Goals:**
- User time 語義改為 `response_at[i-1] → prompt_at[i]`（使用者看回覆、思考、輸入的時間）
- 多 session 並行時，各聚合層級以 interval merge 去重後再加總
- 新 API 可獨立測試，不破壞現有 `UserActiveTime` 用途

**Non-Goals:**
- 追蹤 window focus 以消除多 session 切換的秒級 overlap（YAGNI）
- 修改 AgentTime 計算（per-session sum 語義正確，不需修改）
- 清理舊 `UserActiveTime` 函數（本次保留，後續確認無呼叫方再刪）

## Decisions

### D1：新增 `Interval` 型別及兩個函數至 `aggregator.go`

```go
type Interval struct{ Start, End time.Time }

func UserIntervals(turns []Turn, sessionStart time.Time) []Interval
func MergeAndSum(intervals []Interval) time.Duration
```

**理由**：職責分離，report.go 只負責收集 intervals，aggregator.go 負責計算語義。兩個函數獨立可測。

**`UserIntervals` 細節：**
- 第一個 turn：`[sessionStart, turns[0].PromptAt]`（sessionStart 為零值時略過）
- 後續 turn i（i ≥ 1）：`[turns[i-1].ResponseAt, turns[i].PromptAt]`（`turns[i-1].ResponseAt` 為 nil 時略過）
- Idle threshold 套用於**單一 interval**：`End - Start >= idleThreshold` → 丟棄，不計入
- 只回傳有效（未被 idle 丟棄）的 intervals

**`MergeAndSum` 演算法：**
```
sort intervals by Start
merged = []
for each interval:
    if merged is empty or interval.Start > merged.last.End:
        append interval to merged
    else:
        merged.last.End = max(merged.last.End, interval.End)
sum = Σ (End - Start) for each merged interval
```

### D2：report.go 改用 interval-based 聚合

在第一個 session loop 結束後，建立 `sessUserIntervals map[string][]Interval`，儲存每個 session 的 user intervals。

三個聚合點各自收集相關 sessions 的 intervals 後呼叫 `MergeAndSum`：
1. **總計**：收集所有 session 的 intervals → `MergeAndSum`
2. **ByProject**：per-project 收集所屬 sessions 的 intervals → `MergeAndSum`
3. **groupByWorkItem**：per-group 收集所屬 sessions 的 intervals → `MergeAndSum`

**理由**：在同一層級 merge 才能正確去重。若先 per-session merge 再加總，會遺失跨 session 的重疊資訊。

### D3：新增 interval.go 獨立檔案

將 `Interval`、`UserIntervals`、`MergeAndSum` 放入 `internal/aggregator/interval.go`，測試放入 `internal/aggregator/interval_test.go`。

**理由**：不增加 `aggregator.go` 複雜度；interval 邏輯可獨立演進。

## Risks / Trade-offs

- **多 session 切換秒級 overlap**：使用者在看完 session A 回覆後立刻跳到 session B 下 prompt，兩個 sessions 的 interval 可能有數秒重疊，merge 後會計算為較長的時間。誤差在秒級，可接受 → 本次不處理，後續若需要可追蹤 window focus。
- **`UserActiveTime` 保留**：舊函數語義錯誤但保留，可能造成未來混淆 → 在函數上方加 deprecated 註記，TODO 待確認無呼叫方後刪除。
- **`sessionStart` 來源**：`SessionStart` 欄位若未記錄（零值）則略過第一段 interval，可能低估。目前 DB schema 確認有記錄，影響不大。
