## Context

`tt record prompt` 命令目前透過環境變數 `PROCESS_PID` 和 `PROCESS_START` 取得父 process 資訊，這兩個值由 Claude Code hook 中的 bash shell 命令計算注入。此 bash 計算在 Windows 無法執行，導致 hook 在 Windows 完全失效。

現有依賴 `golang.org/x/sys v0.42.0` 已在 `go.mod`，可直接使用其 OS-specific API，無需新增外部依賴。

## Goals / Non-Goals

**Goals:**

- Hook 字串簡化為純粹的 `tt record prompt`，不含任何 shell 運算
- `tt` binary 自行取得 parent process PID 和啟動時間
- 維持現有 `PROCESS_PID` / `PROCESS_START` env var 作為測試注入 override
- 降級模式（非 darwin/windows）不崩潰，session key 退化為不穩定但功能可用

**Non-Goals:**

- Linux 的準確 process start time（回傳 0，走降級路徑即可）
- 變更 `UpsertSession` 的資料庫語義
- 支援追蹤非父 process 的任意 PID

## Decisions

### D1：新增獨立 `internal/process/` package，而非 inline 在 record.go

**選擇**：獨立 package，build-tag 隔離各平台實作。

**理由**：`record.go` 已承擔 prompt 解析邏輯；OS API 呼叫屬於底層平台關注點，分離後各自可獨立測試。build-tag 讓 Go toolchain 自動選擇正確實作，無需 runtime `GOOS` 判斷。

**替代方案**：`runtime.GOOS` switch — 需要所有平台程式碼共存於同一編譯單元，Windows API type 在 non-Windows 編譯時會報錯。

### D2：darwin 實作使用 `syscall` 取代 `golang.org/x/sys/unix`

**選擇**：`syscall.Sysctl` 系列函式（標準庫）。

**理由**：`golang.org/x/sys/unix.SysctlKinfoProc` 的 `kinfo_proc` struct 在不同 Darwin 版本 layout 有差異；`syscall.SysctlRaw` 搭配 `syscall.KinfoProc` 是更穩定的介面，且不需要 x/sys 版本追蹤。

**替代方案**：`golang.org/x/sys/unix` — 已有依賴，但 struct 欄位存取較脆弱。

### D3：保留 PROCESS_PID / PROCESS_START env var 作為 override

**選擇**：保留，env var 存在時優先使用，否則呼叫 `process.StartTime`。

**理由**：現有測試透過 env var 注入假值，保留可讓測試無需改動 `internal/process` mock。移除會強迫所有測試改用介面注入或 process-level 隔離，成本高於保留。

## Risks / Trade-offs

- **Darwin struct layout 差異** → 以 `syscall.KinfoProc` 為準，加整合測試在 CI 上執行驗證
- **Windows FILETIME 轉換** → 明確記錄轉換公式（100ns intervals from 1601-01-01）並加單元測試驗證邊界值
- **process_other.go 回傳 0** → `UpsertSession` 在 `ProcessPID==0` 時走舊的 INSERT OR IGNORE 路徑，session key 不穩定，但不崩潰；Linux 使用者需接受此限制

## Migration Plan

Hook 字串變更為 `tt record prompt` 後，舊版 hook（含 bash 計算）仍可執行但會傳入舊的 env var，`record.go` 的 env var override 路徑會接收這些值，行為不變。使用者重新執行 `tt setup` 後即切換至新 hook。無資料遷移需求。

## Open Questions

（無）
