## MODIFIED Requirements

### Requirement: 文字報告依 Project 分組

`tt report` 的文字輸出 SHALL 在 tokens 區塊後加入 By Project 區塊，以表格形式每行顯示 project 名稱（使用 `path.Base`）、session 數、agent time、user active time、tokens 數量 (格式為 `input / output`)、以及 est. cost。

#### Scenario: 多 project 分組顯示

- **WHEN** sessions 跨多個 project
- **THEN** By Project 區塊以表格對齊顯示，欄位標頭包含 `Project`、`Sessions`、`Agent Time`、`User Active`、`Tokens (I/O)`、`Cost`
- **THEN** 每個 project 各佔一行，project 名稱只顯示資料夾基底名稱（`path.Base`），依 session 數降序排列，時間以 `Xh Ym` 格式顯示，tokens 數千分位格式化，cost 欄位顯示 `$%.4f`

#### Scenario: project 無 cost 資料

- **WHEN** 某 project 的 sessions 無 cost 欄位（NULL 或 0）
- **THEN** 該 project 的 cost 欄位顯示 `N/A`

## ADDED Requirements

### Requirement: 文字報告顯示 Daily Timeline 統計表

`tt report` 的文字輸出 SHALL 在 tokens/cost 區塊後加入 Daily 統計表格，顯示過去 7 天的每日 session 數與 token 使用量。

#### Scenario: Daily 統計表渲染

- **WHEN** 過去 7 天有 session 資料且執行 `tt report`
- **THEN** 輸出包含 `─── Daily (Last 7 Days) ───` 標題行，與包含 `Date`、`Sessions`、`Input Tokens`、`Output Tokens` 標頭的表格
- **THEN** 每行依日期升序排列，欄位對齊顯示，數字以千分位逗號格式化

### Requirement: 文字報告顯示 By Work Item 分組表

`tt report` 的文字輸出 SHALL 在 `By Project` 之後加入 By Work Item 分組表格（若有多於 1 個 work item 分組或使用者在 CLI 中指定 `--by-work-item` 參數時）。

#### Scenario: By Work Item 表格渲染

- **WHEN** 存在多個不同的 work item 分組（包含 `untagged`）或使用者指定了 `--by-work-item`
- **THEN** 輸出包含 `─── By Work Item ───` 標題行，與包含 `Work Item`、`Project`、`Sessions`、`Agent Time`、`User Active`、`Cost` 標頭的表格
- **THEN** 每列資料對齊顯示，依 Agent Time 降序排列

### Requirement: 文字報告顯示詳細 Sessions 日誌表

`tt report` 的文字輸出 SHALL 在最後加入詳細的 Sessions 日誌表格，列出查詢範圍內的所有 sessions。

#### Scenario: Sessions 日誌表渲染

- **WHEN** 查詢範圍內有 session 資料且執行 `tt report`
- **THEN** 輸出包含 `─── Sessions ───` 標題行，與包含 `Start Time`、`Project`、`Branch`、`Model`、`Turns`、`Agent Time`、`User Time`、`Work Item`、`Cost` 標頭的表格
- **THEN** 時間欄位格式化為本地時間 `YYYY-MM-DD HH:MM:SS`，project 顯示 `path.Base` 名稱，依 session 開始時間降序排列
