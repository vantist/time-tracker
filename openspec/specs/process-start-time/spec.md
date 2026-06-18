# process-start-time Specification

## Purpose
TBD - created by archiving change windows-compat-process-start. Update Purpose after archive.
## Requirements
### Requirement: 跨平台取得 process 啟動時間

`internal/process` package 的 `StartTime(pid int) (int64, error)` 函式 SHALL 回傳指定 PID 的 process 啟動 Unix timestamp（秒）。

- 在 darwin：SHALL 使用 `syscall.SysctlRaw("kern.proc.pid", pid)` 解析 `kinfo_proc.Proc.P_starttime.Sec`
- 在 windows：SHALL 使用 `windows.OpenProcess` + `GetProcessTimes` 將 FILETIME 轉換為 Unix 秒（FILETIME epoch 為 1601-01-01，需減去 11644473600 秒）
- 在其他平台：SHALL 回傳 `(0, nil)`（降級模式，不崩潰）
- 若取得失敗（無權限、PID 不存在）：SHALL 回傳 error

#### Scenario: darwin 成功取得父 process 啟動時間

- **WHEN** 在 darwin 呼叫 `StartTime(os.Getppid())`
- **THEN** 回傳正整數 Unix timestamp 且 error 為 nil

#### Scenario: 非 darwin/windows 平台降級

- **WHEN** 在 linux 或其他平台呼叫 `StartTime(pid)`
- **THEN** 回傳 `(0, nil)`，不 panic，不回傳 error

#### Scenario: 無效 PID

- **WHEN** 呼叫 `StartTime(-1)` 或不存在的 PID
- **THEN** 在 darwin/windows 回傳 non-nil error；在 other 平台仍回傳 `(0, nil)`

### Requirement: record 命令自行解析 parent process 資訊

`tt record prompt` 的 `cmd/tt/record.go` SHALL 依下列優先順序取得 ProcessPID 和 ProcessStart：

1. 若環境變數 `PROCESS_PID` 和 `PROCESS_START` 均已設定且非空，SHALL 使用這兩個值（測試注入 override）
2. 否則，SHALL 呼叫 `os.Getppid()` 取得 PID，並呼叫 `process.StartTime(ppid)` 取得啟動時間

#### Scenario: env var override 存在時優先使用

- **WHEN** 環境變數 `PROCESS_PID=1234` 且 `PROCESS_START=1718000000` 均已設定
- **THEN** `tt record prompt` 使用 ProcessPID=1234、ProcessStart=1718000000，不呼叫 `process.StartTime`

#### Scenario: env var 未設定時自動解析

- **WHEN** 環境變數 `PROCESS_PID` 和 `PROCESS_START` 均未設定
- **THEN** `tt record prompt` 呼叫 `os.Getppid()` 和 `process.StartTime`，以取得的值建立 session

#### Scenario: process.StartTime 回傳 0（降級）

- **WHEN** 在非 darwin/windows 平台且 env var 未設定
- **THEN** `tt record prompt` 使用 ProcessStart=0，`UpsertSession` 走 INSERT OR IGNORE 路徑，不崩潰

### Requirement: setup hook 使用跨平台命令字串

`internal/setup/setup.go` 的 prompt hook 字串 SHALL 為 `tt record prompt`，不含任何 bash shell 替換語法（`$(...)`、`$((...))` 等）。

#### Scenario: setup 寫入的 hook 在 Windows 可執行

- **WHEN** 在 Windows 執行 `tt setup` 後，Claude Code 觸發 UserPromptSubmit hook
- **THEN** hook 執行 `tt record prompt`，命令可在 cmd.exe / PowerShell 正常呼叫，不因 bash 語法報錯

#### Scenario: setup 寫入的 hook 在 macOS 行為不變

- **WHEN** 在 macOS 執行 `tt setup` 後，Claude Code 觸發 UserPromptSubmit hook
- **THEN** hook 執行 `tt record prompt`，`tt` binary 自行取得 parent process 資訊，session 記錄正確建立

