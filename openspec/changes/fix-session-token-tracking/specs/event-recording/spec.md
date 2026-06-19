## MODIFIED Requirements

### Requirement: 記錄 response 事件

系統 SHALL 透過 `tt record response` 子命令接收 hook 呼叫，並更新對應 turn 的 `response_at`、token 欄位（含 `cache_creation_5m_tokens`、`cache_creation_1h_tokens`、`model`）、`estimated_cost_usd`，同時將 `subagent_tokens_settled` 設為 0，表示需由 reconcile 在 process 結束後完整重算 subagent token。

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
- **THEN** 更新 `subagent_tokens_settled = 0`
- **THEN** 根據 turn 的 `model` 查詢定價表，計算並寫入 `estimated_cost_usd`

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

新增 flag：
```
tt record prompt --session <id> --project <path> --tool <tool> --model <model> --transcript-path <path>
```

- `--transcript-path`：Claude Code hook stdin payload 中的 `transcript_path` 欄位值；若未提供則 `prompt_line_offset` 寫入 NULL

#### Scenario: RecordPrompt 記錄 transcript offset（bufio 實作）

- **WHEN** `tt record prompt --session abc123 --transcript-path ~/.claude/projects/foo/abc123.jsonl` 被呼叫
- **THEN** `turns` 表新插入的 turn 寫入 `transcript_path` 以及 `prompt_line_offset = N`（N 為當下 transcript 檔案的行數，以 bufio.Scanner 計行）
- **THEN** 若 transcript 檔案不存在，`prompt_line_offset` 寫入 0，不報錯
- **THEN** 計行過程不將整個 transcript 讀入記憶體（使用串流計行）

#### Scenario: RecordResponse 以 offset 切割 transcript

- **WHEN** `tt record response --session abc123` 被呼叫
- **AND** 最新 turn 的 `prompt_line_offset = 42`, `transcript_path = "~/.claude/projects/foo/abc123.jsonl"`
- **THEN** `ExtractWindow` 只讀取 transcript 從第 42 行起的 assistant entries（第 0–41 行完全忽略）
- **THEN** dedup + sum 邏輯與現行相同，結果寫入 token 欄位

#### Scenario: prompt_line_offset 為 NULL 時 fallback 至原有邏輯

- **WHEN** 最新 turn 的 `prompt_line_offset IS NULL`（舊資料或 `--transcript-path` 未提供）
- **THEN** `ExtractLastTurn` 使用 lastUserIdx 邏輯作為 fallback，行為與原先相同

#### Scenario: transcript_path 為 NULL 時跳過 token 提取

- **WHEN** 最新 turn 的 `transcript_path IS NULL` 且 Stop hook stdin 也未提供 `transcript_path`
- **THEN** token 欄位寫入 NULL，不報錯，exit code 0

## ADDED Requirements

### Requirement: turns 表新增 DB 欄位

系統 SHALL 在 `turns` 表中新增下列欄位（透過 `addTurnColumns` 的 PRAGMA + ALTER TABLE 流程）：

| 欄位 | 型別 | 說明 |
|------|------|------|
| `model` | TEXT | 本 turn 使用的模型，由 Stop hook 或 reconcile 寫入 |
| `cache_creation_5m_tokens` | INTEGER | `cache_creation.ephemeral_5m_input_tokens` |
| `cache_creation_1h_tokens` | INTEGER | `cache_creation.ephemeral_1h_input_tokens` |
| `subagent_tokens_settled` | BOOLEAN DEFAULT 0 | 0 = subagent token 待重算；1 = 已由 reconcile 重算完成 |

所有新欄位允許 NULL（`subagent_tokens_settled` 有 DEFAULT 0），舊 turn 資料不需要 backfill。

#### Scenario: 首次啟動時自動新增欄位

- **WHEN** `tt` 首次在已存在的舊版 DB 上執行任意命令
- **THEN** `addTurnColumns` 偵測缺少的欄位並逐一 ALTER TABLE 新增，不修改現有資料
- **THEN** 命令正常執行，exit code 0

#### Scenario: 欄位已存在時不重複 ALTER

- **WHEN** DB 已包含 `model`、`cache_creation_5m_tokens` 等欄位
- **THEN** `addTurnColumns` 跳過已存在的欄位，不回傳錯誤

### Requirement: countLines 使用 bufio.Scanner

系統 SHALL 在 `recorder.countLines` 中使用 `bufio.Scanner` 逐行計數，不將整個 transcript 讀入記憶體。

#### Scenario: 大型 transcript 計行不耗盡記憶體

- **WHEN** transcript 檔案大小超過 1MB
- **THEN** `countLines` 回傳正確行數，且過程中記憶體使用量不超過 bufio 的 buffer 大小（預設 64KB）

#### Scenario: 超長單行 transcript 不回傳錯誤

- **WHEN** transcript 某行長度超過 bufio.Scanner 預設 buffer（64KB）
- **THEN** `countLines` 使用擴大 buffer（1MB）後成功計行，回傳正確結果，不 panic
