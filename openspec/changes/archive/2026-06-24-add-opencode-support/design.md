## Context

tt 現有支援 Claude Code、Copilot CLI、Antigravity、Codex 四種 AI 工具，共通模式為：AI 工具透過外部 hook（stdin JSON）呼叫 `tt record prompt / response`，response 階段由 tt 自行 parse transcript JSONL 取得 token（reconcile 補算）。

opencode 採用不同整合模式——JS/TS Plugin 系統（事件監聯，在 opencode 行程內執行），且 `message.updated` event 已內含完整 token 資料（input/output/reasoning/cache_read/cache_write + subagent）。因此 opencode 不需 transcript parser，token 直接以 flag 傳入 `tt record response`。

現有相關模組：
- `cmd/tt/record.go` — `resolvePromptInput` / `resolveResponseInput`，依 `--tool` 分派 transcript 解析
- `internal/recorder/response.go` — 寫入 turn token 與 `turn_model_usages`
- `internal/transcript/` — LogProvider 註冊表（`claude-code` / `copilot-cli` / `antigravity`）
- `internal/setup/setup.go` — 多工具 setup 調度與自動偵測
- `cmd/tt/report_cmd.go` — `normalizeAgentName` 工具名正規化

參考實作：社群專案 opencode-token-tracker 已證實 plugin event 可取得 token，但其資料模型（per-message、自有 JSONL、JS 定價）與 tt 不相容，僅能借鏡事件結構與串流保護邏輯。

## Goals / Non-Goals

**Goals:**
- 讓 opencode 使用者透過 `tt setup --opencode` 一鍵安裝 plugin 後，自動記錄 prompt/response 成對 turn、主 agent token、subagent token
- 最大化重用 tt 現有 pricing / DB / report / reconcile 基礎建設
- 對現有四種工具的錄製路徑零影響

**Non-Goals:**
- 不發布 npm package（先本地檔案，未來再考慮）
- 不為 opencode 撰寫 transcript parser
- 不處理 subagent 晚於主 response 完成的事件順序邊界（待實測確認後另案）
- 不改動定價表結構（沿用現有 pricing package，僅確認涵蓋 opencode 常用模型）

## Decisions

### 決策 1：薄 JS/TS Plugin + tt CLI bridge（而非 npm 套件或 transcript parser）

Plugin 監聽 `message.updated` event，透過 `exec` 呼叫 `tt record prompt/response --tool opencode`。

**為何優於其他方案：**
- vs npm 套件：本地檔案免發布、免版本同步，`tt setup --opencode` 冪等產生即可；未來需求出現再包 npm
- vs transcript parser：opencode event 已含 token，重寫 parser 是多餘工作且需處理 opencode 日誌格式變動
- vs 內嵌 Go：opencode plugin 系統僅接受 JS/TS，無法用 Go

**重用最大化：** token 寫入 SQLite 後，report / pricing / reconcile / dashboard 全部沿用，不需改動。

### 決策 2：opencode response 跳過 transcript parser fallback

`resolveResponseInput` 當 `--tool opencode` 時，token 直接來自 `--tokens` flag，不回退 `ExtractWindow`。`resolvePromptInput` 當 `--tool opencode` 時不要求 `--transcript-path`。

**理由：** 其他工具 token 來自 transcript（reconcile 階段 parse）；opencode token 來自 event flag。混用會導致重複計算或空值覆蓋。`internal/transcript/opencode.go` 註冊空實作 LogProvider 純為 provider 註冊表完整性，不實際解析。

### 決策 3：subagent token 透過 `--subagent-tokens` JSON flag 傳入

Plugin 在記憶體暫存 subagent token（`pendingSubTokens[sessionId][messageId]`），主 agent response 完成時隨 `tt record response --subagent-tokens <json>` 一起 flush，`internal/recorder/response.go` 解析後以 `is_subagent = 1` INSERT 進 `turn_model_usages`。

**為何隨主 response 一起送（而非獨立命令）：** turn_model_usages 需關聯 turn_id，而 turn_id 只在主 response 寫入時確定；獨立命令需另傳 session + 時序資訊重查 turn，複雜度更高。暫存 + 一起 flush 與現有「subagent_tokens_settled = 0 待 reconcile 補算」語意相容——opencode 路徑直接提供最終值，reconcile 對 opencode turn 為 no-op。

**與既有 transcript 掃描路徑並存：** `subagent-token-capture` 既有需求（掃描 `tool_use { name: "Agent" }` + meta.json）不變，`--subagent-tokens` 為新增的 opencode 專屬輸入路徑。

### 決策 4：串流防護與去重

Plugin 對 assistant response 觸發條件：`role = "assistant"` AND `modelID` 存在 AND `tokens` 存在 AND（`time.completed` OR `finish` 存在）——避免串流中間態重複觸發。

去重：`dedupKey = ${messageId}-${input}-${output}-${cacheRead}`，`seen.has(dedupKey)` 則跳過。

**理由：** opencode 串流會多次發 `message.updated`，未防護會重複記錄。借鏡 opencode-token-tracker 已驗證邏輯。

### 決策 5：`tt setup --opencode` 冪等產生本地 plugin 檔

產生 `~/.config/opencode/plugins/tt-bridge.ts`。冪等：已存在則不重複產生（不覆蓋使用者可能客製化的內容；若需更新由使用者刪除後重跑）。

**vs Claude Code 的 idempotent-hook-setup：** Claude Code 是 merge `settings.json` 的 `_owner: "tt"` 條目；opencode 是獨立 plugin 檔，採「存在即跳過」較簡。沿用 setup-improvements 的多工具 flag 與自動偵測機制（`--opencode` flag、偵測 `~/.config/opencode/`）。

### 決策 6：`normalizeAgentName` 新增 `"opencode"` → `"OpenCode"`

沿用 agent-attribution 既有正規化規則結構，新增一條分支。report / dashboard 自動正確顯示工具名。

## Risks / Trade-offs

- **[Risk] subagent 晚於主 response 完成導致漏記** → 暫定主 response 完成時 flush pendingSubTokens；若實測發現 subagent 在主 response 之後才完成，需另設計延遲 flush 或獨立補送命令。列為開放問題，實作時優先實測。
- **[Risk] opencode plugin event 結構變更** → plugin 集中在單一檔案 `tt-bridge.ts`，欄位對應表集中；event schema 變動只需改 setup 範本重產生。
- **[Risk] user message 是否帶 model（開放問題 1）** → plugin 先以 `info.model?.modelID ?? ""` 處理，prompt 不傳 model；若實測 user 不帶 model，由 response 階段填入 turn.model（與現有 reconcile 補 model 機制一致）。
- **[Trade-off] 本地 plugin 檔 vs npm** → 簡單但不利版本追蹤；使用者更新 tt 後需手動刪舊 plugin 重跑 setup。可接受，未來再包 npm。
- **[Trade-off] subagent token 暫存於 plugin 記憶體** → opencode 行程重啟會丟失暫存；但 turn 已記錄主 token，subagent 漏記可由 reconcile 偵測 `subagent_tokens_settled = 0` 標記，不致資料損毀。

## Migration Plan

1. 建置 tt 含 opencode 支援
2. 使用者執行 `tt setup --opencode` 產生 plugin
3. 重啟 opencode（plugin 載入）
4. 開始工作，tt 自動記錄
5. `tt report --since 1d` 查詢 opencode 工時

**回滾：** 刪除 `~/.config/opencode/plugins/tt-bridge.ts` 並重啟 opencode 即停用；tt DB 中已記錄的 opencode turn 保留，report 仍可查詢。

## Open Questions

1. **User message 是否帶 model？** 實作時以 `info.model?.modelID ?? ""` 處理並實測；若不帶，確認 response 階段填入 turn.model 機制覆蓋此情境。
2. **Subagent 相對於主 response 的事件順序？** 實作時優先實測；若 subagent 在主 response 之後完成，評估延遲 flush 或獨立補送命令（可能衍生新 change）。
3. **Plugin 部署本地檔案 vs npm？** 本期採本地檔案；npm 發布另案。
