## MODIFIED Requirements

### Requirement: SetupCommandAutoDetection
系統在執行 `tt setup` 且未帶有任何 flag 時，SHALL 自動偵測使用者家目錄（`HOME`）下是否存在各 AI 工具的設定目錄。若偵測到對應目錄存在，系統 SHALL 自動為該工具設定 hooks，不需使用者手動傳入 flag。

具體偵測規則如下：
- 若存在 `~/.claude` 目錄，則自動設定 Claude Code hooks
- 若存在 `~/.copilot` 目錄，則自動設定 GitHub Copilot CLI hooks
- 若存在 `~/.gemini` 目錄，則自動設定 Google Antigravity hooks
- 若存在 `~/.codex` 目錄，則自動設定 OpenAI Codex hooks
- 若存在 `~/.config/opencode` 目錄，則自動設定 OpenCode plugin

#### Scenario: 自動偵測到部分工具存在並設定
- **WHEN** 呼叫 `tt setup`（無參數），且家目錄下僅存在 `~/.claude` 與 `~/.gemini` 目錄
- **THEN** 系統自動設定 Claude Code 與 Google Antigravity 的 hooks
- **THEN** stdout 輸出 `Claude Code hooks configured in ~/.claude/settings.json` 與 `Google Antigravity hooks configured in ~/.gemini/config/hooks.json`

#### Scenario: 自動偵測到 opencode 並設定 plugin
- **WHEN** 呼叫 `tt setup`（無參數），且家目錄下僅存在 `~/.config/opencode` 目錄
- **THEN** 系統自動設定 OpenCode plugin（產生 `~/.config/opencode/plugins/tt-bridge.ts`）
- **THEN** stdout 輸出 `OpenCode plugin configured in ~/.config/opencode/plugins/tt-bridge.ts`

### Requirement: SetupCommandNoToolWarning
系統在執行 `tt setup` 且未帶有任何 flag 時，若未偵測到任何適用工具的設定目錄，SHALL 輸出友善提示訊息並結束，SHALL NOT 修改任何檔案。

#### Scenario: 未偵測到任何適用工具時輸出提示
- **WHEN** 呼叫 `tt setup`（無參數），且家目錄下不存在 `~/.claude`, `~/.copilot`, `~/.gemini`, `~/.codex`, `~/.config/opencode` 中的任何一個目錄
- **THEN** stdout 輸出 `No supported AI tools detected...` 提示訊息
