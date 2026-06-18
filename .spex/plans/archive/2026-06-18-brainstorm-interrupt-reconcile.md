# Interrupt Reconcile：補齊中斷 turn 的 token 與時間

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). -->

## Context

使用者按 Escape 中斷 agent 操作時，`Stop` hook **不觸發**。
結果：
- `prompt_at` 已寫入（`UserPromptSubmit` 已觸發）
- `response_at` 永遠為 NULL（懸空 turn）
- Token 消耗已發生（API 已呼叫），但不記錄進 DB
- `tt report` / `tt serve` 成本與時間低估

Transcript JSONL 仍記錄所有 assistant entries，token 資料可事後補算。

## Decision

**B+**：`tt serve` 啟動時無條件 reconcile 一次；`/api/report` 每次 refresh 若有 active session 則 reconcile；`tt report` 執行前 reconcile 一次。

Reconcile 定義：掃描 `response_at IS NULL` 的懸空 turn，從 transcript 提取 token 窗口，估算 response_at，寫回 DB。

## Rationale

- `response_at IS NULL` 是 idempotent guard：Stop hook 若先寫入，reconcile UPDATE 找不到符合 row → no-op
- Transcript 已有 `prompt_line_offset` anchor，token 窗口可精確切割
- `process.IsAlive` 保護進行中的 turn 不被誤算

## 設計細節

### Part 1: 觸發邏輯

| 觸發點 | hasActiveSession 檢查 | 原因 |
|--------|----------------------|------|
| `tt serve` 啟動 | 否 | 補歷史懸空，無條件 |
| `/api/report` 每次 | 是 | 避免 60 秒無意義鎖競爭 |
| `tt report` | 否 | one-shot，跑一次就結束 |

`hasActiveSession`：
```sql
SELECT process_pid, process_start FROM sessions WHERE ended_at IS NULL
```
任一 row 的 `process.IsAlive(pid, start)` 為 true → 有 active session。

### Part 2: Reconcile 核心

**Step 1: 找懸空 turn**
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

**Step 2: 跳過進行中的 turn**
```go
isLatest := row.nextOffset == nil
processAlive := process.IsAlive(row.ProcessPID, row.ProcessStart)
if isLatest && processAlive {
    continue // 正在進行中，不動
}
```

**Step 3: Token 提取**

新增 `extractWindow(path string, from, to int) (tokensJSON, model string)`：
- `to == -1` → EOF
- 中間 turn：`extractWindow(path, offset, nextOffset)`
- 最後 turn（process dead）：`extractWindow(path, offset, -1)`

**Step 4: response_at 估算**

| 情況 | 值 |
|------|-----|
| 中間 turn | `next_prompt_at - 1ms` |
| 最後 turn（process dead） | transcript 檔案 mtime |

**Step 5: 寫回 DB**
```sql
UPDATE turns
SET response_at = ?,
    input_tokens = ?, output_tokens = ?,
    cache_read_tokens = ?, cache_creation_tokens = ?,
    estimated_cost_usd = ?
WHERE id = ? AND response_at IS NULL
```

### Part 3: MaybeReconcile 實作

```go
var reconcileMu sync.Mutex

func MaybeReconcile(conn *sql.DB) {
    if !reconcileMu.TryLock() { return }
    defer reconcileMu.Unlock()

    unlock, ok := tryLock(lockPath()) // flock cross-process
    if !ok { return }
    defer unlock()

    reconcile(conn)
}
```

鎖策略：
| 層 | 機制 | 保護對象 |
|----|------|---------|
| `sync.Mutex` | in-process | `tt serve` 內的 goroutine 並發 |
| `flock(LOCK_NB)` | cross-process | `tt serve` vs `tt report` 並發 |

Lock file：`~/.tt/reconcile.lock`

### Part 4: Package 結構

**新增 `internal/reconcile/`**
```
internal/reconcile/
  reconcile.go        ← MaybeReconcile, reconcile, hasActiveSession
  lock.go             ← lockPath()
  lock_unix.go        ← tryLock (build tag !windows)，用 golang.org/x/sys/unix.Flock
  lock_windows.go     ← tryLock (build tag windows)，用 golang.org/x/sys/windows.LockFileEx
```

**新增 `internal/transcript/`**（共用 extract 邏輯）
```
internal/transcript/
  extract.go   ← loadTranscript, sumWindow, extractWindow, extractSubagentTokens
```

`cmd/tt/record.go` 改 import `internal/transcript`，刪本地重複實作。

**修改現有檔案**

| 動作 | 路徑 |
|------|------|
| 新增 | `internal/transcript/extract.go` |
| 新增 | `internal/reconcile/reconcile.go` |
| 新增 | `internal/reconcile/lock.go` |
| 新增 | `internal/reconcile/lock_unix.go` |
| 新增 | `internal/reconcile/lock_windows.go` |
| 修改 | `cmd/tt/record.go` → import transcript，刪本地實作 |
| 修改 | `cmd/tt/serve_cmd.go` → 呼叫 MaybeReconcile |
| 修改 | `cmd/tt/report_cmd.go` → 呼叫 MaybeReconcile |

## Open Questions

- `extractSubagentTokens` 目前在 `cmd/tt/record.go`，移到 `internal/transcript` 時需確認 subagent meta 路徑邏輯可獨立於 cmd 層。
- `response_at` 用 mtime 估算最後 turn 的精確度：mtime 是 transcript 最後寫入時間，若 Claude Code crash 而非正常結束，mtime 可能比實際結束時間早或晚——可接受範圍內的誤差。
