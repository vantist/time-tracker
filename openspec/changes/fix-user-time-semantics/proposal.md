## Why

目前 `UserActiveTime` 以 `prompt[i] - prompt[i-1]` 計算，涵蓋了 agent 處理時間，導致 user time 語義錯誤且數值虛高。多個 session 並行時直接加總 user duration 更會重複計算重疊時段，使總計失真。

## What Changes

- User time 語義改為 `response_at[i-1] → prompt_at[i]`（使用者看回覆、思考、輸入的時間）
- 新增 `Interval` 型別及 `UserIntervals`、`MergeAndSum` 函數至 `aggregator.go`
- `internal/report/report.go` 三個聚合點（總計、ByProject、groupByWorkItem）改用 interval-based API，並執行 merge 去重
- Idle threshold 套用於單一 interval（`End - Start >= idleThreshold` → 丟棄）而非 gap

## Capabilities

### New Capabilities

- `user-time`: Interval-based user active time 計算，定義 `response_at → next_prompt_at` 語義及多 session overlap merge 規格

### Modified Capabilities

- `time-aggregation`: User time 聚合規格由 prompt-gap 改為 interval-merge；多 session 並行時 SHALL 合併重疊區間後再加總

## Impact

- Affected specs: `user-time`（new）、`time-aggregation`（delta）
- Affected code:
  - New: `internal/aggregator/interval.go`（Interval、UserIntervals、MergeAndSum）
  - Modified: `internal/report/report.go`（三個聚合點改用新 API）
  - Removed: （無，`UserActiveTime` 保留至確認無其他呼叫方）

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-19-brainstorm-fix-user-time-semantics.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
