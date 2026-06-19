## MODIFIED Requirements

### Requirement: 基本報表輸出

系統 SHALL 透過 `tt report` 命令輸出過去 7 天（預設）的聚合統計，格式為純文字，包含：
- Sessions 數量
- Agent 時間（h m 格式）
- User 主動時間（h m 格式，含使用的 idle threshold）
- Token 總量（input、output、cache hit 比例）
- 預估成本（USD）
- Sessions 表格中包含 `Agent` 欄位（對齊欄寬 `%-12s`）
- 新增 `─── By Agent ───` 統計表格，顯示各 Agent 獨立計算的 User 主動時間與成本。

#### Scenario: 無資料時顯示空報表而非錯誤
- **WHEN** `tt report` 被呼叫，且資料庫中 7 天內無任何 turns
- **THEN** 輸出 "No data for the selected period."
- **THEN** exit code 0

#### Scenario: 有資料時輸出格式正確
- **WHEN** 7 天內有資料，`tt report` 被呼叫
- **THEN** stdout 輸出包含 "Sessions:", "Agent time:", "User active:", "Tokens in:", "Est. cost:" 等欄位
- **THEN** Agent time 格式為 `Xh Ym`（如 `2h 34m`）
- **THEN** stdout 輸出包含 "─── By Agent ───" 區塊，列出各 Agent 名稱、User time、Cost
- **THEN** stdout 的 Sessions 表格中包含 "Agent" 欄位標題，且對應列處顯示該 session 正規化後的 Agent 名稱
