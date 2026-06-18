## 1. 測試（失敗先行）

- [x] 1.1 在 `internal/report/report_test.go` 新增測試：相同 branch 不同 project 應產生兩個不同 GroupResult（驗證複合 key 邏輯）
- [x] 1.2 新增測試：`GroupResult.Project` 應為 `path.Base(project)`，`GroupResult.Label` 不含 project 路徑
- [x] 1.3 新增測試：相同 work_item 不同 project 應產生兩列，`Project` 欄各自對應正確值

## 2. report.go 修改

- [x] 2.1 在 `internal/report/report.go` 的 `GroupResult` struct 新增 `Project string` 欄位
- [x] 2.2 修改 `groupByWorkItem`：新增 `projectOf map[string]string` 和 `displayOf map[string]string`，labelOf 改存複合 key（`project + "|" + label`）
- [x] 2.3 修改 `groupByWorkItem`：`groupState` 加 `project string` 欄位，建構 `GroupResult` 時帶入 `Project: path.Base(g.project)`，`Label` 使用 `displayOf[sessionID]`（不含 project）
- [x] 2.4 確認 1.1~1.3 測試通過

## 3. html.go 修改

- [x] 3.1 修改 `internal/report/html.go`：By Work Item table thead 新增 `<th>Project</th>`（位於 Label 欄右側）
- [x] 3.2 修改 By Work Item table tbody：每列新增對應 `<td>{{ .Project }}</td>`
- [x] 3.3 手動或自動測試：`tt serve` 開啟 dashboard，確認 By Work Item table 顯示 Project 欄且資料正確
