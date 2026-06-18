## Context

Claude Code 的 `Stop` hook 在使用者按 Escape 中斷時**不觸發**。`UserPromptSubmit` hook 已在 turn 開始時寫入 `prompt_at`，但 `response_at` 永遠留 NULL（懸空 turn）。API token 消耗已發生，但 DB 不記錄，導致 `tt report` / `tt serve` 成本低估。

Transcript JSONL 記錄所有 assistant entries（含 token usage）。每個 turn 已儲存 `prompt_line_offset`（transcript 行號錨點），可事後精確切割 token 窗口。

現有提取邏輯（`extractFromTranscriptAtOffset`、`extractSubagentTokens`）位於 `cmd/tt/record.go`，無法被其他 package 重用。

## Goals / Non-Goals

**Goals:**

- 將 transcript 提取邏輯移至 `internal/transcript/`，可被 cmd 層與 reconcile 層共用
- 實作 `MaybeReconcile(conn *sql.DB)`：補算懸空 turn 的 token 與 response_at
- 三個觸發點：`tt serve` 啟動、`/api/report` refresh（有 active session 時）、`tt report` 執行前
- 雙層鎖防並發：in-process `sync.Mutex` + cross-process `flock(LOCK_NB)`

**Non-Goals:**

- 即時監聽 transcript 變化（非 fsnotify/watch 方案）
- 修改 transcript 格式或 Claude Code hook 行為
- Windows 以外平台的 flock 替代（lock_windows.go 用 `LockFileEx`）

## Decisions

### D1: 觸發策略（B+）

| 觸發點 | hasActiveSession 檢查 | 原因 |
|--------|----------------------|------|
| `tt serve` 啟動 | 否 | 補歷史懸空，process 已死 |
| `/api/report` refresh | 是 | active session 期間 transcript 仍在寫，skip 避免鎖競爭 |
| `tt report` | 否 | one-shot CLI，跑一次就結束 |

`hasActiveSession`：
```sql
SELECT process_pid, process_start FROM sessions WHERE ended_at IS NULL
```
任一 row 的 `process.IsAlive(pid, start)` 為 true → 有 active session。

替代方案考慮：A（`tt serve` 背景定期跑）→ 額外 goroutine 複雜度，不必要；C（只在 `tt report` 跑）→ `serve` 儀表板資料仍低估，拒絕。

### D2: response_at 估算

| 情況 | 值 |
|------|-----|
| 中間 turn（有後繼 turn） | `next_prompt_at - 1ms` |
| 最後 turn（process dead） | transcript 檔案 mtime |

mtime 精確度：若 Claude Code crash 而非正常結束，mtime 可能偏移數秒——在 cost reporting 精確度範圍內可接受。

### D3: Idempotency Guard

`WHERE id = ? AND response_at IS NULL` — Stop hook 若先寫入，reconcile UPDATE 找不到符合 row → no-op。reconcile 可安全重跑。

### D4: Package 結構

```
internal/transcript/
  extract.go   ← loadTranscript, sumWindow, extractWindow, extractSubagentTokens
internal/reconcile/
  reconcile.go      ← MaybeReconcile, reconcile, hasActiveSession
  lock.go           ← lockPath()
  lock_unix.go      ← build tag !windows, golang.org/x/sys/unix.Flock
  lock_windows.go   ← build tag windows, golang.org/x/sys/windows.LockFileEx
```

`cmd/tt/record.go` 改 import `internal/transcript`，刪本地重複實作。

### D5: Reconcile 核心查詢

```sql
SELECT
  t.id, t.session_id, t.transcript_path, t.prompt_line_offset, t.prompt_at, t.model,
  s.process_pid, s.process_start,
  (SELECT prompt_line_offset FROM turns t2
   WHERE t2.session_id = t.session_id AND t2.id > t.id
   ORDER BY t2.id LIMIT 1) AS next_offset,
  (SELECT prompt_at FROM turns t2
   WHERE t2.session_id = t.session_id AND t2.id > t.id
   ORDER BY t2.id LIMIT 1) AS next_prompt_at
FROM turns t
JOIN sessions s ON s.id = t.session_id
WHERE t.response_at IS NULL
  AND t.transcript_path IS NOT NULL
  AND t.prompt_line_offset IS NOT NULL
```

進行中 turn 的跳過邏輯：
```go
isLatest := row.nextOffset == nil
processAlive := process.IsAlive(row.ProcessPID, row.ProcessStart)
if isLatest && processAlive {
    continue
}
```

### D6: `extractWindow` 函式簽章

```go
func ExtractWindow(path string, from, to int) (tokensJSON string, model string, err error)
```

- `to == -1` → 讀到 EOF
- 中間 turn：`ExtractWindow(path, offset, nextOffset)`
- 最後 turn（process dead）：`ExtractWindow(path, offset, -1)`

## Risks / Trade-offs

- **mtime 誤差** → response_at 估算偏移數秒內，對 cost reporting 可接受
- **flock 在 NFS** → 跨 NFS 的 flock 行為未定義；使用者 home dir 通常為本地 FS，影響有限
- **`extractSubagentTokens` 路徑邏輯** → 目前在 `cmd/tt/record.go` 內隱含 cmd 層路徑假設，抽出至 `internal/transcript` 時需確認 subagent meta 路徑可獨立於 cmd 層（Open Question）

## Open Questions

- `extractSubagentTokens` 的 subagent meta 路徑邏輯是否完全不依賴 cmd 層的 context？需讀 `record.go` 確認後再抽出。
