## Why

tt 目前支援 Claude Code、Copilot CLI、Antigravity、Codex，但尚未支援 opencode。opencode 使用者無法用 tt 記錄 token 用量與工時，形成費用與時間盲區。opencode 的 plugin 事件系統已內含完整 token 資料（不需 transcript parser），可直接重用 tt 現有 pricing / DB / report 基礎建設，以最小改動延伸支援。

## What Changes

- 新增 `opencode` 為 tt 支援的 AI 工具：`tt record prompt/response --tool opencode`
- `tt record response` 新增 `--subagent-tokens` flag，接收 subagent token JSON 陣列並寫入 `turn_model_usages`
- `tt record response` 於 `--tool opencode` 時跳過 transcript parser fallback（tokens 直接來自 event flag）
- `tt record prompt` 於 `--tool opencode` 時不要求 transcript path
- 新增 `tt setup --opencode`：產生 `~/.config/opencode/plugins/tt-bridge.ts` 本地 plugin，冪等不重複產生
- opencode plugin 監聽 `message.updated` event，將 user prompt / assistant response 對應到 `tt record` 呼叫，含串流去重與 subagent token 暫存
- `normalizeAgentName` 新增 `"opencode"` → `"OpenCode"` 正規化
- `tt setup` 多工具 flag 與自動偵測延伸至 opencode（`--opencode` flag、偵測 `~/.config/opencode/`）

## Non-Goals

- 不發布 npm package：plugin 先以本地檔案產生，未來再考慮 npm 發布
- 不為 opencode 撰寫 transcript parser：opencode token 來自 event，不需解析 JSONL
- 不改動現有 Claude Code / Copilot CLI / Antigravity / Codex 的錄製路徑
- 不處理 opencode plugin 事件順序邊界案例（subagent晚於主 response 完成）的補償邏輯——待實測確認後另案處理

## Capabilities

### New Capabilities

- `opencode-integration`: opencode 工具介接卡——TS plugin 監聽 `message.updated` 事件並橋接至 `tt record prompt/response --tool opencode`，涵蓋事件→flag 對應、subagent token 暫存與隨主 response flush、串流中間態防護與去重、以及 `tt setup --opencode` 冪等產生 plugin 檔。

### Modified Capabilities

- `event-recording`: `tt record prompt` 與 `tt record response` 接受 `tool == "opencode"`；response 透過 `--tokens` flag 直接取得 token（不回退 transcript parser）；prompt 不需 transcript path。
- `subagent-token-capture`: subagent token 可透過 `--subagent-tokens` JSON flag 提供（opencode event 路徑）並寫入 `turn_model_usages`，與既有 transcript 掃描路徑並存。
- `agent-attribution`: `normalizeAgentName` 新增 `"opencode"` → `"OpenCode"` 正規化規則。
- `setup-improvements`: `tt setup` 支援 `--opencode` flag 與 `~/.config/opencode/` 自動偵測。

## Impact

- Affected specs: `opencode-integration` (new), `event-recording`, `subagent-token-capture`, `agent-attribution`, `setup-improvements`
- Affected code:
  - New:
    - `internal/setup/opencode.go` — `SetupOpencode` 函式與 plugin 範本
    - `internal/transcript/opencode.go` — `"opencode"` LogProvider 空實作（註冊用，token 不來自 transcript）
  - Modified:
    - `cmd/tt/record.go` — `resolvePromptInput` / `resolveResponseInput` 處理 `tool == "opencode"`；新增 `--subagent-tokens` flag
    - `internal/recorder/response.go` — 解析 `--subagent-tokens` JSON 並 INSERT `turn_model_usages`
    - `cmd/tt/setup_cmd.go` — 新增 `--opencode` flag 並呼叫 `SetupOpencode`
    - `cmd/tt/report_cmd.go` — `normalizeAgentName` 新增 `"opencode"` 分支
    - `internal/setup/setup.go` — 多工具 setup 調度與自動偵測納入 opencode
  - Removed: (none)

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-23-brainstorm-opencode-support.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
