# Brainstorm: 新增 opencode 支援（token 追蹤 + session 判斷）

**日期**: 2026-06-23
**目標**: 讓 tt 支援 opencode 的 token 用量記錄與 session 判斷，比照現有 Claude Code / Copilot CLI / Antigravity / Codex 的支援方式。

---

## 決定

**Approach 1 — 薄 JS/TS Plugin + tt CLI bridge**。

opencode 沒有像 Claude Code 的外部 hook（stdin JSON 呼叫外部程式），而是 JS/TS Plugin 系統（事件監聽，在 opencode 行程內執行）。寫一個薄 plugin 監聯 `message.updated` event，透過 `exec` 呼叫 `tt record prompt/response --tool opencode`。

opencode 的 token 資料已內含於 `message.updated` event，不需 transcript parser。此模式跟現有工具一致，最大化重用 tt 基礎建設（pricing / DB / report / reconcile）。

---

## 架構

```
┌─────────────────────────────────────────────────┐
│  opencode 行程                                    │
│                                                  │
│  ┌────────────────┐    message.updated event     │
│  │  tt plugin      │◄────────────────────────     │
│  │  (JS/TS)        │                             │
│  │                  │                            │
│  │  role=user ──────┤── exec ──► tt record prompt│
│  │  role=asst ─────┤── exec ──► tt record resp   │
│  └────────────────┘                             │
└─────────────────────────────────────────────────┘
                          │
                          ▼
              ┌──────────────────────┐
              │  tt (Go binary)       │
              │                      │
              │  record prompt ──────► INSERT turn (prompt_at, session_id, tool="opencode")
              │  record response ────► UPDATE turn (response_at, tokens, cost, model)
              │                      │   INSERT turn_model_usages (subagent tokens)
              │                      │
              │  report ─────────────► Query + aggregate
              └──────────────────────┘
```

關鍵差異：其他工具 token 來自 transcript JSONL（reconcile 階段 parse），opencode event 已含完整 token 資料。`tt record response` 直接吃 `--tokens` JSON，不需回退 transcript parser。

---

## Plugin 事件處理邏輯

### User Prompt → `tt record prompt`

觸發條件：
- `role = "user"`
- `time.created` 存在

Flag mapping：
| Flag | 值 |
|------|----|
| `--tool` | `"opencode"` (常數) |
| `--session` | `info.sessionID` |
| `--project` | plugin context `directory` |
| `--model` | `info.model?.modelID ?? ""` (user 多半沒 model，optional) |

### Assistant Response → `tt record response`

觸發條件（全部符合）：
- `role = "assistant"`
- `modelID` 存在
- `tokens` 存在
- `time.completed` OR `finish` 存在（串流結束，非中間態）

Flag mapping：
| Flag | 值 |
|------|----|
| `--tool` | `"opencode"` (常數) |
| `--session` | `info.sessionID` |
| `--model` | `info.model.modelID` |
| `--tokens` | JSON string `{input_tokens, output_tokens, reasoning_tokens?, cache_read_tokens?, cache_write_tokens?}` |
| `--subagent-tokens` | JSON array of `{model, agent, input_tokens, output_tokens, ...}` |

### Subagent 處理

opencode subagent 訊息也在同一 event stream，差異在 `info.agent` 欄位：

| agent 值 | 含義 | tt 處理 |
|----------|------|---------|
| `undefined` 或 `""` | 主 agent | `tt record response` — 更新 turn 主 token |
| `"build"` / `"explore"` 等 | subagent | 暫存 token，等主 agent response 完成後隨 `--subagent-tokens` 一起送 |

Plugin 維護 in-memory map：
```
pendingSubTokens[sessionId][messageId] = {
  model, provider, tokens: {input, output, ...}
}
```

主 agent response 完成時：
1. exec `tt record response` + 主 agent tokens + `--subagent-tokens <json>`
2. 清空該 session 的 pendingSubTokens

### 串流防護

比照 opencode-token-tracker 邏輯：
```
if modelID 存在 && !time.completed && !finish:
  return  // 串流中間態，等下一幀

dedupKey = `${messageId}-${input}-${output}-${cacheRead}`
if seen.has(dedupKey):
  return  // 已處理過

seen.add(dedupKey)
```

---

## tt 端 Required Changes

| 檔案 | 改動 |
|------|------|
| `cmd/tt/record.go` | `resolvePromptInput` 處理 `tool == "opencode"`（不需 transcript path） |
| `cmd/tt/record.go` | `resolveResponseInput` 當 `--tool opencode` 時跳過 transcript parser fallback（tokens 直接來自 event flag） |
| `cmd/tt/record.go` | 新增 `--subagent-tokens` flag |
| `internal/recorder/response.go` | 解析 `--subagent-tokens` JSON，INSERT `turn_model_usages` |
| `internal/transcript/` | 註冊 `"opencode"` LogProvider（空實作，token 不來自 transcript） |
| `cmd/tt/report.go` | `normalizeAgentName` 加 `"opencode"` → `"OpenCode"` |
| `cmd/tt/setup.go` / `internal/setup/` | 新增 `SetupOpencode` + `--opencode` flag |
| `internal/pricing/` | 確認定價表涵蓋 opencode 常用模型（應已涵蓋） |

---

## 安裝流程

```bash
# 1. tt 端安裝 plugin
tt setup --opencode

# 2. 重啟 opencode
```

`tt setup --opencode` 做什麼：
- 產生 `~/.config/opencode/plugins/tt-bridge.ts`（本地 plugin，免 npm）
- 冪等：已存在不重複產生

vs npm 發布：先走本地檔案最簡單，未來可包 npm。

---

## 專案目錄判斷

Plugin context 提供 `directory`（opencode 啟動時的工作目錄）。

驗證：`workitem.ResolveProject(dir)` 跑 `git -C dir rev-parse --show-toplevel`。Plugin 給的 `directory` 是專案目錄（如 `/Users/.../time-tracker`），`ResolveProject` 回傳 git root。跟其他工具的路徑一致。

`isInvalidProject` 只擋 `.gemini` / `.claude` / `.copilot`，不擋 opencode 的 config dir。不需額外處理。

---

## 參考：opencode-token-tracker 分析

社群專案 [opencode-token-tracker](https://github.com/tongsh6/opencode-token-tracker) 已證實 opencode plugin event 可取得 token 資料。但**無法直接重用**：

| 差異 | token-tracker | tt 需要的 |
|------|--------------|-----------|
| 追蹤範圍 | 只記 assistant 回覆 | 需 prompt + response 成對 |
| 寫入目標 | 自有 JSONL | tt 的 SQLite（透過 `tt record` CLI） |
| 定價 | 內建 JS 定價表 | tt 的 Go pricing package |
| 專案辨識 | 無 | 需 git branch、cwd、work-item |
| 資料模型 | per-message | per-turn（prompt_at + response_at + tokens） |

可參考：`MessageInfo` interface、串流保護邏輯、去重策略、定價表。

---

## 開放問題

1. **User message 是否帶 model？** — 需實測。若不帶，prompt 先不傳 model，等 response 時由 reconcile 補。
2. **Subagent 相對於主 response 的事件順序？** — 暫定主 response 完成時一起 flush。若 subagent 在主 response 之後才完成，可能漏記——需實測確認。
3. **Plugin 部署：本地檔案 vs npm 發布？** — 建議先本地檔案，未來可包 npm。

---

## 下一步路由

此計畫涵蓋單一決策（opencode 支援）。建議使用 `/spex-propose` 產生 OpenSpec change proposal，然後 `/spex-apply` 開始實作。