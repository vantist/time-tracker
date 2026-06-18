# windows-compat-process-start

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

`tt setup` 寫入 Claude Code hook 的命令依賴 bash-only 語法：

```bash
PROCESS_PID=$PPID PROCESS_START=$(( $(date +%s) - $(ps -p $PPID -o etime= | tr -d ' ' | awk -F'[:-]' '...') )) tt record prompt
```

此命令在 Windows（cmd.exe / PowerShell）完全無法執行，因為：
- `$PPID` — bash 變數
- `$(...)` — bash 命令替換
- `date +%s`, `ps -p`, `awk`, `tr` — Unix 工具

專案建置產出同時目標 macOS 和 Windows（`install.sh` / `install.ps1`），所以 hook 跨平台相容是必要條件。

`PROCESS_PID` 和 `PROCESS_START` 用途：作為穩定 session key（`internal/db/session.go:UpsertSession`），讓同一個 Claude 進程的多次 `/clear` 都對應同一筆 session 記錄。

現有依賴 `golang.org/x/sys v0.42.0` 已在 `go.mod`，可直接使用。

## Decision

把 `PROCESS_PID` / `PROCESS_START` 的計算從 shell hook 移進 `tt` binary。hook 簡化為 `tt record prompt`，`tt` 自己用 `os.Getppid()` + OS API 取得父 process 啟動時間。

## Rationale

- Shell 計算：Unix-only，難以在 Windows 等價實作
- Go binary 計算：`os.Getppid()` 跨平台；process start time 用 `golang.org/x/sys`（已有依賴）的 OS API，有 build tag 分隔實作
- Hook 字串對兩個平台完全相同，`setup.go` 不需要 platform branch

## Approach

新增 `internal/process/` package，提供：

```go
func StartTime(pid int) (int64, error)
```

兩個 build-tag 檔：

- `process_darwin.go` — `golang.org/x/sys/unix.SysctlKinfoProc("kern.proc.pid", pid)` → `kp.Proc.P_starttime.Sec`
- `process_windows.go` — `windows.OpenProcess` + `GetProcessTimes` → CreationTime 轉 Unix 秒
- `process_other.go` (`//go:build !darwin && !windows`) — 回傳 `0, nil`（降級模式，session key 不穩定但不崩潰）

修改點：

1. `internal/setup/setup.go:18` — hook string 改為 `"tt record prompt"`，同時 `Stop` hook 保持 `"tt record response"` 不變
2. `cmd/tt/record.go` — `resolvePromptInputFromEnv()` 改成呼叫 `process.StartTime(os.Getppid())`，移除 `PROCESS_PID` / `PROCESS_START` env var 讀取邏輯

## Design Notes

- `process_other.go` 降級回傳 `(0, nil)`，`UpsertSession` 在 `ProcessPID==0` 時走舊的 INSERT OR IGNORE 路徑，行為與未設 env var 相同，不破壞現有邏輯
- Windows `GetProcessTimes` 回傳 FILETIME（100-nanosecond intervals since 1601-01-01），需轉換為 Unix epoch 秒
- `resolvePromptInputFromEnv()` 函式名稱可重新命名為 `resolveProcessInfo()` 或直接 inline
- 現有 `PROCESS_PID` / `PROCESS_START` env var 支援可保留作 override（測試用），或完全移除——需決定

## Insights to Capture

- `proposal.md`: scope — 新增 `internal/process` package，修改 setup hook string 及 record.go
- `tasks.md`: (1) 新增 process package with build tags, (2) 修改 record.go 呼叫 process.StartTime, (3) 修改 setup.go hook string, (4) 更新測試

## Open Questions

- `PROCESS_PID` / `PROCESS_START` env var override 要保留還是移除？保留可讓測試注入假值，移除讓程式碼更簡單。
