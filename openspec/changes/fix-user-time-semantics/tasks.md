## 1. Interval 核心 API（TDD：先測試再實作）

- [x] 1.1 在 `internal/aggregator/interval_test.go` 撰寫 `TestUserIntervals` 測試：覆蓋正常相鄰 turn、前一 turn response_at 為 nil 略過、session_start 不為零值產生第一段、session_start 為零值略過、idle threshold 丟棄過長 interval 等情境
- [x] 1.2 在 `internal/aggregator/interval_test.go` 撰寫 `TestMergeAndSum` 測試：覆蓋無重疊加總、部分重疊合併、完全包含、三段多重重疊、空 slice 回傳 0 等情境
- [x] 1.3 建立 `internal/aggregator/interval.go`，定義 `Interval struct{ Start, End time.Time }`，實作 `UserIntervals(turns []Turn, sessionStart time.Time, idleThreshold time.Duration) []Interval`，通過 1.1 的測試
- [x] 1.4 在 `internal/aggregator/interval.go` 實作 `MergeAndSum(intervals []Interval) time.Duration`（排序、merge 重疊、加總），通過 1.2 的測試
- [x] 1.5 執行 `go test ./internal/aggregator/...` 確認全部通過

## 2. report.go 聚合點改寫（TDD：先測試再改寫）

- [x] 2.1 在 `internal/report/report_test.go` 新增整合測試：建立兩個有時間重疊的 sessions，驗證總計 UserTime 不重複計算（等於 merge 後的長度，非兩個 session user time 的加總）
- [x] 2.2 在 `internal/report/report_test.go` 新增 ByProject 整合測試：兩個 session 屬同一 project 且時間重疊，驗證 project 的 UserTime 正確 merge
- [x] 2.3 在 `internal/report/report_test.go` 新增 ByWorkItem 整合測試：兩個 session 屬同一 work item 且時間重疊，驗證 work item 的 UserTime 正確 merge
- [x] 2.4 修改 `internal/report/report.go` 第一個 session loop：建立 `sessUserIntervals map[string][]Interval`，以 `aggregator.UserIntervals(...)` 填入每個 session 的 intervals
- [x] 2.5 修改 `internal/report/report.go` 總計聚合點：收集所有 sessions 的 intervals 後呼叫 `aggregator.MergeAndSum`，取代直接加總 UserActiveTime
- [x] 2.6 修改 `internal/report/report.go` ByProject 聚合點：per-project 收集所屬 sessions 的 intervals，呼叫 `aggregator.MergeAndSum`
- [x] 2.7 修改 `internal/report/report.go` groupByWorkItem 聚合點：per-group 收集所屬 sessions 的 intervals，呼叫 `aggregator.MergeAndSum`
- [x] 2.8 執行 `go test ./internal/report/...` 確認 2.1–2.3 的測試通過

## 3. 舊 API 標記與驗收

- [x] 3.1 在 `internal/aggregator/aggregator.go` 的 `UserActiveTime` 函數上方加 `// Deprecated: use UserIntervals + MergeAndSum instead.` 註記
- [x] 3.2 執行 `go build ./...` 確認無編譯錯誤
- [x] 3.3 手動執行 `tt report` 並確認輸出的 UserTime 數值合理（小於舊計算結果，因 agent 處理時間已移除）
