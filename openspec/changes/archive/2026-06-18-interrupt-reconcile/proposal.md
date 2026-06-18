## Why

使用者按 Escape 中斷 Claude Code 操作時，`Stop` hook **不觸發**，導致 turn 的 `response_at` 永遠為 NULL。Token 消耗已實際發生（API 已呼叫），但不記錄進 DB，造成 `tt report` / `tt serve` 的成本與時間低估。

## What Changes

- 新增 `internal/transcript/` package，將 transcript 提取邏輯從 `cmd/tt/record.go` 抽出共用
- 新增 `internal/reconcile/` package，實作 `MaybeReconcile(conn)`：掃描 `response_at IS NULL` 的懸空 turn，從 transcript 補算 token 與 response_at，寫回 DB
- `tt serve` 啟動時呼叫 `MaybeReconcile`（無條件補歷史懸空）
- `/api/report` 每次 refresh 若有 active session 則呼叫 `MaybeReconcile`
- `tt report` 執行前呼叫 `MaybeReconcile`（無條件）
- Cross-process flock（`~/.tt/reconcile.lock`）+ in-process `sync.Mutex` 防並發重入

## Capabilities

### New Capabilities

- `interrupt-reconcile`: 偵測並補算中斷 turn 的 token 消耗與結束時間，使 report/serve 成本數據準確

### Modified Capabilities

（none）

## Impact

- Affected code:
  - New: `internal/transcript/extract.go`
  - New: `internal/reconcile/reconcile.go`
  - New: `internal/reconcile/lock.go`
  - New: `internal/reconcile/lock_unix.go`
  - New: `internal/reconcile/lock_windows.go`
  - Modified: `cmd/tt/record.go` — 改 import `internal/transcript`，刪本地重複實作
  - Modified: `cmd/tt/serve_cmd.go` — 啟動時呼叫 `MaybeReconcile`
  - Modified: `cmd/tt/report_cmd.go` — 執行前呼叫 `MaybeReconcile`
  - Modified: `cmd/tt/api.go`（或 serve handler） — `/api/report` refresh 時依 active session 呼叫 `MaybeReconcile`

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-18-brainstorm-interrupt-reconcile.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
