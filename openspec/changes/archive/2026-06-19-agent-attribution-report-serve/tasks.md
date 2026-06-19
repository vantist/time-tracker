## 1. Core Data Structures & Normalization (TDD)

- [x] 1.1 在 `internal/report/report_test.go` 中為 `normalizeAgentName` 與資料結構新增單元測試。
- [x] 1.2 在 `internal/report/report.go` 中實作 `normalizeAgentName` 函數，並新增 `AgentSummary` 結構體，同時擴充 `SessionRow` 與 `rowData` 的欄位。

## 2. SQL Query & Data Loading (TDD)

- [x] 2.1 在 `internal/report/report_test.go` 中更新測試，驗證 SQL 載入時能正確讀取 tool 欄位。
- [x] 2.2 在 `internal/report/report.go` 中擴充 SQL 查詢，將 `COALESCE(s.tool, '')` 載入並 scan 至 `SessionRow` 與 `rowData`。

## 3. Aggregation Logic (TDD)

- [x] 3.1 在 `internal/report/report_test.go` 中撰寫分組彙整與獨立 User Active Time 計算的測試案例。
- [x] 3.2 在 `internal/report/report.go` 中實作 By Agent 的分組統計邏輯，各 Agent 獨立計算 User Active Time。

## 4. CLI Text Formatting (TDD)

- [x] 4.1 在 `internal/report/report_test.go` 中新增測試，驗證文字格式化輸出包含 `Agent` 欄位與 `─── By Agent ───` 表格。
- [x] 4.2 在 `internal/report/report.go` 中更新 `FormatText` 實作，顯示 Agent 明細與 `By Agent` 統計表格。

## 5. Web Dashboard & JSON Endpoint (TDD)

- [x] 5.1 在 `internal/report/html_test.go` 中新增測試，驗證 JSON 包含 `by_agent`，且 HTML 模板包含對應渲染區塊。
- [x] 5.2 在 `internal/report/html.go` 中調整 `dashboardHTML`、CSS 以及 JavaScript 渲染邏輯，於儀表板中呈現 By Agent 資訊與 Sessions Agent 欄位。

## 6. Integration Verification

- [x] 6.1 本地執行 `tt report` 與 `tt serve` 進行手動整合測試，驗證輸出與頁面顯示正常。
