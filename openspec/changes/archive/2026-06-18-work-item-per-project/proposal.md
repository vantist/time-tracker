## Why

`tt work` 目前把 work item 存在單一全域檔案 `~/.tt/work-item`，不同 repo 無法各自追蹤不同工作項目。需要讓 work item 以 git repo 為單位隔離，才能在多個專案間切換時保留各自狀態。

## What Changes

- `workitem.Get` / `Set` / `Clear` 新增 `project string` 參數，儲存路徑改為 `~/.tt/work-items/<sha256[:16]-of-project-path>`
- 新增 `resolveProject(dir string) string`：優先取 git root，非 git 目錄則以 CWD 為 key
- `tt work` CLI 傳入 `os.Getwd()` 作為 project 參數
- `recorder.RecordPrompt` 改從 `input.Project`（已為 CWD）取 work item
- 舊全域檔案 `~/.tt/work-item` 不遷移、直接忽略（無法安全對應 project）

## Capabilities

### New Capabilities

- `work-item-per-project`：per-project work item 儲存與查詢，以 git root 或 CWD 為 key，雜湊後存於 `~/.tt/work-items/`

### Modified Capabilities

（無 — 現有 spec 層級行為不變，僅實作細節改變）

## Impact

- Affected specs: `work-item-per-project`（新）
- Affected code:
  - New: `openspec/specs/work-item-per-project/spec.md`
  - Modified: `internal/workitem/workitem.go`, `internal/workitem/workitem_test.go`, `cmd/tt/work.go`, `internal/recorder/recorder.go`
  - Removed: 無

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-18-brainstorm-work-item-per-project.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
