## 1. Result Struct 與 Query 補齊欄位

- [x] 1.1 在 `internal/report/report.go` 的 `Result` struct 加入 `CacheCreationTokens int64` 欄位
- [x] 1.2 在 `Result` struct 加入 `ByProject []ProjectSummary` 欄位，並在同檔定義 `ProjectSummary struct { Project string; SessionsCount int; AgentTimeSec int64; CostUSD *float64 }`
- [x] 1.3 在 `internal/report/report.go` 的 SQL query 加入 `COALESCE(t.cache_creation_tokens, 0)` 欄位，對應現有 SELECT 順序
- [x] 1.4 在 `rowData` struct 加入 `CacheCreationTokens int64` 欄位，更新對應的 `rows.Scan(...)` 呼叫
- [x] 1.5 在 rows 後處理邏輯累積 `CacheCreationTokens` 到 `Result`（同 InputTokens 的加總方式）
- [x] 1.6 在 rows 後處理邏輯按 `session.project` 分組累積 ProjectSummary，產生 `Result.ByProject`（依 SessionsCount 降序排列）

## 2. 測試：Result Struct 與 Query（TDD，先寫測試）

- [x] 2.1 在 `internal/report/report_test.go` 加入測試：Query 回傳的 Result.CacheCreationTokens 加總正確（使用現有測試 DB 設置方式插入含 cache_creation_tokens 的 turns）
- [x] 2.2 在 `internal/report/report_test.go` 加入測試：Result.ByProject 正確分組並依 sessions 數降序排列
- [x] 2.3 在 `internal/report/report_test.go` 加入測試：project 無 cost 資料時 ProjectSummary.CostUSD 為 nil

## 3. FormatText 改版

- [x] 3.1 在 `internal/report/report.go` 的 `FormatText` 函式，加入 Tokens 區塊（`─── Tokens ─────...`），依序顯示 Input、Output、Cache read、Cache create，數字用 `humanize` 或 `fmt.Sprintf("%s", formatInt(n))` 千分位格式（先看現有格式方式，沿用已有 helper；若無則實作 `formatInt(n int64) string` 用 `strconv` + 手動插逗號）
- [x] 3.2 在 FormatText 加入 Cost 區塊（`─── Cost ────...`），顯示 Est. cost 行
- [x] 3.3 在 FormatText 加入 By Project 區塊（`─── By Project ──...`），每個 ProjectSummary 各一行，格式 `  <project>  <sessions>  <agentH>h <agentM>m  <cost|N/A>`，欄寬對齊

## 4. 測試：FormatText（TDD，先寫測試）

- [x] 4.1 在 `internal/report/report_test.go` 加入測試：FormatText 輸出含 `─── Tokens` 行，且 Input/Output/Cache read/Cache create 值正確
- [x] 4.2 在 `internal/report/report_test.go` 加入測試：FormatText 輸出含 `─── By Project` 行，project 名稱與 cost 正確
- [x] 4.3 在 `internal/report/report_test.go` 加入測試：project CostUSD 為 nil 時顯示 `N/A`

## 5. FormatJSON 補齊欄位

- [x] 5.1 在 `internal/report/report.go` 的 `FormatJSON` 函式，map 中加入 `"cache_creation_tokens": r.CacheCreationTokens`、`"cache_read_tokens": r.CacheReadTokens`、`"by_project": r.ByProject`

## 6. 測試：FormatJSON（TDD，先寫測試）

- [x] 6.1 在 `internal/report/report_test.go` 加入測試：FormatJSON 輸出 JSON 含 `cache_creation_tokens`、`cache_read_tokens`、`by_project` 陣列

## 7. Daily breakdown 後處理

- [x] 7.1 定義 `DailyStat struct { Date string; Sessions int; InputTokens int64; OutputTokens int64 }` 於 `internal/report/report.go`，加入 `Result.Daily []DailyStat`
- [x] 7.2 在 rows 後處理邏輯按 `DATE(prompt_at)` 分組累積 DailyStat（取 `prompt_at` 的 UTC 日期字串 `YYYY-MM-DD`），保留最近 7 天（由 `opts.Since` 決定起始，後處理過濾）

## 8. 測試：Daily breakdown（TDD，先寫測試）

- [x] 8.1 在 `internal/report/report_test.go` 加入測試：Result.Daily 按日期排序，某日無 session 時該日不出現在陣列中（前端補零）

## 9. HTML Dashboard（internal/report/html.go）

- [x] 9.1 建立 `internal/report/html.go`，定義 `const dashboardHTML string`，包含完整 HTML/CSS/JS inline template（使用 Go `text/template` 語法，資料從 `/api/report` JSON fetch）
- [x] 9.2 實作 Summary 卡片區塊：顯示 Sessions、Agent time、User active、Input/Output/Cache read/Cache creation tokens、Est. cost
- [x] 9.3 實作 Daily timeline 區塊：7 天 bar chart，純 CSS `div` 寬度/高度 `%`，橫軸日期，縱軸 sessions count；無 session 日顯示空 bar（高度 0）
- [x] 9.4 實作 By Project table 區塊：欄位 project、sessions、agent time、total tokens、cost
- [x] 9.5 實作 Session 明細 table 區塊：欄位時間（local time）、project、branch、model、turns、agent time（分鐘）、cost
- [x] 9.6 實作前端 JS：`setInterval` 每 60s fetch `/api/report`，更新 DOM 數值，fetch 失敗只 `console.error` 不拋 exception
- [x] 9.7 在 `internal/report/html.go` 實作 `ServeHTTP` handler（或 `HandleDashboard(w, r)` 函式），`GET /` 回傳 `dashboardHTML`（Content-Type text/html）
- [x] 9.8 在 `internal/report/html.go` 實作 `HandleAPIReport(db *sql.DB, opts Options) http.HandlerFunc`，呼叫 `Query` + `json.Marshal(result)`，回傳 Content-Type application/json

## 10. 測試：HTML handler（TDD，先寫測試）

- [x] 10.1 在 `internal/report/html_test.go` 加入測試：`HandleDashboard` 回應 HTTP 200，Content-Type 含 `text/html`，body 含 `<html>`
- [x] 10.2 在 `internal/report/html_test.go` 加入測試：`HandleAPIReport` 回應 HTTP 200，Content-Type 含 `application/json`，body 可 JSON unmarshal 且含 `by_project`、`daily` 欄位

## 11. tt serve subcommand（cmd/tt/serve_cmd.go）

- [x] 11.1 建立 `cmd/tt/serve_cmd.go`，定義 `serveCmd *cobra.Command`，flags：`--port int`（預設 7890）、`--since string`（預設 `7d`，與 report 一致）
- [x] 11.2 在 `serveCmd.RunE` 中：解析 `--since`、建立 HTTP mux（`http.NewServeMux()`）、掛 `GET /` → `HandleDashboard`、`GET /api/report` → `HandleAPIReport`
- [x] 11.3 印出 `Serving at http://localhost:<port>` 後呼叫 `openBrowser("http://localhost:<port>")`
- [x] 11.4 呼叫 `http.ListenAndServe`；若 err != nil 印出含 port 號的錯誤並 `return err`
- [x] 11.5 實作 `openBrowser(url string)` 函式：`switch runtime.GOOS { case "darwin": exec.Command("open", url) case "linux": exec.Command("xdg-open", url) default: exec.Command("cmd", "/c", "start", url) }`，啟動後不等待（`cmd.Start()`）
- [x] 11.6 在 `cmd/tt/main.go`（或 root command 初始化處）將 `serveCmd` 加入 root command

## 12. 驗收測試

- [ ] 12.1 執行 `go build ./...` 確認無編譯錯誤
- [ ] 12.2 執行 `go test ./internal/report/...` 確認所有新舊測試通過
- [ ] 12.3 手動執行 `tt report` 確認 Tokens 區塊與 By Project 區塊出現在輸出中
- [ ] 12.4 手動執行 `tt report --json | jq .` 確認 `by_project`、`cache_creation_tokens` 欄位存在
- [ ] 12.5 手動執行 `tt serve` 確認終端機印出 `Serving at http://localhost:7890`，瀏覽器自動開啟，dashboard 可見四個區塊
