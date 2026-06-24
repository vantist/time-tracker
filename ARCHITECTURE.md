# Architecture

## Overview

`tt` 是一個 Go CLI 工具，自動追蹤 AI 開發工作流的時間與費用。透過 Claude Code、Copilot CLI、OpenCode、VS Code Copilot 等 AI 工具的 hook 系統，靜默記錄每次 prompt/response 事件至本地 SQLite，並提供依專案、時間範圍、工作項目的報表查詢。無外部 runtime 依賴，單一二進位。

## 模組結構

```
cmd/tt/          CLI 進入點（Cobra）
  main.go        根指令設定
  record.go      record prompt / record response（hook 接收）
  work.go        tt work（工作項目標記）
  report_cmd.go  tt report（查詢報表）
  serve_cmd.go   tt serve（Web dashboard HTTP server）
  config_cmd.go  tt config set（設定管理）
  setup_cmd.go   tt setup（hook 安裝）
  version.go     tt version（版本資訊）

internal/
  db/            SQLite schema、連線、session upsert
  recorder/      RecordPrompt / RecordResponse（寫入 DB）
  report/        Query + FormatText/JSON/HTML（讀取 DB）；HandleDashboard/HandleAPIReport（HTTP handler）
  aggregator/    AgentTime / UserActiveTime（時間計算）
  pricing/       per-model 定價表 + 費用估算
  config/        ~/.tt/config.json 讀寫
  workitem/      ~/.tt/work-item 讀寫
  setup/         settings.json hook 合併（Claude Code, Copilot CLI, OpenCode, VS Code Copilot）
  transcript/    ExtractWindow()：從 JSONL transcript 擷取 token 用量（含 subagent）；支援 5 種 provider
  reconcile/     MaybeReconcile()：補齊 Stop hook 未觸發的 dangling turns
  process/       IsAlive()：跨平台 PID 存活檢查（防 PID 重用）
```

## 資料流

### 記錄流程

```
Claude Code UserPromptSubmit hook
  → stdin JSON {session_id, cwd, model, conversation_id, transcript_path, line_count}
  → tt record prompt
  → recorder.RecordPrompt()
      ├─ git branch 自動偵測
      ├─ workitem.Get() → ~/.tt/work-item
      ├─ process.StartTime(pid) → process_pid, process_start
      ├─ db.UpsertSession()  → sessions 表
      └─ INSERT turns (prompt_at, transcript_path, prompt_line_offset)

Claude Code Stop hook
  → stdin JSON {session_id} + token 欄位
  → tt record response
  → recorder.RecordResponse()
      ├─ 解析 token JSON (若未提供則從 log 擷取)
      ├─ pricing.Calculate(model, tokens)
      ├─ UPDATE turns (response_at, tokens, cost)
      └─ INSERT turn_model_usages (is_subagent = 0)

Reconcile（補齊）
  → reconcile.MaybeReconcile()（tt serve 啟動時 / API 請求時）
      ├─ 找出 response_at 或 input_tokens 為 NULL，或 subagent_tokens_settled = 0 的 turns
      ├─ process.IsAlive() 確認 session 是否結束（或 dangling 時間過長）
      ├─ transcript.ExtractWindow() 從 JSONL 讀取 token（遞迴累加 subagent 用量）
      ├─ 交易（Transaction）：DELETE 舊 usages ➔ INSERT 所有 main & subagent usages
      └─ UPDATE turns（回填 response_at、tokens、cost，並置 subagent_tokens_settled = 1）
```

### 報表流程

```
tt report [--since 7d] [--project X] [--by-work-item]
  → report.Query()
      ├─ JOIN turns + sessions
      ├─ aggregator.AgentTime()     = Σ(response_at - prompt_at)
      └─ aggregator.UserActiveTime() = Σ(inter-prompt gaps < idle_threshold)
  → FormatText() 或 FormatJSON()
```

## 資料庫 Schema（SQLite，`~/.tt/data.db`）

```sql
sessions (id PK, project, tool, model, branch, work_item, started_at, ended_at,
          process_pid, process_start, conversation_id)
turns    (id PK, session_id FK, prompt_at, response_at,
          input_tokens, output_tokens, cache_read_tokens,
          cache_creation_tokens, cache_creation_5m_tokens,
          cache_creation_1h_tokens, estimated_cost_usd,
          transcript_path, prompt_line_offset, model,
          subagent_tokens_settled)
turn_model_usages (id PK, turn_id FK, model, is_subagent,
                   input_tokens, output_tokens, cache_read_tokens,
                   cache_creation_tokens, cache_creation_5m_tokens,
                   cache_creation_1h_tokens, estimated_cost_usd)
```

- 一個 session = 一次 Claude Code/Copilot CLI 啟動
- 一個 turn = 一次 prompt-response 循環
- `process_pid` / `process_start`：用於 reconcile 判斷 session 是否仍存活
- `transcript_path` / `prompt_line_offset`：reconcile 從 JSONL 擷取 token 的位置指針
- `TT_DB_PATH` env var 可覆寫 DB 路徑（測試用）

## 工作項目優先順序

`work_item` > `branch` > `"untagged"`（報表分組依此順序 fallback）

## Hook 整合

| Hook 事件 | 指令 | 用途 |
|-----------|------|------|
| `UserPromptSubmit` | `tt record prompt` | 記錄 prompt |
| `Stop` | `tt record response` | 記錄 token 與費用 |
| Copilot `userPromptSubmitted` | `tt record prompt --tool copilot-cli` | 同上 |
| Copilot `agentStop` | `tt record response --tool copilot-cli` | token 為 NULL（不支援） |
| Antigravity `PreInvocation` | `tt record prompt --tool antigravity` | 記錄 Antigravity prompt |
| Antigravity `Stop` | `tt record response --tool antigravity` | 記錄 Antigravity token 與費用 |
| Codex `UserPromptSubmit` | `tt record prompt --tool codex` | 記錄 Codex prompt |
| Codex `Stop` | `tt record response --tool codex` | 記錄 Codex token 與費用 |
| OpenCode `onEvent` | `tt record prompt/response --tool opencode` | 記錄 OpenCode prompt/response（TypeScript bridge） |
| VS Code Copilot `onDidCreate` | `tt record prompt/response --tool vscode-copilot` | 記錄 VS Code Copilot Chat 活動（TypeScript bridge） |

Hook 失敗靜默處理（exit 0，錯誤寫 stderr），不阻擋 AI 工具正常運作。

## 關鍵設計決策

- **Go 單一二進位**：冷啟動 <10ms，無 runtime 依賴
- **modernc.org/sqlite**：純 Go SQLite，無 cgo，跨平台 build
- **定價表硬編碼**：不開放設定，跟隨 binary 版本更新
- **idle threshold**：預設 15 分鐘，`tt config set idle-threshold <minutes>` 可調整
- **stdin 優先於 flag**：hook 傳入 stdin JSON，flag 保留供手動/測試用
- **Reconcile 機制**：Stop hook 可能未觸發（process kill、crash），reconcile 透過 transcript JSONL 補齊 dangling turns，`tt serve` 啟動及 API 請求時自動執行
- **Subagent token 歸因**：`transcript.ExtractWindow()` 解析 Agent tool-use block，遞迴加總 subagent JSONL token
- **間隔時間合併算法 (Interval Merging)**：在統計開發者工時（UserActiveTime）時，將重疊的 active intervals 在時間軸上進行排序並合併，避免多個並行或交錯的 session 重複計算工時，大幅提升工時統計精準度。
- **OpenCode 整合**：透過 TypeScript bridge plugin（`tt-bridge.ts`）監聽 OpenCode 事件，直接呼叫 `tt record` CLI 記錄 prompt/response
- **VS Code Copilot 整合**：透過 TypeScript bridge extension 監聽 workspaceStorage 檔案變更，解析 transcripts、chatSessions、debug-logs 三種格式，使用多層級 token 估算策略（實際 token 數 > character-to-token ratio > ratio-based input:output）

---
Related: [Commands](docs/commands.md) | [Conventions](docs/conventions.md) | [Configuration](docs/configuration.md)
