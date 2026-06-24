# Spex Insights

## 2026-06-22 — integration-testing-unpushed-features [spex-apply]

**Promote candidates:**
- [ ] Override `PROCESS_PID` and `PROCESS_START` env vars in E2E tests to isolate process keys
  > **Why**: By default, CLI hooks deduce process identity based on the parent process PID and start time. In multi-step integration tests, multiple command invocations share the same test runner process PID, causing separate tests to collate into the same session.
  > **How to apply**: In E2E tests executing CLI commands, override process identity by explicitly setting different `PROCESS_PID` and `PROCESS_START` values in the environment before running each tool.
- [ ] Write mock transcript files in two steps (before prompt, and append response before response command) to E2E test offset parsing
  > **Why**: When recording a prompt, the hook records the line offset in the transcript. If the response exists in the file before the prompt command is run, the offset will capture it as historical, causing subsequent response commands to skip parsing.
  > **How to apply**: When E2E testing transcript offset parsing, always write the prompt-only transcript before running `record prompt`, then append LLM responses before calling `record response`.

**Plan deviations:** none

---

## 2026-06-21 — repair-session-branch [spex-apply]

**Promote candidates:**
- [ ] Reuse existing scan/query structs for updates when they map to the same columns
  > **Why**: Defining separate structs for SELECT scans vs UPDATE variables leads to redundant type definitions and field mapping boilerplate when they share the exact same fields.
  > **How to apply**: If the update target columns align with the scanned/queried columns, reuse the scan struct (e.g. `sessInfo`) directly in the update logic instead of introducing a dedicated `updateInfo` struct.
- [ ] Helper for mock git repository initialization in tests
  > **Why**: Mocking git branch or commit history in unit tests requires running multiple commands (`git init`, `git config`, `git checkout`, `git commit`). Duplicating this sequence across multiple test cases increases boilerplate.
  > **How to apply**: Create a helper like `initGitRepo(t, dir, branch)` to automate repo initialization, basic config, and an initial commit to bind the branch name cleanly.

**Plan deviations:** none

---

## 2026-06-21 — antigravity-session-recovery [spex-apply]

**Promote candidates:**

- [ ] timezone-aware datetime calculations directly in SQLite
  > **Why**: Comparing time differences inside Go loops requires scanning rows, fetching timestamps, and parsing them in Go. Offloading this directly to SQLite using datetime functions (e.g. strftime) makes code much simpler and faster.
  > **How to apply**: For time delta comparisons on database records, implement timezone-aware datetime calculations directly inside the query where possible.

- [ ] pointer-to-pointer scanning for NULL columns in database/sql
  > **Why**: Using wrapper structs like sql.NullString requires checking validity flags. Scanning NULLable columns into pointer-to-pointer variables (e.g. **time.Time or **string) automatically handles NULL states cleanly without boilerplate.
  > **How to apply**: Pass pointer variables matching **T to Scan() for NULLable columns.

**Plan deviations**: none

---

## 2026-06-21 — antigravity-session-fix [spex-apply]

**Promote candidates:**

- [ ] Database transaction updates on related tables
  > **Why**: When reconciling turn-level events, updating session-level metadata (such as backfilling sessions.model) inside the same database transaction tx ensures atomic updates, avoids race conditions, and maintains data consistency.
  > **How to apply**: Whenever reconciling or updating records in a table where related parent/child tables share fields that should align, bundle all updates within a single tx.Begin() / tx.Commit() block.

**Plan deviations:** none

---

## 2026-06-21 — antigravity-turns-fix [spex-apply]

**Promote candidates:**

- [ ] Safe JSON file loading using `os.ReadFile` + `json.Unmarshal` in CLI/library helpers
  > **Why**: Using `os.Open` and `json.NewDecoder` requires verbose error handling and manual `Close()`. For small configuration files (like settings.json), `os.ReadFile` and `json.Unmarshal` are much simpler and flatter.
  > **How to apply**: For reading configuration or metadata files under 10MB, use `os.ReadFile` and `json.Unmarshal` directly.

- [ ] Home-directory-sensitive file path probing in CLI providers
  > **Why**: Storing paths statically can break when tools migrate directories (e.g. `~/.gemini/antigravity` vs `~/.gemini/antigravity-cli`). Probing for file existence in `ResolvePath` prevents users from having to manually configure path overrides.
  > **How to apply**: Implement fallback probing using `os.Stat(expandHome(...))` inside provider paths to detect migrated paths automatically.

**Plan deviations:** none

---

## [spex-apply] tool-log-provider — 2026-06-20

### Promote candidates

- [ ] Normalize camelCase keys to snake_case at unmarshal boundary
  > **Why**: Different CLI hook payload schemas have different field case conventions (`transcript_path` vs `transcriptPath`). Normalizing camelCase keys directly in `readStdinJSON` prevents duplicate normalization logic across multiple commands.
  > **How to apply**: When defining hook payloads for new CLI tools, add camelCase json tags alongside snake_case in `hookPayload` struct, and normalize them in `readStdinJSON` immediately after unmarshaling.

**Plan deviations:** none

---

## [spex-apply] fix-session-token-tracking — 2026-06-19

### Promote candidates

- [ ] subagent usageFields aggregation — 所有欄位必須傳遞（含 `Ephemeral5m/1h`）
  > **Why**: `extractSubagentTokens` 中未將 `sumSubagentWindow` 回傳值的 `CacheCreation.Ephemeral5m/1h` 累加至 acc，導致 subagent 快取 token 完全遺失。只複製 4 個基本欄位、漏掉新欄位的模式。
  > **How to apply**: 新增欄位至 usageFields 時，務必 grep 確認所有 aggregation 呼叫點（sumWindow caller、extractSubagentTokens caller）都有累加該欄位。

- [ ] fallback window 範圍提取為變數 — `winFrom, winTo` 模式
  > **Why**: `/clear` fallback 時 `acc` 已改用 `prevUserIdx+1..lastUserIdx` 範圍重算，但 `extractSubagentTokens` 仍使用原始 `lastUserIdx+1..len(all)`——導致在空窗口內尋找 subagent，token 遺失。
  > **How to apply**: 將 primary/fallback 窗口範圍提取為 `winFrom, winTo` 變數，`sumWindow` 與 `extractSubagentTokens` 都使用同一組變數呼叫。

- [ ] JSON→DB 序列化邊界新欄位檢查清單
  > **Why**: `marshalWindowResult`、`tokenPayload` struct、UPDATE SQL 三處各需新增欄位，缺一則靜默遺失。直到 code review 才發現。
  > **How to apply**: Stop hook 的 token 流：`WindowResult` → `marshalWindowResult` (map) → `tokenPayload` (JSON) → `conn.Exec UPDATE SQL`。新增欄位時四個步驟都要確認。

**Plan deviations:** none

---

## [spex-apply] fix-user-time-semantics — 2026-06-19

### Promote candidates

- [ ] `d > 0` guard in interval keep closure — 任何 interval-based 計算都應守衛非正值
  > **Why**: 當 sessionStart > turns[0].PromptAt（時鐘偏差或資料異常）時產生負 duration，不守衛會讓 user time 縮水。
  > **How to apply**: 每次建立 Interval 後計算 duration 前先檢查 d > 0；interval 過濾條件應同時守衛正值與 idle threshold。

- [ ] Dead parameter check after function signature refactor
  > **Why**: `groupByWorkItem` 改用 `sessUserIntervals` 後，`idleThreshold` 參數未一起清理，code review 才發現。編譯器不報錯，呼叫方 misleading。
  > **How to apply**: 改函數簽章時立即 grep 函數體確認所有參數都被使用；把計算移至外層後舊 threshold/config 參數最容易成為殭屍。

**Plan deviations:** none

---

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

- [x] `bufio.Scanner` for JSONL requires explicit 1MB buffer — `sc.Buffer(make([]byte, 64*1024), 1024*1024)`
  > **Why**: Default 64KB Scanner token limit silently stops on large lines (image tool results, large tool outputs). With 1MB buffer cap it handles real transcripts. `io.ReadAll` loads entire file into memory which is worse for large sessions.
  > **How to apply**: `bufio.NewScanner` + `sc.Buffer(make([]byte, 64*1024), 1024*1024)` for any JSONL line counting. The 1MB cap matches Claude Code's practical max line size.

- [ ] Pass an already-open DB conn into helpers rather than calling `db.Open()` a second time
  > **Why**: Two sequential `db.Open()` calls per hook invocation; each open acquires a file lock and runs migrate(). Redundant overhead on every Stop event.
  > **How to apply**: In hook commands, open DB once in `RunE` and pass the conn down to all helpers that need it.

**Plan deviations:** Task 10.7 implemented in `cmd/tt/record.go` rather than `internal/recorder/response.go` — transcript parsing lives in the cmd layer; `RecordResponse` only accepts pre-parsed token JSON.

---

## 2026-06-19 — align-report-serve [spex-apply]

**Promote candidates:**
- [ ] addCost pointer-to-pointer float64 summation helper
  > **Why**: Simple helper encapsulates DRY null-checking and value allocation logic when aggregating optional cost metrics.
  > **How to apply**: Elevate to a general utility module (like pricing or pricing/helpers) if other reporting or logging modules perform cost sums on pointers.
- [ ] Avoid JS template literals backticks inside Go raw string literals
  > **Why**: Go's raw string literal delimiter is also the backtick (`). If JavaScript code inside `const HTML = `...`` uses backticks, it terminates the Go string early, breaking compilation.
  > **How to apply**: Always use ES5-style string concatenation (e.g., `'h ' + h`) or convert Go's multi-line string to double-quoted escaped strings if JS template literals are required.

**Plan deviations:** none

---

## 2026-06-19 — agent-attribution-report-serve [spex-apply]

**Promote candidates:**
- [ ] Early normalization in data loading layer
  > **Why**: Storing raw data in intermediary variables and normalizing them at multiple endpoints is error-prone. Normalizing as soon as database fields are scanned ensures consistency across CLI text formatting, JSON APIs, and HTML web dashboard rendering.
  > **How to apply**: When implementing report aggregations of columns that require normalization, run normalization function inside the `rows.Next()` scanning loop.

- [ ] Avoid JS template literals backticks inside Go raw string literals
  > **Why**: Go's raw string literal delimiter is also the backtick (`). If JavaScript code inside `const HTML = `...`` uses backticks, it terminates the Go string early, breaking compilation.
  > **How to apply**: Always use ES5-style string concatenation (e.g., `'h ' + h`) or convert Go's multi-line string to double-quoted escaped strings if JS template literals are required.

**Plan deviations:** none

---

## 2026-06-20 — subagent-model-tracking [spex-apply]

**Promote candidates:**
- [ ] Consolidated model usage mapping helper `makeMainUsage`
  > **Why**: Reusable mapping of transcript aggregator fields to `ModelUsage` encapsulates mapping logic, preventing duplicate struct assignments across multiple extraction entry points (e.g. `ExtractWindow` and `ExtractLastTurn`).
  > **How to apply**: When extracting fields from raw source maps into reporting structs, utilize mapper/builder functions to keep instantiation DRY.
- [ ] Atomic SQLite turn usage detail transactions
  > **Why**: Deleting old turn detail usages and inserting new detail values must happen atomically alongside updating the parent `turns` record. Failing to do so in a single transaction can lead to mismatched states on partial failure.
  > **How to apply**: Always wrap turn reconciliations and event recordings in explicit SQLite transaction blocks (`tx.Begin()` / `tx.Commit()`) with deferred `Rollback()` calls.

**Plan deviations:** none

---

## 2026-06-20 — token-calculation-research [spex-apply]

**Promote candidates:**
- [ ] Deduplicate Home Directory Expansion in CLI commands
  > **Why**: When referencing relative home directory paths (like `~/.copilot/...` or `~/.gemini/...`) in Go's file operations, tilde expansion does not happen automatically. Having a shared helper `expandHome` in `extract.go` avoids duplicating home-directory resolution logic across multiple log parsers.
  > **How to apply**: Ensure any tilde-prefixed path is wrapped in `expandHome` before calling `os.Open` or similar OS file calls.
- [ ] pricing test assertCost helper
  > **Why**: The pricing test had repeated `if got == nil { t.Fatal... }` check blocks. Extracting this to `assertCost(t, got, want)` helper makes tests cleaner and easier to read.
  > **How to apply**: When writing table-driven or repeated assertions, extract common assertion sequences to clean helper functions.

**Plan deviations:** none

---

## 2026-06-20 — models-expansion-robust-suffix-normalization [spex-apply]

**Promote candidates:**
- [ ] Consolidate related unit tests into table-driven tests
  > **Why**: When expanding pricing tables or adding test cases for new models, writing individual functions for each test case leads to huge amounts of boilerplate and duplicate assertions.
  > **How to apply**: Group related function-level behavior (such as `Calculate`) into struct-based table-driven tests (`tests := []struct{...}`) to make test expansion declarative and clean.
- [ ] Combine arithmetic operations to reduce floating-point divisions
  > **Why**: Evaluating terms like `float64(tokens)/1e6 * rate` multiple times can lead to compounding floating-point precision issues and unnecessary division instructions.
  > **How to apply**: Sum up the weighted token counts first, and then perform a single division by `1e6` at the end of the cost calculation function.

**Plan deviations:** none

---

## 2026-06-20 — setup-expansion [spex-apply]

**Promote candidates:**
- [ ] Resetting CLI flags in Cobra integration tests
  > **Why**: When running multiple Cobra tests in the same process, CLI flag values can persist across test executions because the global command variables are reused. If not explicitly reset, flags set in one test can bleed into subsequent tests.
  > **How to apply**: When writing Cobra integration tests, always explicitly reset all command flags by calling `cmd.Flags().Set("flag-name", "default-value")` at the beginning of each test case.

- [ ] Re-assign derived home-relative paths when changing `HOME` env var in tests
  > **Why**: Re-setting the `HOME` environment variable via `t.Setenv` mid-test is effective, but any path variable derived before that re-set (e.g. `configPath`) will still point to the old directory, causing tests to write to the wrong temp folder.
  > **How to apply**: Always re-calculate home-relative paths (like `filepath.Join(home, ...)`) immediately after re-setting `HOME` or updating a `home` directory mock variable.

**Plan deviations:** none

---

## 2026-06-20 — copilot-setup [spex-apply]

**Promote candidates:**

- [ ] Consolidate hook-merging filtering and appending loops into a shared helper `mergeHookEntries`
  > **Why**: Setting up different AI tools (Claude, Antigravity, Codex, Copilot) involves the same pattern: filtering out existing entries belonging to `_owner == "tt"` and appending new entries to the remaining list. Keeping this logic separate leads to boilerplate code and potential inconsistencies.
  > **How to apply**: When implementing hook setups for new AI tools, pass the existing entries slice and new entries slice to `mergeHookEntries` to safely filter and merge.

**Plan deviations:** none

---

## 2026-06-20 — setup-cmd-improvements [spex-apply]

**Promote candidates:**

- [ ] Struct-driven declarative flag and behavior definition for CLI commands
  > **Why**: Hardcoding nested if-else blocks for each flag value makes CLI commands scale poorly and duplicate registration/dispatch boilerplate. Encapsulating flag names, descriptions, detectors, and installers in a list of configurations simplifies command registration and RunE methods.
  > **How to apply**: When a command manages multiple tool setups or modules, use a struct-driven metadata slice to dynamically register and dispatch behavior in loops.

- [ ] Generic hook setup/updater helpers
  > **Why**: AI tool setups repeat similar operations (obtaining user home, reading JSON files, validating structures, calling merge functions, and writing with secure permissions). Extracting common setup flow to helpers like `setupToolHooks` and `updateSection` avoids duplicate operations and code drift.
  > **How to apply**: Use unified path-building and json-updating helper functions inside internal/setup rather than writing separate home directory resolution and file merge logic for each tool.

**Plan deviations:** none

---

## 2026-06-21 — detailed-token-cost-categories-breakdown [spex-apply]

**Promote candidates:**
- [ ] Centralize token category string formatting using a shared helper `formatTokens`
  > **Why**: Tables and row logs in reports all format tokens using the same `input / output / cache read / cache create` pattern. Extracting this to a single helper function avoids duplicate string building logic and ensures consistency.
  > **How to apply**: Use `formatTokens(in, out, read, create)` whenever displaying multi-category token details in text reports or logs.

- [ ] Write exported files with secure `0o600` file permissions
  > **Why**: Report outputs may contain proprietary project structures, agent work times, or cost/pricing data. Restricting write permissions using `0o600` prevents other local users on shared environments from accessing these reports.
  > **How to apply**: When implementing any report export or dump command that writes to a user-specified path, use `os.WriteFile(path, data, 0600)`.

**Plan deviations:** none

---

## [spex-debugging] session-path-shift — 2026-06-22

### Misses
- 🔴 blocking: Using `json.Decoder` in a `dec.More()` loop without checking or handling decoder error propagation caused an infinite loop during path resolution of JSONL logs.

### Promote candidates
- [ ] Use `bufio.Scanner` + `json.Unmarshal` instead of `json.Decoder` for JSONL parsing
  > **Why**: `json.Decoder` gets stuck in a permanent error state when encountering trailing newlines or corrupt lines in JSONL, causing `dec.More()` to loop infinitely. `bufio.Scanner` safely advances to the next line.
  > **How to apply**: Always read newline-delimited JSON files line-by-line using `bufio.NewScanner` and call `json.Unmarshal` on each line.

---

## [spex-debugging] antigravity-token-zero — 2026-06-22

### Promote candidates
- [ ] Extract tool-specific binary/protobuf metadata from secondary local SQLite database files
  > **Why**: Google Antigravity keeps detailed model usage in an SQLite database rather than the standard JSONL transcript file. Resolving and parsing these database files is necessary to keep token/pricing calculations correct.
  > **How to apply**: Map standard session logs to the tool's database files (e.g., `conversations/<session_id>.db`) and parse the internal binary formats (e.g., protobuf blobs in `gen_metadata` table).

---

## [spex-debugging] antigravity-cost-na — 2026-06-22

### Promote candidates
- [ ] Normalize pricing lookups across minor model versions (e.g. dot vs. hyphen naming differences)
  > **Why**: Settings files of different tools use dots (Claude Opus 4.6), while the internal pricing table uses hyphens (claude-opus-4-6). Fallback lookup prevents cost estimation failure on minor formatting inconsistencies.
  > **How to apply**: Implement fallback lookup logic in pricing tools that replaces dots with hyphens if direct map lookup fails.

---

## [spex-debugging] workitem-project-not-git-root — 2026-06-24

### Promote candidates
- [ ] Recorder should store git-root-resolved project path alongside raw cwd
  > **Why**: `workitem` keys work-item labels by git root (`ResolveProject`), but `recorder` stores the raw cwd as `sessions.project`. The report now shells out to `git rev-parse` per unique path at read time to normalize — this compensates for historical rows but is avoidable cost for future rows. Documenting this invariant prevents future "why does the report show subfolder names" regressions.
  > **How to apply**: Add a `project_root` column to `sessions` populated via `workitem.ResolveProject(input.Project)` at record time; report reads `project_root` directly and only falls back to read-time resolution for pre-migration rows. Normalize in `recorder.RecordPrompt` (recorder.go:34) so the invariant holds at the source.

---

## 2026-06-24 — copilot-cli-token-fix [spex-apply]

**Promote candidates:**
- [ ] Extract a `resolveTranscriptPathIfExists(tool, sessionID) (path string, found bool)` helper for the ResolvePath + os.Stat + silent-skip pattern
  > **Why**: The pattern "derive transcript path via provider.ResolvePath(sessionID, ""), stat it, silent-skip on missing" now appears in 3 sites (reconcileTurn defensive guard, reconcileCopilotSession, repairSessions Copilot branch). Each duplicates `expandHomePath(p.ResolvePath(...))` + `os.Stat` + the missing-file return.
  > **How to apply**: Add a helper in reconcile.go returning `(path, found)`. Use it in all 3 sites. Removes ~6 lines per site and ensures consistent silent-skip semantics for any future provider with sessionID-derivable paths.
- [ ] Document the "session-cumulative vs per-turn-delta" token semantics distinction in transcript provider conventions
  > **Why**: Copilot CLI `modelMetrics` is a session-cumulative snapshot (each session.shutdown contains the running total, not a per-event delta). ExtractWindow must reset `modelUsages` on each shutdown so only the latest cumulative value remains; summing would double-count across shutdowns. Claude Code / Antigravity tokens are per-turn deltas and must keep accumulating. This distinction is provider-specific and easy to get wrong when adding a new transcript provider.
  > **How to apply**: Add a note to the transcript package doc or provider.go stating whether each provider's metrics are cumulative-snapshot (reset-on-event) or per-event-delta (accumulate), and that ExtractWindow must handle accordingly. Future providers should declare their semantics at registration.

**Plan deviations:**
- Task 4.1 said assert `turns.model` but `repairSessions` only updates `sessions.model`; test asserts `sessions.model` (matches spec scenario "Copilot session 用 CopilotProvider 解 model" which targets resolveModel → sessions.model). `turns.model` is set separately by reconcileCopilotSession.
- Task 1.6 said modify `vscode_copilot.go` only; also modified `vscode_copilot_debug.go` because the spec scenario "mainModel empty falls back to currentModel" lives under the debug-logs requirement and that file has its own shutdown parser with the same `mainModel` bug. Broader fix, spec-aligned.

---






