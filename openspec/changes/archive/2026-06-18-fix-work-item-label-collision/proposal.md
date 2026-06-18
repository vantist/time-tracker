## Why

BY WORK ITEM 報表以純字串 label（branch name 或 work_item）做 groupBy key，
導致不同 repo 的相同 branch name（如 "main"）被合併成同一列，造成跨 repo sessions/cost 混算。

## What Changes

- `internal/report/report.go`：`groupByWorkItem` 改以 `(project, label)` 複合 key 分組；`GroupResult` struct 新增 `Project` 欄位
- `internal/report/html.go`：By Work Item table 新增 Project 欄（顯示 `path.Base(project)`）

## Capabilities

### New Capabilities

（無新功能）

### Modified Capabilities

- `work-item-tagging`：groupBy key 從純字串 label 改為 `(project, label)` 複合 key，Report 輸出的 GroupResult 新增 Project 欄位
- `web-dashboard`：By Work Item table 新增 Project 欄，顯示 `path.Base(project)`

## Impact

- Affected code:
  - Modified: `internal/report/report.go`
  - Modified: `internal/report/html.go`

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-18-brainstorm-work-item-label-collision.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
