# Spex Insights

## [spex-debugging] workflow-subagent-tokens-missing — 2026-06-18

### Promote candidates

- [ ] Claude Code transcript `content` blocks live under `message.content`, not top-level
  > **Why**: `extractSubagentTokens` scanned `e.Content` (top-level) which is always nil — transcript JSONL puts tool_use blocks under `e.message.content`. Zero subagent IDs found → all subagent tokens silently dropped.
  > **How to apply**: When parsing Claude Code JSONL entries for tool_use/content blocks, always read `entry.Message.Content`, never `entry.Content`. Verify against a real transcript before writing new struct tags.

## [spex-debugging] token-count-mismatch — 2026-06-18

### Promote candidates

- [ ] reconcile `WHERE` 條件必須涵蓋 `input_tokens IS NULL`，不能只用 `response_at IS NULL`
  > **Why**: Stop hook 可能寫入 response_at 但 tokensJSON 為空（transcript 在 /clear 後 flush 前被讀取、offset 超出行數）。此時 reconcile 的 `WHERE response_at IS NULL` 永遠跳過該 turn，tokens 消失無法補救。
  > **How to apply**: 任何 reconcile/backfill 查詢的 WHERE 條件：`(response_at IS NULL OR input_tokens IS NULL)` — 兩種不完整狀態都需修補。UPDATE 語句依現有 response_at 是否存在而分支：已設只補 tokens，未設則同時寫 response_at + tokens。

## [spex-apply] windows-compat-process-start — 2026-06-18

### Promote candidates

- [ ] `syscall.SysctlRaw`/`KinfoProc` 在 Go 1.26 標準庫不存在；darwin process info 須用 `golang.org/x/sys/unix.SysctlKinfoProc`
  > **Why**: 設計文件說用 `syscall.SysctlRaw` 但 Go 1.26 的 `syscall` 沒有此符號。`unix.SysctlKinfoProc` 更直接且 type-safe。
  > **How to apply**: darwin OS API → 先確認標準 `syscall` 是否有對應符號；不存在時用 `golang.org/x/sys/unix`。

- [ ] Env var composite-key override：parse 失敗應 fallback 而非 early return with partial data
  > **Why**: `PROCESS_PID="abc" PROCESS_START="1234"` 在 early return 路徑下產生 `{ProcessPID:0, ProcessStart:1234}`——無意義組合。程式碼審查確認此為 bug。
  > **How to apply**: 「兩個 env var 組成一個 composite key」的 override 邏輯：兩者都成功 parse 才用 override，否則 fallback。

### Plan deviations

- `process_darwin.go` 用 `unix.SysctlKinfoProc` 代替設計文件所說的 `syscall.SysctlRaw`（Go 1.26 標準庫不存在後者）

---

## 2026-06-18 — setup-hook-dedup [spex-apply]

**Promote candidates:**

- [ ] Write config files with 0o600 (not 0o644) and their containing directories with 0o700
  > **Why**: settings.json can hold MCP env vars with API keys. 0o644 makes it world-readable on multi-user machines. Caught in code review.
  > **How to apply**: Any function that writes a config file in a user's home directory: use 0o600 for files, 0o700 for the directory.

- [ ] Never silently reset structured config on parse failure — return an error instead
  > **Why**: json.Unmarshal failure followed by settings={} then os.WriteFile destroys all existing user config. Caught in code review.
  > **How to apply**: When loading JSON config for mutation: if Unmarshal fails and the file exists and is non-empty, return the error — do not silently proceed with an empty struct.

**Plan deviations:** none

---

## [spex-apply] session-identity — 2026-06-18

### Promote candidates

- [ ] macOS `ps` uses `etime=` (HH:MM:SS format), not `etimes=` (seconds, Linux only)
  > **Why**: `ps -p $PID -o etimes=` fails on macOS with "keyword not found". Needed awk parsing to convert HH:MM:SS to seconds. Discovery cost ~20 min during task 5.2.
  > **How to apply**: Any shell script needing process elapsed seconds on macOS: use `ps -p $PID -o etime= | tr -d ' ' | awk -F'[:-]' '{n=NF;s=0;if(n>=1)s+=$n;if(n>=2)s+=$(n-1)*60;if(n>=3)s+=$(n-2)*3600;if(n>=4)s+=$(n-3)*86400;print s}'`

- [ ] Get-or-create DB pattern: return `(id string, err error)` from upsert functions
  > **Why**: UpsertSession needed to return the canonical sessions.id to avoid a second SELECT. Returning the ID from the upsert is cleaner than a follow-up query.
  > **How to apply**: When a DB upsert needs the canonical PK of the affected row, include it in the return signature: `func Upsert(db, row) (id string, err error)`.

- [ ] SQLite `ON CONFLICT` for non-PK UNIQUE constraints requires explicit `UNIQUE INDEX` — without it, SELECT+UPDATE is the correct two-step pattern
  > **Why**: Tried to use INSERT OR IGNORE approach for `(process_pid, process_start)` but the columns lack a UNIQUE constraint (adding one would change schema). SELECT+INSERT-or-UPDATE is cleaner here.
  > **How to apply**: When upsert key is not the PK and adding a UNIQUE constraint is undesirable, use explicit SELECT → branch → INSERT or UPDATE.

## 2026-06-18 — ai-tool-time-tracker [spex-apply]

**Promote candidates:**

- [ ] `bufio.Scanner` is unsafe for JSONL line counting — use `io.ReadAll` + `bytes.Count`
  > **Why**: Claude transcript lines can embed large tool_result payloads exceeding Scanner's 64 KB default, silently truncating the count and storing a wrong `prompt_line_offset` that causes token double-counting.
  > **How to apply**: Any line-counting function over JSONL/transcript files: use `bytes.Count(data, []byte("\n"))` after `io.ReadAll`, not `bufio.Scanner`.

- [ ] Pass an already-open DB conn into helpers rather than calling `db.Open()` a second time
  > **Why**: Two sequential `db.Open()` calls per hook invocation; each open acquires a file lock and runs migrate(). Redundant overhead on every Stop event.
  > **How to apply**: In hook commands, open DB once in `RunE` and pass the conn down to all helpers that need it.

**Plan deviations:** Task 10.7 implemented in `cmd/tt/record.go` rather than `internal/recorder/response.go` — transcript parsing lives in the cmd layer; `RecordResponse` only accepts pre-parsed token JSON.

---

## [spex-debugging] claude-code-token-null — 2026-06-18

### Misses

- 🟡 painful: model search bounded by `lastUserIdx` → `len(all)-1` → when Stop fires after `/clear`, `lastUserIdx` is the final entry; range is empty, model returns "".

### Promote candidates

- [x] `extractFromTranscript`: model is session-scoped, not turn-scoped — search entire transcript for last assistant entry, not just current-turn range
  > **Why**: Bounded range `(lastUserIdx, end]` breaks whenever Stop fires before any new assistant entry is appended (e.g. `/clear`, rapid stop). Model doesn't change within a session, so searching the whole transcript is always correct.
  > **How to apply**: When extracting session-scoped metadata from JSONL, search the full transcript (`i >= 0`), not just the current turn window.

- [x] `extractFromTranscript`: token extraction needs fallback to previous turn window when /clear race occurs
  > **Why**: Same root cause as model-extraction bug. When Stop fires immediately after /clear, `lastUserIdx` points to the /clear user entry — primary range `[lastUserIdx+1, end)` is empty. Fallback searches `[prevUserIdx+1, lastUserIdx)` to retrieve tokens from the actual last turn.
  > **How to apply**: After primary range yields `total == 0`, find `prevUserIdx` (the user entry before `lastUserIdx`) and re-run dedup+sum on that window. Fixed in `cmd/tt/record.go:extractFromTranscript`.

### Plan deviations

- Task 6.2 (update SQL grouping) was listed as conditional work but turned out to be N/A: report SQL already uses `sessions.id` as group key, and turns now correctly reference stable ID, so no SQL change was needed.
