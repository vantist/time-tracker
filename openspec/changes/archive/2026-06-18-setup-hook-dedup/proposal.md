## Why

`tt setup --claude-code` 每次執行都會 append hook 條目到 `~/.claude/settings.json`，沒有去重機制。重複安裝產生重複條目；hook 內容更新後重新 setup，舊版本也不會被移除。

## What Changes

- `internal/setup/setup.go` 的 `SetupClaudeCode()` 改為 idempotent：先移除 `_owner == "tt"` 的舊條目，再插入新版本
- `ttHooks` 每個 outer entry 加 `"_owner": "tt"` marker
- 補 idempotency 測試

## Capabilities

### New Capabilities

- `idempotent-hook-setup`: `tt setup --claude-code` 多次執行結果與一次相同，hook 內容更新後重新 setup 會取代舊版本而非疊加

### Modified Capabilities

（無）

## Impact

- Affected code:
  - Modified: `internal/setup/setup.go`
  - Modified: `internal/setup/setup_test.go`

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-18-brainstorm-setup-hook-dedup.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
