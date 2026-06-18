## Context

目前開發者使用 Claude Code 與 Copilot CLI 時，沒有任何工具可以量化實際花費的 AI 時間與 token 成本。這兩個工具都提供 hook 系統，可在事件發生時呼叫外部 binary，這是唯一可靠的計時切入點，不需要 plugin、daemon 或 monkey-patching。

## Goals / Non-Goals

**Goals:**
- 零摩擦：hook 自動呼叫，不需要使用者手動操作
- 精確的 agent 時間（`response_at - prompt_at`，無估計誤差）
- 近似的 user 主動時間（idle gap sum，可接受近似）
- 單一 binary，無外部 runtime 依賴
- SQLite 本機儲存，資料完整保留供重新聚合

**Non-Goals:**
- 即時監控或 dashboard
- 跨機器同步
- 支援 ChatGPT、Gemini 等其他 AI 工具（可未來擴充）
- 精確的 user 時間（session 開著不等於在使用，idle threshold 是合理近似）

## Decisions

### 1. 語言選 Go，不選 Python / Node.js

Hook 呼叫頻率高（每次 prompt、每次 response），啟動時間直接影響使用者感受。Go 單一 binary、冷啟動 < 10ms，Python/Node 需要 runtime，冷啟動 100ms+ 且需要管理虛擬環境。

### 2. SQLite driver 選 `modernc.org/sqlite`（pure Go），不選 `mattn/go-sqlite3`（cgo）

`mattn/go-sqlite3` 需要 cgo，交叉編譯複雜，`goreleaser` 跨平台發布會遇到問題。`modernc.org/sqlite` pure Go，無 cgo 依賴，二進位靜態連結，交叉編譯零摩擦。效能略低（< 2x），但寫入頻率低（每次 prompt/response），完全可接受。

### 3. User 時間用 idle threshold 近似，不精確追蹤

精確追蹤需要 OS 層 idle 監聽（`IOKit` on macOS、`XScreenSaver` on Linux），增加複雜度且需要 daemon。Idle threshold 是業界慣用近似法（toggl、wakatime 都用這個方式）。預設 15 分鐘，可透過 `tt config set idle-threshold <minutes>` 調整。

**計算方式**：在報表聚合時，遍歷同一 session 的 turns，計算相鄰 turns 之間的 gap，gap < idle_threshold 的部分累加為 user 主動時間。

### 4. Session 從 hook payload 取得，不自行管理

Claude Code `UserPromptSubmit` hook 提供 `session_id`，`Stop` hook 提供相同 `session_id`。Session 的 `started_at` = 第一個 prompt 的 `prompt_at`，`ended_at` = 最後一個 response 的 `response_at`（upsert 更新）。不另外追蹤 session 生命週期，簡化狀態管理。

### 5. 定價表 hard-code 在 binary，不用設定檔或遠端 API

設定檔讓使用者誤以為可以自訂定價，造成混淆。遠端 API 需要 network 請求，增加 hook 呼叫延遲。定價表版本隨 binary 更新，使用 `model` 欄位匹配，未知 model 的 `estimated_cost_usd` 寫 NULL。

### 6. 工作項目優先順序：`work_item > branch > "untagged"`

`tt work "login redesign"` 手動標記存入 `~/.tt/work-item`（純文字），每次 `tt record prompt` 讀取並寫入 `sessions.work_item`。Git branch 從 `git branch --show-current` 取得，失敗時 fallback 到 cwd。

## Risks / Trade-offs

- **Hook payload 欄位名稱不確定**：`Stop` hook 的 token 欄位格式（`usage.input_tokens` vs `input_tokens`？）需在實作 `tt record response` 時確認 Claude Code 文件或測試 hook。Copilot CLI `agentStop` payload 是否帶 token 資料也未確認，需要 fallback 到 Chronicle 解析。
  → Mitigation：`estimated_cost_usd` 和 token 欄位允許 NULL，hook 解析失敗時記錄事件但跳過 token 欄位，不中斷 recording。

- **Pure Go SQLite 效能較低**：`modernc.org/sqlite` 在大量讀寫時比 cgo 版本慢 1.5-2x。
  → Mitigation：hook 呼叫只有單筆寫入，報表查詢是 read-only 聚合，實際資料量小（每天數十條 turns），效能不構成問題。

- **Idle threshold 近似誤差**：User 時間是估計值，實際專注時間可能更短或更長。
  → Mitigation：報表明確標示「user active（idle threshold: 15m）」，使用者知道這是近似值。Threshold 可設定，讓使用者根據自己的工作模式調整。

## Migration Plan

初始安裝（無現有資料）：
1. `brew install tt` 或下載 binary
2. `tt setup` 自動寫入 `~/.claude/settings.json`（Claude Code hooks）
3. 手動複製 hook 設定到 `~/.copilot/hooks/`（Copilot CLI）
4. 驗證：`tt report` 應顯示空報表而非錯誤

## Open Questions

- `Stop` hook payload 確切欄位名稱（`usage.input_tokens`？`input_tokens`？）— 需查 Claude Code hook 文件或實測
- Copilot CLI `agentStop` payload 是否帶 token 資料 — 若無，需實作 Chronicle 解析 fallback
- `tt setup` 是否自動備份現有 `~/.claude/settings.json`，或要求使用者手動合併
