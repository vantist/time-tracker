# agent-attribution Specification

## Purpose
TBD - created by archiving change agent-attribution-report-serve. Update Purpose after archive.
## Requirements
### Requirement: Agent 名稱正規化

系統 SHALL 對轉入的 Agent (Tool) 名稱進行正規化處理，以統一呈現在報表與 Dashboard 中。正規化規則如下：
- 清除前後空白字元並轉為小寫。
- 如果名稱為空字元，則正規化為 `"unknown"`。
- 如果名稱為 `"claude-code"`、`"claudecode"` 或 `"claude"`，正規化為 `"Claude Code"`。
- 如果名稱為 `"copilot-cli"`、`"copilotcli"` 或 `"copilot"`，正規化為 `"Copilot CLI"`。
- 其餘名稱保留其小寫及清除空白後的字串。

#### Scenario: Claude Code 正常正規化
- **WHEN** 傳入 `"claude-code"`、`"ClaudeCode"` 或 `"claude"`
- **THEN** 回傳 `"Claude Code"`

#### Scenario: Copilot CLI 正常正規化
- **WHEN** 傳入 `"copilot-cli"`、`"CopilotCli"` 或 `"copilot"`
- **THEN** 回傳 `"Copilot CLI"`

#### Scenario: 空值或空白處理
- **WHEN** 傳入 `""` 或 `"   "`
- **THEN** 回傳 `"unknown"`

#### Scenario: 其他自訂 Agent 名稱處理
- **WHEN** 傳入 `"My-Custom-Agent  "`
- **THEN** 回傳 `"my-custom-agent"`

### Requirement: 依 Agent 分組統計彙整

系統 SHALL 依據正規化後的 Agent 名稱將資料進行分組，並針對各分組計算其 Sessions 數量、Agent time、User Active Time、Token 數量與預估成本。
其中各 Agent 的 User Active Time 必須基於該 Agent 的 session 時間範圍獨立執行時間區間合併與加總（不與其他 Agent 的時間重疊合併）。

#### Scenario: 多個 Agent 數據分組彙整與時間獨立計算
- **WHEN** 系統中存在複數個 session 且包含 `"Claude Code"` 與 `"Copilot CLI"` 兩種 Agent
- **THEN** 產生的彙整結果中，兩者的統計數據（Sessions 數量、Agent time、Tokens、Cost）均正確分開累計，且其 User Active Time 分別獨立計算，不受另一方時間區間重疊影響

