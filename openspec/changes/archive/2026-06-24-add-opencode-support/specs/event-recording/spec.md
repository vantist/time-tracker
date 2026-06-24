## MODIFIED Requirements

### Requirement: 記錄 prompt 事件

系統 SHALL 透過 `tt record prompt` 子命令接收 hook 呼叫，並在 SQLite 中建立或更新 session 紀錄，同時寫入 turn 的 `prompt_at` 時間戳。

命令簽章：
```
tt record prompt --session <id> --project <path> --tool <tool> --model <model>
```

- `--session`：hook 提供的 session ID（字串）
- `--project`：git root 路徑，若非 git repo 則為 cwd
- `--tool`：`"claude-code"`、`"copilot-cli"`、`"antigravity"` 或 `"opencode"`
- `--model`：模型名稱字串，未知時允許任意值

當 `--tool` 為 `"opencode"` 時，系統 SHALL NOT 要求 `--transcript-path`，且 `prompt_line_offset` 寫入 NULL（opencode token 不來自 transcript）。

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

#### Scenario: Antigravity 多步驟 prompt 去重

- **WHEN** 呼叫 `tt record prompt --session abc123 --tool antigravity --model gemini-3.5-flash`，且同 session 已存在 `response_at` 為 NULL 的 active turn
- **THEN** `turns` 表不插入新紀錄
- **THEN** 命令回傳 exit code 0，不重複插入

#### Scenario: opencode prompt 不需 transcript-path

- **WHEN** `tt record prompt --session s1 --project /repo --tool opencode --model ""` 被呼叫，且未帶 `--transcript-path`
- **THEN** `turns` 表插入一筆新 turn（`session_id = "s1"`, `tool = "opencode"`, `prompt_line_offset = NULL`）
- **THEN** 命令不報錯，exit code 0

### Requirement: 記錄 response 事件

系統 SHALL 透過 `tt record response` 子命令接收 hook 呼叫，並更新對應 turn 的 `response_at`、token 欄位（含 `cache_creation_5m_tokens`、`cache_creation_1h_tokens`、`model`）、`estimated_cost_usd`，同時將 `subagent_tokens_settled` 設為 0。此外，系統 MUST 將該 turn 當下的 model token 消耗以 `is_subagent = 0` (主 Agent) 寫入 `turn_model_usages` 關聯表。

命令簽章：
```
tt record response --session <id> --tokens <json> [--tool <tool>] [--subagent-tokens <json>]
```

- `--session`：與 `tt record prompt` 相同的 session ID
- `--tokens`：JSON 字串，包含 token 計數（欄位名稱允許多種格式）
- `--tool`：可選，AI 工具識別碼（stdin JSON 優先；flag 保留供 opencode plugin 與測試使用）
- `--subagent-tokens`：可選 JSON 字串陣列，每個元素為 `{model, agent, input_tokens, output_tokens, cache_read_tokens?, cache_creation_tokens?, reasoning_tokens?}`；提供時系統 SHALL 將每個元素以 `is_subagent = 1` 寫入 `turn_model_usages`

當 `--tool` 為 `"opencode"` 時（透過 stdin JSON 或 CLI flag 傳入），系統 SHALL 直接採用 `--tokens` 與 `--subagent-tokens` flag 值，SHALL NOT 回退至 transcript parser（`ExtractWindow`）提取 token。

#### Scenario: 成功記錄 response 並計算成本

- **WHEN** `tt record response --session abc123 --tokens '{"input_tokens":1000,"output_tokens":200,"cache_read_tokens":500,"cache_creation_tokens":0}'` 被呼叫
- **THEN** 最新一筆 `session_id = "abc123"` 且 `response_at IS NULL` 的 turn，更新 `response_at = 目前 unix ms`
- **THEN** 更新 `input_tokens = 1000`, `output_tokens = 200`, `cache_read_tokens = 500`, `cache_creation_tokens = 0`
- **THEN** 更新 `subagent_tokens_settled = 0`
- **THEN** 根據 turn 的 `model` 查詢定價表，計算並寫入 `estimated_cost_usd`
- **THEN** 在 `turn_model_usages` 表中寫入一筆明細（`turn_id` 為本 turn ID, `model` 為該 turn model, `is_subagent = 0`，以及對應的 token 與預估費用）

#### Scenario: token JSON 欄位名稱容錯

- **WHEN** `--tokens` JSON 使用 `usage.input_tokens` 巢狀格式（`{"usage":{"input_tokens":1000,"output_tokens":200}}`）
- **THEN** 系統正確解析 token 值，功能與扁平格式相同

#### Scenario: token JSON 缺欄位時記錄 NULL

- **WHEN** `--tokens` JSON 缺少 `cache_read_tokens` 欄位
- **THEN** `turns.cache_read_tokens` 寫入 NULL，不報錯，exit code 0

#### Scenario: 找不到對應 prompt turn 時靜默跳過

- **WHEN** `--session abc123` 下找不到 `response_at IS NULL` 的 turn（可能 prompt 記錄失敗）
- **THEN** 命令不報錯，exit code 0，不修改 any 資料

#### Scenario: opencode response 直接採用 flag token 不回退 transcript parser

- **WHEN** `tt record response --tool opencode --session s1 --model claude-sonnet-4-6 --tokens '{"input_tokens":500,"output_tokens":120}'` 被呼叫
- **THEN** 系統以 `--tokens` flag 值寫入 turn token 欄位
- **THEN** 系統不呼叫 `ExtractWindow` / `ExtractLastTurn`，不讀取任何 transcript 檔案

#### Scenario: --subagent-tokens 寫入 turn_model_usages 為 is_subagent = 1

- **WHEN** `tt record response --session s1 --tokens '<main json>' --subagent-tokens '[{"model":"claude-haiku","agent":"build","input_tokens":100,"output_tokens":50}]'` 被呼叫
- **THEN** `turn_model_usages` 寫入一筆 `is_subagent = 1` 明細（`model = "claude-haiku"`, `input_tokens = 100`, `output_tokens = 50`，並依該 model 定價計算費用）
- **THEN** 該明細的 `turn_id` 關聯至本次更新的 turn

#### Scenario: --subagent-tokens 未提供時不寫入 subagent 明細

- **WHEN** `tt record response --session s1 --tokens '<main json>'` 被呼叫且未帶 `--subagent-tokens`
- **THEN** `turn_model_usages` 僅寫入主 agent 一筆 `is_subagent = 0` 明細，不寫入任何 `is_subagent = 1` 紀錄
