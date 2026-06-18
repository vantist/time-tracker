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

## 12. 驗收測試（原有）

- [x] 12.1 執行 `go build ./...` 確認無編譯錯誤
- [x] 12.2 執行 `go test ./internal/report/...` 確認所有新舊測試通過
- [x] 12.3 手動執行 `tt report` 確認 Tokens 區塊與 By Project 區塊出現在輸出中
- [x] 12.4 手動執行 `tt report --json | jq .` 確認 `by_project`、`cache_creation_tokens` 欄位存在
- [x] 12.5 手動執行 `tt serve` 確認終端機印出 `Serving at http://localhost:7890`，瀏覽器自動開啟，dashboard 可見四個區塊

## 13. Model 記錄修正（internal/recorder/response.go）

- [x] 13.1 在 `transcriptEntry` 的 `Message` struct 加入 `Model string \`json:"model"\`` 欄位
- [x] 13.2 在 `extractFromTranscript`（原 `extractTokensFromTranscript`）中，從最後一個非 sidechain assistant entry 取 `message.model`，連同 tokens 一起回傳（可改回傳 struct 或新增 return value）
- [x] 13.3 在 `RecordResponse` 取得 model 後，執行 `UPDATE sessions SET model=? WHERE id=? AND (model='' OR model IS NULL)`；若 model 為空字串則略過 UPDATE

## 14. 測試：Model 記錄（TDD，先寫測試）

- [x] 14.1 在 `internal/recorder/response_test.go` 加入測試：transcript JSONL 含 `message.model` 時，`sessions.model` 被正確寫入
- [x] 14.2 在 `internal/recorder/response_test.go` 加入測試：`sessions.model` 已有值時，UPDATE 不覆蓋
- [x] 14.3 在 `internal/recorder/response_test.go` 加入測試：transcript `message.model` 為空時，sessions.model 不變，tokens 仍正常記錄

## 15. Pricing Normalize 與 Table 更新（internal/pricing/pricing.go）

- [x] 15.1 加入 `normalize(model string) string`：`strings.LastIndex(model, "/")` 取後段；若無 `/` 則回傳原值
- [x] 15.2 在 `Calculate` 函式改用 `table[normalize(model)]` 查詢
- [x] 15.3 更新 pricing table：移除 `claude-haiku-4-5-20251001` key，改用 `claude-haiku-4-5`（$1.00 input）；修正 `claude-opus-4-8` 為 $5.00；新增 `claude-fable-5`、`claude-opus-4-7`、`claude-opus-4-6`、`claude-opus-4-5`、`claude-sonnet-4-5`、`claude-haiku-3-5`

## 16. 測試：Pricing Normalize（TDD，先寫測試）

- [x] 16.1 在 `internal/pricing/pricing_test.go` 加入測試：`vertex_ai/claude-sonnet-4-6` normalize 後能正確查到 pricing，回傳非 nil cost
- [x] 16.2 在 `internal/pricing/pricing_test.go` 加入測試：`claude-haiku-4-5`（無前綴）查到 $1.00/MTok 定價
- [x] 16.3 在 `internal/pricing/pricing_test.go` 加入測試：`claude-opus-4-8` 定價為 $5.00/MTok input（非舊的 $15.00）
- [x] 16.4 在 `internal/pricing/pricing_test.go` 加入測試：未知 model 回傳 nil

## 17. Report Struct 補欄位（internal/report/report.go）

- [x] 17.1 `ProjectSummary` struct 加入 `UserActiveTimeSec int64 \`json:"user_active_time_sec"\`` 欄位
- [x] 17.2 `SessionRow` struct 加入 `UserTimeSec int64 \`json:"user_time_sec"\`` 與 `WorkItem string \`json:"work_item"\`` 欄位
- [x] 17.3 在 `Query()` 的 per-project 後處理，計算 `aggregator.UserActiveTime(ps.turns, idleThreshold)` 並寫入 `ProjectSummary.UserActiveTimeSec`（`projMap` 的 turns 切片已存在）
- [x] 17.4 在 `Query()` 的 per-session 後處理，計算 `aggregator.UserActiveTime(sessTurns[sid], idleThreshold)` 並寫入 `SessionRow.UserTimeSec`；從 `sessMap[sid].workItem` 寫入 `SessionRow.WorkItem`（需在 `sessState` struct 加 `workItem string` 欄位，在 rows 迴圈中從 `r.workItem` 取值）
- [x] 17.5 `Query()` 永遠執行 `res.Groups = groupByWorkItem(allRows, sessTurns, idleThreshold)`，移除 `opts.ByWorkItem` 條件判斷
- [x] 17.6 `groupByWorkItem` 結果按 `AgentTimeSec` 降序排列（在 `return result` 前加 `sort.Slice`）
- [x] 17.7 `FormatJSON` 的 map 加入 `"groups": r.Groups`

## 18. 測試：Report Struct 補欄位（TDD，先寫測試）

- [x] 18.1 在 `internal/report/report_test.go` 加入測試：`ProjectSummary.UserActiveTimeSec` 計算正確（兩個 session 各一個 turn，驗證 user time > 0）
- [x] 18.2 在 `internal/report/report_test.go` 加入測試：`SessionRow.WorkItem` 正確回傳已設定的 work item 值
- [x] 18.3 在 `internal/report/report_test.go` 加入測試：`SessionRow.UserTimeSec` 計算正確
- [x] 18.4 在 `internal/report/report_test.go` 加入測試：`Result.Groups` 永遠非 nil（即使 `opts.ByWorkItem=false`）
- [x] 18.5 在 `internal/report/report_test.go` 加入測試：`FormatJSON` 輸出含 `groups` 陣列

## 19. Dashboard HTML 補欄位（internal/report/html.go）

- [x] 19.1 By Project table header 加入 `User time` 欄（`<th>User time</th>`，位於 Agent time 後）
- [x] 19.2 By Project table 渲染邏輯（JS）：讀 `p.user_active_time_sec`，以 `fmtTime()` 格式化後插入對應 `<td>`
- [x] 19.3 Sessions table header 加入 `User time` 與 `Work item` 欄（`<th>User time</th><th>Work item</th>`）
- [x] 19.4 Sessions table 渲染邏輯（JS）：讀 `s.user_time_sec` 與 `s.work_item`，插入對應 `<td>`
- [x] 19.5 在 By Project section 後加入 By Work Item section：HTML `<div class="section"><h2>By Work Item</h2><table id="tbl-workitem">...</table></div>`
- [x] 19.6 By Work Item table header：`Label`、`Sessions`、`Agent time`、`User time`、`Cost`
- [x] 19.7 By Work Item 渲染邏輯（JS）：遍歷 `d.groups`，每個元素插入一列；若 `d.groups` 長度 ≤ 1 則隱藏該 section（`section.style.display='none'`）
- [x] 19.8 By Work Item 渲染邏輯（JS）：`g.agent_time_sec`、`g.user_active_time_sec`、`g.estimated_cost_usd`（`fmtCost()`）、`g.sessions_count`

## 20. 驗收測試（新增功能）

- [x] 20.1 執行 `go build ./...` 確認無編譯錯誤
- [x] 20.2 執行 `go test ./...` 確認所有測試通過
- [x] 20.3 手動執行 `tt serve`，確認 By Project table 有 User time 欄
- [x] 20.4 手動執行 `tt serve`，確認 Session 明細 table 有 User time 與 Work item 欄
- [x] 20.5 手動執行 `tt work "test-feature" && tt record prompt --session test123 --project .`，再執行 `tt serve`，確認 By Work Item section 出現
- [x] 20.6 確認 `curl localhost:7890/api/report | jq '.sessions[0].user_time_sec, .sessions[0].work_item, .by_project[0].user_active_time_sec, .groups'` 均有值
