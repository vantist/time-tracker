## 1. DB Migration

- [x] 1.1 在 `internal/db/schema.go` 的 `migrate()` 中新增 `ALTER TABLE sessions ADD COLUMN IF NOT EXISTS process_pid INTEGER`（SQLite 語法：`ADD COLUMN`，不支援 `IF NOT EXISTS`，改用 `PRAGMA table_info` 判斷或直接使用 migration version 機制）
- [x] 1.2 新增 `ALTER TABLE sessions ADD COLUMN process_start INTEGER`（Unix epoch 秒）
- [x] 1.3 新增 `ALTER TABLE sessions ADD COLUMN conversation_id TEXT`（原 session_id UUID）
- [x] 1.4 撰寫測試：`TestMigrate_NewColumns`，對一個舊版 DB（僅有原始欄位）執行 migration 後，確認三個新欄位存在且現有 rows 值為 NULL

## 2. DB 層 UpsertSession 更新

- [x] 2.1 在 `internal/db/session.go` 的 `Session` struct 新增欄位：`ProcessPID int64`、`ProcessStart int64`、`ConversationID string`
- [x] 2.2 修改 `UpsertSession`：當 `ProcessPID != 0` 且 `ProcessStart != 0` 時，改以 `(process_pid, process_start)` 為 key 執行 upsert（`INSERT OR IGNORE`，然後 UPDATE `conversation_id`、`ended_at`）；否則保持原行為（以 `id` 為 key）
- [x] 2.3 撰寫測試：`TestUpsertSession_StableKey`，模擬同一 `(process_pid, process_start)` 但不同 `conversation_id` 的兩次呼叫，確認第二次不建立新 session 而是更新 `conversation_id`
- [x] 2.4 撰寫測試：`TestUpsertSession_FallbackToID`，`ProcessPID = 0` 時確認退回原有 id-based upsert 行為

## 3. Recorder 層更新

- [x] 3.1 在 `internal/recorder/recorder.go` 的 `PromptInput` struct 新增 `ProcessPID int64`、`ProcessStart int64`
- [x] 3.2 修改 `RecordPrompt`：從 `PromptInput` 取得 `ProcessPID`、`ProcessStart`，傳入 `db.UpsertSession`；`turns` 的 `session_id` 仍使用 `input.SessionID`（conversation-level）
- [x] 3.3 撰寫測試：`TestRecordPrompt_StableSession`，確認同一 `(ProcessPID, ProcessStart)` 不同 `SessionID` 的兩次呼叫僅產生一個 session、兩個 turn

## 4. CLI `tt record prompt` 讀取 env var

- [ ] 4.1 在 `cmd/tt/record.go` 的 `resolvePromptInput()` 中，讀取 `os.Getenv("PROCESS_PID")` 與 `os.Getenv("PROCESS_START")` 並轉為 `int64`
- [ ] 4.2 將讀取到的值填入 `recorder.PromptInput.ProcessPID` 與 `recorder.PromptInput.ProcessStart`
- [ ] 4.3 若 `PROCESS_START` 解析失敗或為 0，在 `stderr` 輸出警告 `tt: PROCESS_START empty or invalid, session key may be unstable`，並將 `ProcessStart` 設 0（讓 UpsertSession 降級）
- [ ] 4.4 撰寫測試：`TestResolvePromptInput_EnvVars`，設定 env var 後確認 `PromptInput` 的 `ProcessPID`/`ProcessStart` 正確填入

## 5. Hook 設定更新

- [ ] 5.1 修改 `internal/setup/setup.go`，將 `UserPromptSubmit` hook command 改為：
  ```
  PROCESS_PID=$PPID PROCESS_START=$(( $(date +%s) - $(ps -p $PPID -o etimes= | tr -d ' ') )) tt record prompt
  ```
- [ ] 5.2 手動在 macOS 環境驗證 hook 指令可正確取得 `PROCESS_PID` 與 `PROCESS_START`（非零值）
- [ ] 5.3 撰寫測試：`TestSetupClaudeCode_HookCommand`，確認 `settings.json` 寫入的 `UserPromptSubmit` hook command 包含 `PROCESS_PID` env var 前綴

## 6. Session 工作時間計算更新

- [ ] 6.1 確認現有 `report` 或 `dashboard` 查詢 session 工作時間的 SQL（在 `internal/db/` 或 `internal/report/` 中搜尋），確認是否以 `session_id` 為 group key
- [ ] 6.2 若工作時間以 `session.id` 計算（舊行為），更新查詢改以 `(process_pid, process_start)` 分組（有值時）或 fallback 到 `session.id`（NULL 的舊資料）
- [ ] 6.3 撰寫測試：`TestSessionDuration_SpanConversations`，確認同一工作 session（相同 process key）的多段 turns 計算出正確的累計工作時間

## 7. 整合驗證

- [ ] 7.1 執行 `go test ./...` 確認全部測試通過
- [ ] 7.2 手動執行 `tt setup` 確認 `~/.claude/settings.json` 的 hook 指令已更新
- [ ] 7.3 觸發一次 Claude Code UserPromptSubmit hook，確認 `~/.tt/data.db` 中 session 的 `process_pid`、`process_start`、`conversation_id` 欄位有值
