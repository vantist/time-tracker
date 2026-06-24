## MODIFIED Requirements

### Requirement: Agent 名稱正規化

系統 SHALL 對轉入的 Agent (Tool) 名稱進行正規化處理，以統一呈現在報表與 Dashboard 中。正規化規則如下：
- 清除前後空白字元並轉為小寫。
- 如果名稱為空字元，則正規化為 `"unknown"`。
- 如果名稱為 `"claude-code"`、`"claudecode"` 或 `"claude"`，正規化為 `"Claude Code"`。
- 如果名稱為 `"copilot-cli"`、`"copilotcli"` 或 `"copilot"`，正規化為 `"Copilot CLI"`。
- 如果名稱為 `"opencode"`，正規化為 `"OpenCode"`。
- 其餘名稱保留其小寫及清除空白後的字串。

#### Scenario: Claude Code 正常正規化
- **WHEN** 傳入 `"claude-code"`、`"ClaudeCode"` 或 `"claude"`
- **THEN** 回傳 `"Claude Code"`

#### Scenario: Copilot CLI 正常正規化
- **WHEN** 傳入 `"copilot-cli"`、`"CopilotCli"` 或 `"copilot"`
- **THEN** 回傳 `"Copilot CLI"`

#### Scenario: OpenCode 正常正規化
- **WHEN** 傳入 `"opencode"` 或 `"OpenCode"`
- **THEN** 回傳 `"OpenCode"`

#### Scenario: 空值或空白處理
- **WHEN** 傳入 `""` 或 `"   "`
- **THEN** 回傳 `"unknown"`

#### Scenario: 其他自訂 Agent 名稱處理
- **WHEN** 傳入 `"My-Custom-Agent  "`
- **THEN** 回傳 `"my-custom-agent"`
