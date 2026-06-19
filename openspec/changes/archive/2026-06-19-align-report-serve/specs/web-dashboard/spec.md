## MODIFIED Requirements

### Requirement: 網頁 dashboard 顯示 By Project table

dashboard SHALL 包含 By Project table，欄位為 project、sessions、agent time、tokens、cost。

#### Scenario: By Project table 渲染

- **WHEN** 瀏覽器請求 `GET /` 且有多個 project 的 sessions
- **THEN** table 每列對應一個 project，包含 project 名稱、session 數、agent time、以及 tokens 欄位（格式為格式化後的 `input_tokens / output_tokens`）、est. cost
