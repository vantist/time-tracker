# event-recording Specification

## Purpose
TBD - created by archiving change ai-tool-time-tracker. Update Purpose after archive.
## Requirements
### Requirement: 記錄 prompt 事件

系統 SHALL 透過 `tt record prompt` 子命令接收 hook 呼叫，並在 SQLite 中建立或更新 session 紀錄，同時寫入 turn 的 `prompt_at` 時間戳。

命令簽章：
```
tt record prompt --session <id> --project <path> --tool <tool> --model <model>
```

- `--session`：hook 提供的 session ID（字串）
- `--project`：git root 路徑，若非 git repo 則為 cwd
- `--tool`：`"claude-code"` 或 `"copilot-cli"`
- `--model`：模型名稱字串，未知時允許任意值

#### Scenario: 首次 prompt 建立 session 與 turn

- **WHEN** `tt record prompt --session abc123 --project /home/user/myproject --tool claude-code --model claude-sonnet-4-6` 被呼叫
- **THEN** `sessions` 表中 `id = "abc123"` 的 session 不存在時，建立新 session（`project = "/home/user/myproject"`, `tool = "claude-code"`, `started_at = 目前 unix ms`）
- **THEN** `turns` 表中插入一筆新 turn（`session_id = "abc123"`, `model = "claude-sonnet-4-6"`, `prompt_at = 目前 unix ms`）
- **THEN** 命令回傳 exit code 0，無輸出到 stdout

#### Scenario: 同 session 第二次 prompt 不重建 session

- **WHEN** `session_id = "abc123"` 的 session 已存在，再次呼叫 `tt record prompt --session abc123 ...`
- **THEN** `sessions` 表中 `id = "abc123"` 的 session 不被重複建立（upsert 不覆蓋 `started_at`）
- **THEN** `turns` 表插入新的 turn 紀錄（同一 session 下的第二個 turn）

#### Scenario: git branch 自動偵測並存入 session

- **WHEN** `--project` 指向的路徑是 git repo，且 `git branch --show-current` 回傳非空字串
- **THEN** `sessions.branch` 存入該 branch 名稱

#### Scenario: 非 git repo 時 branch 為 NULL

- **WHEN** `--project` 指向的路徑不是 git repo，或 `git branch --show-current` 回傳空字串
- **THEN** `sessions.branch` 寫入 NULL，不報錯

### Requirement: 記錄 response 事件

系統 SHALL 透過 `tt record response` 子命令接收 hook 呼叫，並更新對應 turn 的 `response_at`、token 欄位、`estimated_cost_usd`，同時更新 session 的 `ended_at`。

命令簽章：
```
tt record response --session <id> --tokens <json>
```

- `--session`：與 `tt record prompt` 相同的 session ID
- `--tokens`：JSON 字串，包含 token 計數（欄位名稱允許多種格式）

#### Scenario: 成功記錄 response 並計算成本

- **WHEN** `tt record response --session abc123 --tokens '{"input_tokens":1000,"output_tokens":200,"cache_read_tokens":500,"cache_creation_tokens":0}'` 被呼叫
- **THEN** 最新一筆 `session_id = "abc123"` 且 `response_at IS NULL` 的 turn，更新 `response_at = 目前 unix ms`
- **THEN** 更新 `input_tokens = 1000`, `output_tokens = 200`, `cache_read_tokens = 500`, `cache_creation_tokens = 0`
- **THEN** 根據 turn 的 `model` 查詢定價表，計算並寫入 `estimated_cost_usd`
- **THEN** 更新 `sessions.ended_at = 目前 unix ms`

#### Scenario: token JSON 欄位名稱容錯

- **WHEN** `--tokens` JSON 使用 `usage.input_tokens` 巢狀格式（`{"usage":{"input_tokens":1000,"output_tokens":200}}`）
- **THEN** 系統正確解析 token 值，功能與扁平格式相同

#### Scenario: token JSON 缺欄位時記錄 NULL

- **WHEN** `--tokens` JSON 缺少 `cache_read_tokens` 欄位
- **THEN** `turns.cache_read_tokens` 寫入 NULL，不報錯，exit code 0

#### Scenario: 找不到對應 prompt turn 時靜默跳過

- **WHEN** `--session abc123` 下找不到 `response_at IS NULL` 的 turn（可能 prompt 記錄失敗）
- **THEN** 命令不報錯，exit code 0，不修改任何資料

### Requirement: Transcript-anchored token capture

系統 SHALL 在 `tt record prompt` 時記錄當下 transcript 的行數（`prompt_line_offset`），作為後續 token 提取的 anchor，使 `tt record response` 能精確切割屬於本 turn 的 assistant entries，不受 `/clear` 或快速連續輸入影響。

**設計動機**：Stop hook 讀取 transcript 若用「找最後一個 user entry」作為 anchor，在使用者 `/clear` 後立即下指令時，lastUserIdx 會指向新的 user entry，導致 token window 為空。改用 `prompt_line_offset`（prompt 當下的行數）作為固定 anchor，無論 transcript 後續如何增長，切割範圍永遠正確。

新增 flag：
```
tt record prompt --session <id> --project <path> --tool <tool> --model <model> --transcript-path <path>
```

- `--transcript-path`：Claude Code hook stdin payload 中的 `transcript_path` 欄位值；若未提供則 `prompt_line_offset` 寫入 NULL

#### Scenario: RecordPrompt 記錄 transcript offset

- **WHEN** `tt record prompt --session abc123 --transcript-path ~/.claude/projects/foo/abc123.jsonl` 被呼叫
- **THEN** `turns` 表新插入的 turn 寫入 `transcript_path = "~/.claude/projects/foo/abc123.jsonl"` 以及 `prompt_line_offset = N`（N 為當下 transcript 檔案的行數，以 `\n` 分隔計算）
- **THEN** 若 transcript 檔案不存在，`prompt_line_offset` 寫入 0，不報錯

#### Scenario: RecordResponse 以 offset 切割 transcript

- **WHEN** `tt record response --session abc123` 被呼叫
- **AND** 最新 turn 的 `prompt_line_offset = 42`, `transcript_path = "~/.claude/projects/foo/abc123.jsonl"`
- **THEN** `extractFromTranscript` 只讀取 transcript 從第 42 行起的 assistant entries（第 0–41 行完全忽略）
- **THEN** dedup + sum 邏輯與現行相同，結果寫入 `input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_creation_tokens`

#### Scenario: prompt_line_offset 為 NULL 時 fallback 至原有邏輯

- **WHEN** 最新 turn 的 `prompt_line_offset IS NULL`（舊資料或 `--transcript-path` 未提供）
- **THEN** `extractFromTranscript` 使用原有 lastUserIdx 邏輯作為 fallback，行為與原先相同

#### Scenario: transcript_path 為 NULL 時跳過 token 提取

- **WHEN** 最新 turn 的 `transcript_path IS NULL` 且 Stop hook stdin 也未提供 `transcript_path`
- **THEN** token 欄位寫入 NULL，不報錯，exit code 0

### Requirement: Hook 呼叫失敗不中斷 AI 工具

系統 SHALL 在任何 `tt record` 呼叫失敗時（SQLite 鎖定、磁碟滿、解析錯誤），回傳 exit code 0 並將錯誤記錄到 stderr，確保 hook 失敗不影響 AI 工具繼續運作。

#### Scenario: SQLite 寫入失敗時仍回傳 exit code 0

- **WHEN** SQLite 資料庫檔案被鎖定（另一個 `tt` 程序正在寫入）
- **THEN** 命令在 stderr 輸出錯誤訊息（含錯誤描述）
- **THEN** 命令回傳 exit code 0

