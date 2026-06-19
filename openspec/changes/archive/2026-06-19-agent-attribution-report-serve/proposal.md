## Why

目前 `tt report` 與 `tt serve` 沒標示與區分 AI Agent 名稱，導致 Claude Code 與 Copilot CLI 的數據全部混在一起，無法單獨統計不同 AI 工具的使用時間與 token 費用。

## What Changes

- 在終端機報表（`tt report`）與網頁儀表板（`tt serve`）新增 AI Agent 名稱欄位與 By Agent 的彙整統計表格。
- 對 Agent 名稱進行正規化處理（如 `claude-code` 轉為 `Claude Code`，空值轉換為 `unknown`）。

## Capabilities

### New Capabilities

- `agent-attribution`: 區分與彙整統計不同 AI Agent 的工作時間與 Token 費用。

### Modified Capabilities

- `report-query`: 擴充 `tt report` 命令輸出，新增 Agent 欄位及 `By Agent` 統計表格。
- `web-dashboard`: 擴充網頁儀表板，新增 By Agent 區塊以及在 Sessions 列表中新增 Agent 欄位。

## Impact

- Affected specs:
  - `openspec/specs/agent-attribution/spec.md`
  - `openspec/specs/report-query/spec.md`
  - `openspec/specs/web-dashboard/spec.md`
- Affected code:
  - New: (none)
  - Modified:
    - `internal/report/report.go`
    - `internal/report/html.go`
  - Removed: (none)

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-19-brainstorm-agent-attribution.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
