# token-calculation-research

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

使用者希望能研究 GitHub 專案 `steipete/CodexBar` 的多種 AI 工具（Antigravity, GitHub Copilot, Claude Code, Codex）的 Token 計算機制，以評估是否能將相似功能實作於 `tt` 時間追蹤器中。
在調研過程中，我們發現：
1. **API 的限制**：官方/內部 API（如 Anthropic OAuth API、GitHub Copilot API、Google Quota API）大多只提供累計額度（quota/spend limit），**不提供 turn 等級的精細對話 Token 明細**。
2. **開源工具的實作方式**：開源工具如 `ccusage` (TypeScript) 與 `tokscale` 近期已改為**直接解析本地日誌**來獲取最精準的 Token 使用數據。
3. **本地日誌分析**：
   - **GitHub Copilot CLI**：會寫入本地 `~/.copilot/session-state/<session_id>/events.jsonl`。其中的 `session.shutdown` 事件包含完整的 `modelMetrics`，有精確的 `inputTokens`、`outputTokens`、`cacheReadTokens`、`cacheWriteTokens` 及 `reasoningTokens`。
   - **Antigravity**：會寫入本地 `~/.gemini/antigravity/brain/<conversation_id>/.system_generated/logs/transcript.jsonl`，包含 turn 等級的 token 與模型資料。
   - **Claude Code**：會寫入本地 `~/.claude/projects/**/*.jsonl`，目前 `tt` 已經支援此檔案的解析。

## Decision

未來 `tt` 將擴充對 `GitHub Copilot CLI` 與 `Antigravity` 的 Token 追蹤支援，全面採用 **「本地日誌靜默掃描與解析（Local Log Scraper）」** 方案。
當 `Stop` 鉤子觸發時，`tt` 會利用傳入的 `sessionId` 定位本地的對應事件日誌（`.jsonl`），並提取精準的 Model 名稱與 Token 數據存入資料庫。

## Rationale

1. **零依賴與效能**：不需要用戶提供任何 API 金鑰，也無任何網路請求，完全離線運作，符合 `tt` 的核心原則。
2. **Turn 精度**：這是唯一能將 Token 消耗與特定工作區間（Turn/Work Item）精確對齊的方法。
3. **現有架構對齊**：`tt` 已有讀取與解析 JSONL 的底層實作，擴充其他工具的 Log Parser 成本極低。

## Approach

1. **Copilot CLI 解析器**：
   - 當 `tool == "copilot-cli"` 時，`tt` 讀取 `~/.copilot/session-state/<sessionId>/events.jsonl`。
   - 篩選 `"type":"session.shutdown"` 事件，反序列化 `modelMetrics` 物件，提取對應模型的 `inputTokens`、`outputTokens`、`cacheReadTokens` 及 `cacheWriteTokens`。
2. **Antigravity 解析器**：
   - 當 `tool == "antigravity"` 時，`tt` 讀取 `~/.gemini/antigravity/brain/<sessionId>/.system_generated/logs/transcript.jsonl`，統計主 Agent 的 model usage。

## Design Notes

### Copilot CLI `session.shutdown` 事件結構範例：
```json
{
  "type": "session.shutdown",
  "data": {
    "modelMetrics": {
      "gpt-5.4": {
        "usage": {
          "inputTokens": 9848523,
          "outputTokens": 31389,
          "cacheReadTokens": 9721856,
          "cacheWriteTokens": 0,
          "reasoningTokens": 12340
        }
      }
    }
  }
}
```

### 系統架構整合：
1. `cmd/tt/record.go`：在 `record response` 指令執行時，新增根據 `tool` 分流讀取本地 log 的邏輯。
2. `internal/transcript/`：新增 `copilot_transcript.go` 與 `antigravity_transcript.go` 來實作對應格式的 JSONL 解析。

## Insights to Capture

- `specs/model-cost-tracking/spec.md`: 新增支援 Copilot 與 Antigravity 本地 log 的解析規範。
- `internal/pricing/pricing.go`: 更新定價表，加入 `gpt-5.4`、`gpt-5-mini` 與 `claude-sonnet-4.6` 等常見 Copilot/Antigravity 預設模型的定價條目。

## Open Questions

（無，設計已收斂）
