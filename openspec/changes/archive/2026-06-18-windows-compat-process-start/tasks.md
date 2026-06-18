## 1. internal/process package（TDD）

- [x] 1.1 新增 `internal/process/process_other.go`（build tag `!darwin && !windows`）：`StartTime(pid int) (int64, error)` 回傳 `(0, nil)`
- [x] 1.2 新增 `internal/process/process_darwin_test.go`：在 darwin 呼叫 `StartTime(os.Getppid())` 應回傳正整數且 error 為 nil；呼叫 `StartTime(-1)` 應回傳 non-nil error
- [x] 1.3 新增 `internal/process/process_darwin.go`（build tag `darwin`）：使用 `syscall.SysctlRaw("kern.proc.pid", pid)` 解析 `syscall.KinfoProc.Proc.P_starttime.Sec`
- [x] 1.4 新增 `internal/process/process_windows_test.go`：FILETIME 轉換邏輯單元測試（驗證 1601-01-01 epoch 轉換公式）
- [x] 1.5 新增 `internal/process/process_windows.go`（build tag `windows`）：使用 `golang.org/x/sys/windows.OpenProcess` + `GetProcessTimes` 將 FILETIME 轉換為 Unix 秒

## 2. cmd/tt/record.go 整合（TDD）

- [x] 2.1 更新 `cmd/tt/record_test.go`：新增測試驗證「env var 均未設定時，呼叫 `process.StartTime(os.Getppid())` 並使用其結果」（可用 `PROCESS_PID=""` 觸發此路徑）
- [x] 2.2 修改 `cmd/tt/record.go`：`resolvePromptInputFromEnv()`（或重新命名後的函式）改為：先檢查 `PROCESS_PID` / `PROCESS_START` env var，若均非空則使用；否則呼叫 `os.Getppid()` 和 `process.StartTime(ppid)`
- [x] 2.3 確認現有 `cmd/tt/record_test.go` 中依賴 env var 的測試全數通過

## 3. internal/setup/setup.go hook 字串

- [x] 3.1 更新 `internal/setup/setup_test.go`：新增或修改測試驗證 prompt hook 字串為 `tt record prompt`（不含 `$PPID`、`date`、`ps`、`awk`）
- [x] 3.2 修改 `internal/setup/setup.go`：將 UserPromptSubmit hook 命令從含 bash 計算的長字串改為 `tt record prompt`

## 4. 驗收

- [x] 4.1 執行 `go test ./internal/process/...` 確認所有 process package 測試通過
- [x] 4.2 執行 `go test ./cmd/tt/... ./internal/setup/...` 確認所有相關測試通過
- [x] 4.3 執行 `go build ./...` 確認無編譯錯誤（包含 darwin/windows/other build tags）
- [x] 4.4 手動執行 `tt setup` 並確認寫入的 hook 字串為 `tt record prompt`
