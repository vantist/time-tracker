# Fix User Time Semantics — Interval-Based Merge

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

目前 `UserActiveTime` 計算方式是 `prompt[i] - prompt[i-1]`，這包含了 agent 處理時間，導致 user time 虛高且語義不正確。另外，多個 session 並行時直接加總 user duration 會重複計算重疊時段，使總計失真。

這個工具的目的是評估：agent 實際跑了多少時間，以及使用者實際花了多少時間操作。因此 user time 的準確性對成效評比至關重要。使用者確認平時就會同時開多個 session 多工操作。

Agent time 目前 per-session sum 正確（各自 agent 實體的執行時間本身不衝突），不需修改。

## Decision

**User time 語義改為 `response_at[i-1] → prompt_at[i]`**（你看回覆、思考、打字的時間），並在所有聚合層級（總計、ByProject、ByWorkItem）做 interval merge 去重。

## Rationale

改為 response→prompt 間隔後，agent 處理時間從 user time 中移出，`agent_time + user_time ≈ 總 wall-clock`（idle 部分扣掉）。Merge 去重解決多 session 並行的重疊問題，得到真正的「人在鍵盤前的時間」。

可接受的結構性誤差：多 session 切換時的幾秒 overlap（例如看完 A 回覆後立刻跳到 B 下 prompt），無法在不追蹤 window focus 的情況下消除，YAGNI。

## Approach

**新增 interval-based API 至 `aggregator.go`：**

```go
type Interval struct{ Start, End time.Time }

// UserIntervals 從每個 turn 產生 [response_at[i-1], prompt_at[i]] 區間
// 第一個 turn：[sessionStart, prompt_at[0]]（sessionStart 為零值時略過）
func UserIntervals(turns []Turn, sessionStart time.Time) []Interval

// MergeAndSum 將 intervals 排序、合併重疊後求總和
func MergeAndSum(intervals []Interval) time.Duration
```

**`report.go` 修改三個聚合點：**

1. **總計**：per-session 各自產生 `UserIntervals`，全部收集後一次 `MergeAndSum`
2. **ByProject**：per-project 收集所有 session 的 user intervals，`MergeAndSum`
3. **groupByWorkItem**：per-group 收集所有 session 的 user intervals，`MergeAndSum`

`AgentTime` 保持不變（per-session sum）。

## Design Notes

**UserIntervals 細節：**
- 每個 turn 需要前一個 turn 的 `response_at`
- `responseAt` 為 nil（agent 還沒回應）時略過該區間
- `sessionStart` 為零值時不產生第一段初始間隔
- Idle threshold 套用於每個 interval：`End - Start >= idleThreshold` → 丟棄（不計入）

**MergeAndSum 演算法：**
```
sort intervals by Start
merged = []
for each interval:
    if merged is empty or interval.Start > merged.last.End:
        append
    else:
        merged.last.End = max(merged.last.End, interval.End)
sum = sum(End - Start for each merged interval)
```

**report.go 結構調整：**
- 在第一個 loop 結束後，建立 `sessUserIntervals map[string][]Interval`
- ByProject 和 groupByWorkItem 都收集 `sessUserIntervals[sid]` 再 merge，不再傳 merged turns

**Backward compatibility：**
- `aggregator.UserActiveTime` 可保留不動（現有測試不破壞），新增 `UserIntervals` + `MergeAndSum` 為新 API
- report.go 改用新 API，舊函數之後可清理

## Insights to Capture

- `design.md`: UserIntervals 回傳 `[response_at[i-1], prompt_at[i]]`；idle threshold 套用於單一 interval 而非 gap
- `specs/user-time/spec.md`: User time SHALL be `response_at → next_prompt_at`；多 session 並行 SHALL merge overlapping intervals
- `tasks.md`: (1) 新增 `Interval`、`UserIntervals`、`MergeAndSum` 至 aggregator.go 並補測試; (2) 修改 report.go 總計、ByProject、groupByWorkItem 三個聚合點使用新 API; (3) 移除或保留舊 `UserActiveTime`（建議保留至確認無其他呼叫方）

## Open Questions

- **完全消除 window-focus 誤差**：使用者表示後續會探討。目前先以 interval merge 為準，誤差在秒級可接受。
