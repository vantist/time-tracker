## MODIFIED Requirements

### Requirement: SetupCommandAutoDetection
系統在執行 `tt setup` 且未帶有任何 flag 時，SHALL 自動偵測使用者家目錄（`HOME`）下是否存在各 AI 工具的設定目錄。若偵測到對應目錄存在，系統 SHALL 自動為該工具設定 hooks，不需使用者手動傳入 flag。

具體偵測規則如下：
- 若存在 `~/.claude` 目錄，則自動設定 Claude Code hooks
- 若存在 `~/.copilot` 目錄，則自動設定 GitHub Copilot CLI hooks
- 若存在 `~/.gemini` 目錄，則自動設定 Google Antigravity hooks
- 若存在 `~/.codex` 目錄，則自動設定 OpenAI Codex hooks
- 若存在 VS Code 且已安裝 GitHub Copilot Chat 擴充套件，則自動設定 VS Code Copilot bridge

#### Scenario: 自動偵測到部分工具存在並設定
- **WHEN** 呼叫 `tt setup`（無參數），且家目錄下僅存在 `~/.claude` 與 `~/.gemini` 目錄
- **THEN** 系統自動設定 Claude Code 與 Google Antigravity 的 hooks
- **THEN** stdout 輸出 `Claude Code hooks configured in ~/.claude/settings.json` 與 `Google Antigravity hooks configured in ~/.gemini/config/hooks.json`

#### Scenario: 自動偵測到 VS Code Copilot 並設定 bridge
- **WHEN** 呼叫 `tt setup`（無參數），且偵測到 VS Code 已安裝 GitHub Copilot Chat 擴充套件
- **THEN** 系統自動安裝 VS Code Copilot bridge extension
- **THEN** stdout 輸出 `VS Code Copilot bridge installed`

### Requirement: SetupCommandNoToolWarning
系統在執行 `tt setup` 且未帶有任何 flag 時，若未偵測到任何適用工具的設定目錄，SHALL 輸出友善提示訊息並結束，SHALL NOT 修改任何檔案。

#### Scenario: 未偵測到任何適用工具時輸出提示
- **WHEN** 呼叫 `tt setup`（無參數），且家目錄下不存在 `~/.claude`, `~/.copilot`, `~/.gemini`, `~/.codex` 中的任何一個目錄
- **THEN** stdout 輸出 `No supported AI tools detected...` 提示訊息

## ADDED Requirements

### Requirement: SetupCommandVSCodeCopilot
系統 SHALL 支援 `tt setup --vscode-copilot` flag，用於安裝 VS Code Copilot bridge extension。

#### Scenario: 手動安裝 VS Code Copilot bridge
- **WHEN** 呼叫 `tt setup --vscode-copilot`
- **THEN** 系統安裝 VS Code Copilot bridge extension 到 VS Code
- **THEN** stdout 輸出 `VS Code Copilot bridge installed`

#### Scenario: VS Code 未安裝時警告
- **WHEN** 呼叫 `tt setup --vscode-copilot`，但 VS Code 未安裝
- **THEN** 系統輸出警告訊息 `VS Code not found, skipping VS Code Copilot bridge installation`
- **THEN** 不修改任何檔案
