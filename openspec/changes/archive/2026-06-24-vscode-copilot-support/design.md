## Context

tt 是一個 Go CLI 工具，透過 Claude Code / Copilot CLI / OpenCode hook 自動記錄 AI 工作時間與 token 費用。現有的 Copilot 整合只支援 Copilot CLI（透過 `~/.copilot/hooks/tt.json`），不支援 VS Code 內建的 GitHub Copilot Chat。

VS Code Copilot Chat 的 session 檔案存放在 `workspaceStorage/<ext-id>/GitHub.copilot-chat/` 下，包含三種格式：
- `transcripts/*.jsonl`：對話事件（session.start、user.message、assistant.message 等）
- `chatSessions/*.json`：session metadata（modelId、details、thinking.tokens）
- `debug-logs/{sessionId}/main.jsonl`：LLM 請求事件（inputTokens、outputTokens、cachedTokens）

參考專案 `ai-engineering-fluency`（MIT license）已驗證這些格式的解析邏輯和 token 估算策略。

## Goals / Non-Goals

**Goals:**
- 讓使用者透過 `tt setup --vscode-copilot` 一键安裝 VS Code Extension，自動追蹤 Copilot Chat 活動
- 從三種 session 檔案格式提取 timestamps、model 資訊、tool calls、token 使用量
- 提供多層級 token 估算：實際 token 數 > character-to-token ratio > ratio-based input:output
- 與現有 tt report 系統整合，顯示 `vscode-copilot` 作為 tool 類型

**Non-Goals:**
- 追蹤 Copilot inline code suggestions（VS Code API 不支援觀察其他 extension 的 completions）
- 追蹤 Copilot 的 quota/entitlement（需要 polling GitHub API，不在範圍內）
- 建立完整的 VS Code Extension UI（只做輕量 bridge）
- 支援 Copilot Extensions / MCP 平台（已被 MCP 取代，不適合被動追蹤）

## Decisions

### Decision 1: VS Code Extension 只做輕量 bridge

**選擇**：Extension 只負責監聽 workspaceStorage 檔案變更，呼叫 `tt record prompt/response --tool vscode-copilot` CLI

**替代方案**：
- A: Extension 內建完整解析邏輯（TypeScript）
- B: Extension 只做 bridge，解析在 Go 端（選中）

**理由**：
- 避免維護兩套解析邏輯（TypeScript + Go）
- 與現有 Copilot CLI 整合模式一致（hook 呼叫 tt CLI）
- Go 端可以重用現有的 transcript 解析框架

### Decision 2: Token 估算策略三層回退

**選擇**：優先使用 debug-log 的實際 token 數，回退到 character-to-token ratio，再回退到 ratio-based input:output

**理由**：
- Debug logging 可能未啟用，需要 fallback
- Character-to-token ratio（0.25 tokens/char）是經過驗證的估算方法
- Ratio-based input:output 根據 tool call 數量調整，比固定 ratio 更準確

### Decision 3: Session 檔案解析在 Go 端實作

**選擇**：新增 `internal/transcript/vscode_copilot.go` 和 `internal/transcript/vscode_copilot_token.go`

**理由**：
- 與現有 Copilot transcript 解析器（`copilot_transcript.go`）架構一致
- 可重用現有的 `LogProvider` 介面
- 避免引入 TypeScript build 流程

## Risks / Trade-offs

- **[Risk] Session 檔案格式變更** → Copilot 版本更新可能改變格式。Mitigation: 使用 struct tag + `json:"-"` 忽略未知欄位，保持向後相容
- **[Risk] Debug logging 未啟用** → 無法取得實際 token 數。Mitigation: 三層回退策略，確保總有估算值
- **[Risk] workspaceStorage 路徑差異** → 不同 OS 和 VS Code 變體的路徑不同。Mitigation: 參考 ai-engineering-fluency 的路徑 discovery 邏輯
- **[Trade-off] Extension 安裝需要 .vsix** → 需要預建 .vsix 或 go:embed 嵌入。Mitigation: 先支援手動安裝，之後再加自動安裝
