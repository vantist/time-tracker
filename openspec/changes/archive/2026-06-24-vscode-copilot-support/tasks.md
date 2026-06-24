## 1. Session Parsing — transcripts JSONL

- [x] 1.1 建立 `internal/transcript/vscode_copilot.go`，定義 VSCodeCopilotProvider struct 和 LogProvider 介面
- [x] 1.2 實作 transcripts JSONL 解析器：解析 session.start 事件（sessionId, startTime, copilotVersion, vscodeVersion）
- [x] 1.3 實作 transcripts JSONL 解析器：解析 user.message 事件（content, timestamp, parentId）
- [x] 1.4 實作 transcripts JSONL 解析器：解析 assistant.message 事件（content, toolRequests, reasoningText, messageId, timestamp）
- [x] 1.5 實作 transcripts JSONL 解析器：解析 tool.execution_start/complete 事件（toolCallId, toolName, arguments, success, timestamps）
- [x] 1.6 實作 malformed JSON 行的容錯處理（跳過无效行，繼續處理）
- [x] 1.7 撰寫 transcripts JSONL 解析器的單元測試

## 2. Session Parsing — chatSessions JSON

- [x] 2.1 實作 chatSessions JSON 解析器：提取 modelId 和 details（model 名稱）
- [x] 2.2 實作 chatSessions JSON 解析器：提取 thinking.tokens（thinking token 數）
- [x] 2.3 實作 chatSessions JSON 解析器：提取 session metadata（requesterUsername, responderUsername, initialLocation）
- [x] 2.4 撰寫 chatSessions JSON 解析器的單元測試

## 3. Session Parsing — debug-logs JSONL

- [x] 3.1 實作 debug-logs JSONL 解析器：解析 llm_request 事件（inputTokens, outputTokens, cachedTokens, model）
- [x] 3.2 實作 debug-logs JSONL 解析器：解析 session.shutdown 事件（per-model usage, totalNanoAiu）
- [x] 3.3 實作 debug log 目錄不存在時的 nil 回傳
- [x] 3.4 撰寫 debug-logs JSONL 解析器的單元測試

## 4. Session Discovery

- [x] 4.1 實作 workspaceStorage 路徑 discovery：macOS 路徑 `~/Library/Application Support/Code/User/workspaceStorage/*/GitHub.copilot-chat/`
- [x] 4.2 實作 workspaceStorage 路徑 discovery：Linux 路徑 `~/.config/Code/User/workspaceStorage/*/GitHub.copilot-chat/`
- [x] 4.3 實作 workspaceStorage 路徑 discovery：Windows 路徑 `%APPDATA%/Code/User/workspaceStorage/*/GitHub.copilot-chat/`
- [x] 4.4 實作 VS Code 變體支援（Insiders, VSCodium, Cursor）
- [x] 4.5 撰寫 session discovery 的單元測試

## 5. Token Estimation

- [x] 5.1 建立 `internal/transcript/vscode_copilot_token.go`，定義 TokenEstimator struct
- [x] 5.2 實作優先級 1：從 debug-logs 提取實際 token 數（llm_request 事件）
- [x] 5.3 實作優先級 1：從 transcripts 提取 session.shutdown 的 modelMetrics
- [x] 5.4 實作優先級 2：character-to-token ratio 估算（預設 0.25 tokens/char）
- [x] 5.5 實作優先級 3：ratio-based input:output 估算（根據 tool call 數量選擇 130:1, 50:1, 10:1）
- [x] 5.6 實作 model-specific token ratio 支援
- [x] 5.7 實作 cost 計算（使用 modelPricing 資料）
- [x] 5.8 撰寫 token 估算器的單元測試

## 6. VS Code Extension Bridge

- [x] 6.1 建立 `vscode-extension/` TypeScript 專案（package.json, tsconfig.json）
- [x] 6.2 實作 FileWatcher：監聽 workspaceStorage 檔案變更
- [x] 6.3 實作 TT Bridge：呼叫 `tt record prompt --tool vscode-copilot`
- [x] 6.4 實作 TT Bridge：呼叫 `tt record response --tool vscode-copilot`
- [x] 6.5 實作 tt CLI 未找到時的警告處理
- [x] 6.6 實作 extension activation（onStartupFinished + Copilot Chat 偵測）
- [x] 6.7 建立 .vsix build 流程

## 7. Setup Command Integration

- [x] 7.1 新增 `--vscode-copilot` flag 到 `tt setup` 命令
- [x] 7.2 實作 `internal/setup/vscode_copilot.go`：安裝 VS Code Extension
- [x] 7.3 實作 VS Code 未安裝時的警告處理
- [x] 7.4 更新自動偵測邏輯：偵測 VS Code + Copilot Chat 擴充套件
- [x] 7.5 撰寫 setup 命令的單元測試

## 8. Record Command Integration

- [x] 8.1 更新 `cmd/tt/record.go`：支援 `--tool vscode-copilot`
- [x] 8.2 更新 `internal/report/report.go`：顯示 vscode-copilot tool 類型
- [x] 8.3 撰寫 record 命令的整合測試

## 9. End-to-End Testing

- [x] 9.1 建立 VS Code Copilot session 測試資料（transcripts + chatSessions + debug-logs）
- [x] 9.2 撰寫端到端測試：從 session 檔案到 SQLite 記錄
- [x] 9.3 驗證 report 命令正確顯示 vscode-copilot 數據
