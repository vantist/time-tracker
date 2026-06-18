## ADDED Requirements

### Requirement: 文字報告顯示完整 token 統計

`tt report` 的文字輸出 SHALL 在「Sessions / Agent time / User active」之後加入 Tokens 區塊，包含 Input、Output、Cache read、Cache creation 四個欄位，以及 Est. cost 欄位。

#### Scenario: 基本 token 區塊顯示

- **WHEN** 使用者執行 `tt report`（或帶 `--since`）
- **THEN** 輸出 MUST 包含 `─── Tokens ─────` 標題行，以及 `Input:`、`Output:`、`Cache read:`、`Cache create:` 四行，數字以千分位逗號格式化

#### Scenario: 無 session 時的 token 欄位

- **WHEN** DB 中無符合條件的 session
- **THEN** 四個 token 欄位 SHALL 顯示 `0`，Est. cost 顯示 `$0.0000`

### Requirement: 文字報告依 Project 分組

`tt report` 的文字輸出 SHALL 在 tokens 區塊後加入 By Project 區塊，每行顯示 project 名稱、session 數、agent time、est. cost。

#### Scenario: 多 project 分組顯示

- **WHEN** sessions 跨多個 project
- **THEN** By Project 區塊每個 project 各佔一行，格式為 `<project>  <sessions>  <agent_time>  <cost>`，依 session 數降序排列

#### Scenario: project 無 cost 資料

- **WHEN** 某 project 的 sessions 無 cost 欄位（NULL 或 0）
- **THEN** 該 project 的 cost 欄位顯示 `N/A`

### Requirement: JSON 報告補齊完整欄位

`tt report --json` 輸出 SHALL 包含 `output_tokens`、`cache_read_tokens`、`cache_creation_tokens`、`by_project`（陣列）欄位。

#### Scenario: JSON 輸出包含 by_project 陣列

- **WHEN** 使用者執行 `tt report --json`
- **THEN** JSON 根物件 MUST 含 `by_project` 陣列，每個元素含 `project`（string）、`sessions`（int）、`agent_time_seconds`（int）、`cost_usd`（float，可為 null）

#### Scenario: JSON token 欄位完整

- **WHEN** 使用者執行 `tt report --json`
- **THEN** JSON 根物件 MUST 含 `output_tokens`（int）、`cache_read_tokens`（int）、`cache_creation_tokens`（int）
