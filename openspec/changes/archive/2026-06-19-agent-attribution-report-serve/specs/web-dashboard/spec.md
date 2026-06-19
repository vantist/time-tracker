## ADDED Requirements

### Requirement: 網頁 dashboard 顯示 By Agent table

dashboard SHALL 包含 By Agent table，欄位為 agent、sessions、agent time、tokens、cost。

#### Scenario: By Agent table 渲染
- **WHEN** 瀏覽器請求 `GET /` 且有多個 agent 的 sessions
- **THEN** table 每列對應一個 agent，包含正規化後的 agent 名稱、session 數、agent time、以及 tokens 欄位（格式為格式化後的 `input_tokens / output_tokens`）、est. cost

## MODIFIED Requirements

### Requirement: 網頁 dashboard 顯示 Session 明細 table

dashboard SHALL 包含 Session 明細 table，每列對應一筆 session，欄位含時間、project、branch、agent、model、turns、agent time、cost。

#### Scenario: Session 明細 table 渲染
- **WHEN** 瀏覽器請求 `GET /` 且 DB 有 sessions
- **THEN** table 每列包含 session 開始時間（local time）、project、branch、正規化後的 agent 名稱、model、turns 數、agent time（分鐘）、est. cost

### Requirement: /api/report JSON endpoint

`GET /api/report` SHALL 回傳與 `tt report --json` 相同結構的 JSON，包含 by_project 與完整 token 欄位。

#### Scenario: JSON endpoint 回傳正確 Content-Type
- **WHEN** 瀏覽器或 curl 請求 `GET /api/report`
- **THEN** 回應 `Content-Type: application/json`，body 為合法 JSON

#### Scenario: JSON endpoint 欄位完整
- **WHEN** 請求 `GET /api/report`
- **THEN** JSON 含 `sessions`（int）、`agent_time_seconds`（int）、`input_tokens`（int）、`output_tokens`（int）、`cache_read_tokens`（int）、`cache_creation_tokens`（int）、`cost_usd`（float）、`by_project`（陣列）、`by_agent`（陣列）、`daily`（陣列，7 天）
