# VS Code GitHub Copilot 支援

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

tt 目前支援 Claude Code 和 Copilot CLI 的時間追蹤。使用者希望擴展到 VS Code 內建的 GitHub Copilot，讓使用 VS Code Copilot Chat 的使用者也能自動記錄 AI 工作時間。

### 研究發現

1. VS Code 公開 API 無法直接觀察 Copilot 的 inline suggestions 或 chat interactions
2. Copilot Extensions 平台已被 MCP 取代，不適合被動追蹤
3. VS Code Copilot 的 session 檔案存放在 `workspaceStorage/<ext-id>/GitHub.copilot-chat/` 下
4. 已有成熟開源參考：[ai-engineering-fluency](https://github.com/rajbos/ai-engineering-fluency)（95 stars，MIT license）

### 參考專案：ai-engineering-fluency

前身是 `copilot-token-tracker`，已解決大部分技術問題：
- 支援 VS Code、JetBrains、Visual Studio、CLI 等多平台
- 有完整的 session 檔案解析邏輯（`sessionParser.ts`、`tokenEstimation.ts`）
- 有 token 估算策略（character-to-token ratio + ratio-based input:output 估算）
- 有 debug log 解析（`extractAllTokensFromDebugLog`）可取得實際 token 數
- 有 model pricing 資料（`modelPricing.json`）

## Decision

參考 `ai-engineering-fluency` 的實作邏輯，用 Go 實作 VS Code Copilot session 檔案解析器，與 tt 架構完全整合。

## Rationale

1. 最符合 tt 的「被動追蹤」設計哲學 — 使用者不需要改變行為
2. 參考已有成熟實作，不需要重新發明輪子
3. 用 Go 實作（與 tt 同語言），避免 TypeScript build 流程的複雜度
4. `ai-engineering-fluency` 的 token 估算策略經過大量實際使用驗證

## Approach

### 架構

```
┌─────────────────────────────────────────────────┐
│  VS Code Extension (TypeScript, 輕量)            │
│                                                  │
│  ┌──────────────────┐  ┌──────────────────────┐ │
│  │ FileWatcher       │  │ Activation Events    │ │
│  │ 監聽 workspace-   │  │ onStartupFinished    │ │
│  │ Storage 檔案變更  │  │                      │ │
│  └────────┬─────────┘  └──────────┬───────────┘ │
│           │                       │              │
│           ▼                       ▼              │
│  ┌────────────────────────────────────────────┐  │
│  │ TT Bridge                                  │  │
│  │ - 呼叫 tt record prompt/response CLI       │  │
│  │ - 傳遞 --tool vscode-copilot               │  │
│  └────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
        ↓ (呼叫 CLI)
┌─────────────────────────────────────────────────┐
│  tt CLI (Go)                                     │
│                                                  │
│  ┌────────────────────────────────────────────┐  │
│  │ Session Parser (internal/transcript/)       │  │
│  │ - 解析 transcripts/*.jsonl                  │  │
│  │ - 解析 chatSessions/*.json                  │  │
│  │ - 解析 debug-logs/*/main.jsonl              │  │
│  │ - 提取 timestamps、model、tool calls        │  │
│  └────────────────────┬───────────────────────┘  │
│                       │                          │
│                       ▼                          │
│  ┌────────────────────────────────────────────┐  │
│  │ Token Estimator                            │  │
│  │ - 優先使用 debug-logs 的實際 token 數       │  │
│  │ - 回退到 character-to-token ratio 估算      │  │
│  │ - 根據 tool call 數量調整 input:output ratio│  │
│  └────────────────────┬───────────────────────┘  │
│                       │                          │
│                       ▼                          │
│  ┌────────────────────────────────────────────┐  │
│  │ SQLite DB                                  │  │
│  └────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
```

### 三個資料來源（從 ai-engineering-fluency 學到）

**Source 1: `transcripts/*.jsonl`**（workspaceStorage）
- 事件類型：`session.start`、`user.message`、`assistant.message`、`assistant.turn_start/end`、`tool.execution_start/complete`
- 包含：timestamps、content、tool requests、reasoning text
- **不包含：token 使用量**
- 用途：session 時間、對話次數、tool call 數量

**Source 2: `chatSessions/*.json`**（workspaceStorage）
- 包含：`modelId`（如 `copilot/gpt-5-codex`）、`details`（如 `GPT-5-Codex (Preview) • 1x`）
- 包含：`thinking.tokens`（只有 thinking tokens）
- **不包含：完整的 input/output tokens**
- 用途：model 資訊、session metadata

**Source 3: `debug-logs/{sessionId}/main.jsonl`**（workspaceStorage）⭐ 關鍵資料來源
- 包含：`llm_request` 事件，每個事件有 `attrs.inputTokens`、`attrs.outputTokens`、`attrs.cachedTokens`
- 包含：`session.shutdown` 事件，有 `data.modelMetrics` 和 `data.totalNanoAiu`
- **有完整的實際 token 使用量**（但需要 Copilot 啟用 debug logging）
- 用途：精確的 token 數、model breakdown、費用計算

### Token 估算策略（來自 ai-engineering-fluency）

**優先級 1：使用實際 token 數**
- 從 `debug-logs/*/main.jsonl` 的 `llm_request` 事件提取
- 從 `transcripts/*.jsonl` 的 `session.shutdown` 事件提取（`data.modelMetrics`）

**優先級 2：Character-to-token ratio 估算**
- 預設 ratio：0.25 tokens/char（4 字元 ≈ 1 token）
- 可用 `tokenEstimators.json` 中的 model-specific ratios
- 從 transcript 內容估算 output tokens

**優先級 3：Ratio-based input:output 估算**
- 根據 tool call 數量選擇 ratio：
  - ≥20 tool calls：130:1（heavy agent）
  - 5-19 tool calls：50:1（medium）
  - <5 tool calls：10:1（simple chat）
- 從估算的 output tokens 推算 input tokens

### 安裝流程

`tt setup --vscode-copilot` → 嵌入 .vsix → `code --install-extension`

VS Code Extension 只做輕量工作：
- 監聽 workspaceStorage 檔案變更
- 呼叫 `tt record prompt/response --tool vscode-copilot` CLI
- 傳遞 session ID 和檔案路徑

### 新增的 tt 命令

- `tt setup --vscode-copilot` — 安裝 VS Code Extension
- `tt record prompt --tool vscode-copilot` — 記錄 prompt（由 Extension 呼叫）
- `tt record response --tool vscode-copilot` — 記錄 response（由 Extension 呼叫）

### 新增的 Go 模組

- `internal/transcript/vscode_copilot.go` — VS Code Copilot session 檔案解析器
  - 解析 transcripts JSONL（event-based 格式）
  - 解析 chatSessions JSON（delta-based 格式）
  - 解析 debug-logs JSONL（llm_request 事件）
- `internal/transcript/vscode_copilot_token.go` — Token 估算邏輯
  - character-to-token ratio 估算
  - ratio-based input:output 估算
  - debug-log 實際 token 提取
- `internal/setup/vscode_copilot.go` — VS Code Extension 安裝邏輯

### 報表整合

- `tt report` 會顯示 `vscode-copilot` 作為 tool 類型
- 顯示：session 時間、model 使用、token 使用量（實際或估算）、費用估算

## Insights to Capture

- `design.md`: VS Code Copilot session 檔案格式（transcripts + chatSessions + debug-logs）
- `specs/<capability>/spec.md`: VS Code Copilot 追蹤需求
- `proposal.md`: VS Code Extension scope
- `tasks.md`:
  - 建立 VS Code Extension 專案（TypeScript，輕量 bridge）
  - 實作 transcripts JSONL 解析器（event-based 格式）
  - 實作 chatSessions JSON 解析器（delta-based 格式）
  - 實作 debug-logs JSONL 解析器（llm_request 事件）
  - 實作 character-to-token ratio 估算器
  - 實作 ratio-based input:output 估算器
  - 實作 TT Bridge（呼叫 tt record CLI）
  - 實作 FileWatcher（監聽 workspaceStorage 檔案變更）
  - 新增 `internal/transcript/vscode_copilot.go`
  - 新增 `internal/transcript/vscode_copilot_token.go`
  - 新增 `internal/setup/vscode_copilot.go`
  - 更新 `tt setup` 命令支援 `--vscode-copilot`
  - 更新報告顯示 `vscode-copilot` tool 類型
  - 撰寫測試

## Open Questions

- Debug logging 是否預設啟用？（影響實際 token 數的可用性）
- Session 檔案格式可能隨 VS Code/Copilot 版本變更（需要版本相容性處理）
- VS Code Extension 的 vsix 嵌入方式需要確認（go:embed + 預建 vsix）
- workspaceStorage 路徑在不同 OS 上的差異（macOS vs Linux vs Windows）
