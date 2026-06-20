## Context

目前 `tt` (AI Tool Time Tracker) 會在 Stop hook 呼叫 `tt record response` 時，解析本地的日誌（主要是 Claude Code 產生的 `.jsonl` 檔案）來取得精確的 token 消耗與模型名稱。
為了擴展對其他主流 AI 工具的支援，我們需要新增對 `GitHub Copilot CLI` 與 `Antigravity` 的 Token 追蹤支援。這兩種工具同樣會在本地寫入 JSONL 格式的事件日誌：
- **GitHub Copilot CLI**：會寫入本地 `~/.copilot/session-state/<sessionId>/events.jsonl`，在 `"type":"session.shutdown"` 的事件中包含完整的 `modelMetrics`，擁有精確的模型 input/output/cache token 消耗。
- **Antigravity**：會寫入本地 `~/.gemini/antigravity/brain/<sessionId>/.system_generated/logs/transcript.jsonl`，記錄了主 Agent 與 subagent 的對話日誌，包含 token 數量與模型資訊。

## Goals / Non-Goals

**Goals:**
- 新增 `internal/transcript/copilot_transcript.go` 與 `internal/transcript/antigravity_transcript.go`，實作對應日誌格式的 JSONL 解析。
- 在 `cmd/tt/record.go` 中，當執行 `record response` 且為 `copilot-cli` 或 `antigravity` 時，自動讀取並解析上述對應的本地日誌檔案。
- 補足/更新定價表 `internal/pricing/pricing.go` 以涵蓋 `gpt-5.4` 與 `gpt-5-mini`。

**Non-Goals:**
- 支援其他尚未在 Context 中定義的 AI 開發工具（如 Codex）。
- 實作遠端 API 來查詢即時額度（完全離線/本地日誌解析）。
- 修改 `tt` 現有的 SQLite 資料庫 schema（現有欄位已能容納 input, output, cache 等 tokens）。

## Decisions

1. **日誌檔案定位與路徑解析**
   - **Copilot CLI**：日誌檔案路徑為 `~/.copilot/session-state/<sessionId>/events.jsonl`。其中 `<sessionId>` 對應執行 hook 時傳入的 `sessionId`。
   - **Antigravity**：日誌檔案路徑為 `~/.gemini/antigravity/brain/<sessionId>/.system_generated/logs/transcript.jsonl`。其中 `<sessionId>` 對應 `sessionId`（在 Antigravity 中即為 conversationId）。
   - 使用 Go 的 `os.UserHomeDir()` 來解析波浪號 `~` 所代表的使用者家目錄。

2. **Copilot CLI 解析邏輯**
   - 讀取日誌，逐行解析 JSONL，過濾出 `"type": "session.shutdown"` 的事件。
   - 解析該事件下的 `data.modelMetrics`，其下各個 model entry（例如 `gpt-5.4`）包含 `usage` 對象：
     - `usage.inputTokens` -> Input
     - `usage.outputTokens` -> Output
     - `usage.cacheReadTokens` -> CacheRead
     - `usage.cacheWriteTokens` -> CacheCreation/CacheCreation5m
     - `usage.reasoningTokens` 可選累加到 Output 或獨立處理（依定價表通常將 reasoning 包含於 output 額度）。

3. **Antigravity 解析邏輯**
   - 讀取日誌，逐行解析 JSONL。
   - 針對每一行中的 `tool_calls` 或 model output 欄位進行解析與累加統計。

## Risks / Trade-offs

- **[Risk]** 使用者環境中的 `~` (Home Directory) 路徑解析失敗。
  - **Mitigation** 使用 Go `os.UserHomeDir()` 獲取絕對路徑，若失敗則退回當前目錄或報錯，但 hook 本身仍靜默處理（exit 0），不阻擋使用者。
- **[Risk]** 本地日誌檔案不存在或格式不符合預期（如 session 尚未 shutdown 導致 shutdown 事件未寫入）。
  - **Mitigation** 若檔案不存在或未找到 `session.shutdown`，則回傳空/零 token 或使用預設值，不中斷執行，以確保 hook 靜默失敗原則。
