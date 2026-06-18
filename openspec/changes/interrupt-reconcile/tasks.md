## 1. 建立 internal/transcript package

- [x] 1.1 讀 `cmd/tt/record.go` 確認 `extractSubagentTokens` 的 subagent meta 路徑邏輯是否依賴 cmd 層 context
- [x] 1.2 建立 `internal/transcript/extract.go`，將 `loadTranscript`、`sumWindow`、`extractFromTranscriptAtOffset`（改名 `ExtractWindow`）移入，export 為公開函式
- [x] 1.3 將 `extractSubagentTokens` 移入 `internal/transcript/extract.go`，確認路徑邏輯獨立於 cmd 層
- [x] 1.4 寫 `internal/transcript/extract_test.go`：測試 `ExtractWindow` 在 from/to 範圍正確、to=-1 讀到 EOF、路徑不存在時回傳 error（TDD：先寫 failing test）
- [x] 1.5 修改 `cmd/tt/record.go` 改 import `internal/transcript`，刪本地重複實作，確認現有測試通過

## 2. 建立 internal/reconcile package（鎖機制）

- [x] 2.1 建立 `internal/reconcile/lock.go`：定義 `lockPath()` 回傳 `~/.tt/reconcile.lock`
- [x] 2.2 建立 `internal/reconcile/lock_unix.go`（build tag `//go:build !windows`）：實作 `tryLock(path string) (unlock func(), ok bool)`，用 `golang.org/x/sys/unix.Flock` 配合 `LOCK_EX|LOCK_NB`
- [x] 2.3 建立 `internal/reconcile/lock_windows.go`（build tag `//go:build windows`）：實作 `tryLock` 用 `golang.org/x/sys/windows.LockFileEx`
- [x] 2.4 寫 `internal/reconcile/lock_test.go`：測試同一 process 兩次 tryLock 時第二次 ok=false（TDD：先寫 failing test）

## 3. 實作 reconcile 核心邏輯

- [x] 3.1 建立 `internal/reconcile/reconcile.go`：實作 `hasActiveSession(conn *sql.DB) bool`
- [x] 3.2 實作私有 `reconcile(conn *sql.DB)`：執行設計文件 D5 的 SQL 查詢，loop 每個懸空 turn
- [x] 3.3 在 `reconcile` loop 中加入進行中 turn 的跳過邏輯（`isLatest && processAlive → continue`）
- [x] 3.4 在 `reconcile` loop 中呼叫 `internal/transcript.ExtractWindow` 提取 token
- [x] 3.5 實作 `response_at` 估算：中間 turn 用 `next_prompt_at - 1ms`，最後 turn 用 transcript mtime
- [x] 3.6 執行 `UPDATE turns SET response_at=?, input_tokens=?, output_tokens=?, cache_read_tokens=?, cache_creation_tokens=?, estimated_cost_usd=? WHERE id=? AND response_at IS NULL`
- [x] 3.7 實作 `MaybeReconcile(conn *sql.DB)`：in-process `sync.Mutex.TryLock` + `tryLock(lockPath())`，成功才呼叫 `reconcile`
- [x] 3.8 寫 `internal/reconcile/reconcile_test.go`：測試懸空 turn 補算後 response_at 不為 NULL、進行中 turn skip、idempotency（補算前已有 response_at 的 row 不被修改）（TDD：先寫 failing tests）

## 4. 整合觸發點

- [ ] 4.1 修改 `cmd/tt/serve_cmd.go`：在 HTTP server 啟動前呼叫 `reconcile.MaybeReconcile(conn)`
- [ ] 4.2 修改 `cmd/tt/report_cmd.go`：在輸出報告前呼叫 `reconcile.MaybeReconcile(conn)`
- [ ] 4.3 修改 `/api/report` handler（`cmd/tt/api.go` 或相關 serve handler）：呼叫前先 `hasActiveSession`，false 才呼叫 `MaybeReconcile`

## 5. 驗證

- [ ] 5.1 執行 `go build ./...` 確認全 package 無編譯錯誤（含 windows build tag）
- [ ] 5.2 執行 `go test ./internal/transcript/...` 確認 extract 測試全過
- [ ] 5.3 執行 `go test ./internal/reconcile/...` 確認 reconcile 測試全過
- [ ] 5.4 執行 `go test ./cmd/tt/...` 確認 record.go 重構後現有測試全過
- [ ] 5.5 手動驗證：跑一個 Claude Code session → Escape 中斷 → 執行 `tt report` → 確認 token 與成本已補算
