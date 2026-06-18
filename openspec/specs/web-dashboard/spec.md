# web-dashboard Specification

## Purpose
TBD - created by archiving change enrich-report-and-tt-serve. Update Purpose after archive.
## Requirements
### Requirement: tt serve 啟動 HTTP server

`tt serve` subcommand SHALL 啟動本機 HTTP server，預設 port 7890，並自動開啟瀏覽器。

#### Scenario: 預設 port 啟動

- **WHEN** 使用者執行 `tt serve`
- **THEN** server 在 `localhost:7890` 監聽，終端機印出 `Serving at http://localhost:7890`，並呼叫系統指令開啟瀏覽器

#### Scenario: 自訂 port

- **WHEN** 使用者執行 `tt serve --port 8080`
- **THEN** server 在 `localhost:8080` 監聽，終端機印出 `Serving at http://localhost:8080`

#### Scenario: port 已佔用

- **WHEN** 指定 port 已被其他程序佔用
- **THEN** `tt serve` SHALL 印出明確錯誤訊息（含 port 號）並以非零 exit code 結束

### Requirement: 網頁 dashboard 顯示 Summary 卡片

`GET /` 回傳的 HTML dashboard SHALL 包含 Summary 卡片，顯示 Sessions 總數、Agent time、User active、Input/Output/Cache tokens、Est. cost。

#### Scenario: Summary 卡片正確顯示

- **WHEN** 瀏覽器請求 `GET /`
- **THEN** 回應 HTML 包含 Sessions、Agent time、User active、Input tokens、Output tokens、Cache read、Cache creation、Est. cost 等資料欄位

#### Scenario: 無資料時的 Summary 卡片

- **WHEN** DB 中無 session 且瀏覽器請求 `GET /`
- **THEN** 數值欄位顯示 `0`，cost 顯示 `$0.0000`

### Requirement: 網頁 dashboard 顯示 Daily Timeline

dashboard SHALL 包含 7 天 daily timeline，以純 CSS bar chart 呈現，橫軸為日期，bar 高度代表 session 數。

#### Scenario: 7 天 bar chart 渲染

- **WHEN** 瀏覽器請求 `GET /` 且過去 7 天有 sessions
- **THEN** dashboard 包含 7 個日期的 bar，bar 寬度/高度以 CSS `%` 表示，不使用 SVG 或 canvas

#### Scenario: 某日無 session

- **WHEN** 過去 7 天中某日無 session
- **THEN** 該日顯示高度為 0 的 bar（或空白佔位），不省略該日

### Requirement: 網頁 dashboard 顯示 By Project table

dashboard SHALL 包含 By Project table，欄位為 project、sessions、agent time、tokens、cost。

#### Scenario: By Project table 渲染

- **WHEN** 瀏覽器請求 `GET /` 且有多個 project 的 sessions
- **THEN** table 每列對應一個 project，包含 project 名稱、session 數、agent time、total tokens、est. cost

### Requirement: 網頁 dashboard 顯示 Session 明細 table

dashboard SHALL 包含 Session 明細 table，每列對應一筆 session，欄位含時間、project、branch、model、turns、agent time、cost。

#### Scenario: Session 明細 table 渲染

- **WHEN** 瀏覽器請求 `GET /` 且 DB 有 sessions
- **THEN** table 每列包含 session 開始時間（local time）、project、branch、model、turns 數、agent time（分鐘）、est. cost

### Requirement: By Work Item table 顯示 Project 欄

dashboard 的 By Work Item table SHALL 包含 Project 欄，顯示 `GroupResult.Project`（即 `path.Base(project)`），位於 Label 欄右側。

#### Scenario: By Work Item table 包含 Project 欄

- **WHEN** 瀏覽器請求 `GET /` 且 BY WORK ITEM 報表有資料
- **THEN** By Work Item table thead 包含 `Project` 欄位標題
- **THEN** 每列對應的 `<td>` 顯示該 group 的 `path.Base(project)` 值

#### Scenario: 相同 label 不同 project 顯示為獨立列

- **WHEN** 兩個 repo 均有 branch = "main" 的 sessions
- **THEN** By Work Item table 顯示兩列，Label 欄均為 "main"，Project 欄各自顯示不同的 repo 名稱

### Requirement: /api/report JSON endpoint

`GET /api/report` SHALL 回傳與 `tt report --json` 相同結構的 JSON，包含 by_project 與完整 token 欄位。

#### Scenario: JSON endpoint 回傳正確 Content-Type

- **WHEN** 瀏覽器或 curl 請求 `GET /api/report`
- **THEN** 回應 `Content-Type: application/json`，body 為合法 JSON

#### Scenario: JSON endpoint 欄位完整

- **WHEN** 請求 `GET /api/report`
- **THEN** JSON 含 `sessions`（int）、`agent_time_seconds`（int）、`input_tokens`（int）、`output_tokens`（int）、`cache_read_tokens`（int）、`cache_creation_tokens`（int）、`cost_usd`（float）、`by_project`（陣列）、`daily`（陣列，7 天）

### Requirement: 前端每 60 秒自動刷新

dashboard 前端 JavaScript SHALL 每 60 秒呼叫 `/api/report` 並更新頁面資料，不重新整理整頁。

#### Scenario: 自動刷新觸發

- **WHEN** dashboard 頁面已載入 60 秒
- **THEN** 前端 MUST 對 `/api/report` 發出 fetch 請求並更新 DOM 中的數值

#### Scenario: fetch 失敗不崩潰

- **WHEN** `/api/report` 回傳非 2xx 或網路錯誤
- **THEN** 現有 DOM 資料保持不變，console 記錄錯誤，不顯示 JS exception

