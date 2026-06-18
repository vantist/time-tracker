## Why

`tt setup` 寫入的 Claude Code hook 命令依賴 bash-only 語法（`$PPID`、`$(date +%s)`、`ps`、`awk`），在 Windows cmd.exe / PowerShell 完全無法執行。專案同時目標 macOS 和 Windows，hook 跨平台相容是必要條件。

## What Changes

- 新增 `internal/process/` package，提供跨平台 `StartTime(pid int) (int64, error)` 函式（darwin / windows / other 三個 build-tag 實作）
- `cmd/tt/record.go`：`resolvePromptInputFromEnv()` 改為呼叫 `process.StartTime(os.Getppid())`；保留 `PROCESS_PID` / `PROCESS_START` env var 作為測試注入 override
- `internal/setup/setup.go`：hook string 從含 shell 計算的長命令簡化為 `tt record prompt`

## Capabilities

### New Capabilities

- `process-start-time`: 跨平台取得指定 PID 的 process 啟動 Unix timestamp（darwin 用 sysctl kinfo_proc、windows 用 GetProcessTimes、其他回傳 0 降級）

### Modified Capabilities

（無 spec-level 行為變更）

## Impact

- Affected code:
  - New: `internal/process/process_darwin.go`, `internal/process/process_windows.go`, `internal/process/process_other.go`
  - Modified: `cmd/tt/record.go`, `cmd/tt/record_test.go`, `internal/setup/setup.go`, `internal/setup/setup_test.go`
  - Removed: （無）

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-18-brainstorm-windows-compat-process-start.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
