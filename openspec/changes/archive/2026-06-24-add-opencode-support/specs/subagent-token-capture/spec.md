## ADDED Requirements

### Requirement: 以 --subagent-tokens flag 直接提供 subagent token

系統 SHALL 支援透過 `tt record response --subagent-tokens <json>` 直接提供 subagent token 資料（opencode event 路徑），作為 transcript 掃描提取路徑的替代方案。當 `--subagent-tokens` flag 被提供時：

- 系統 SHALL 將 flag JSON 陣列中每個元素的 `{model, input_tokens, output_tokens, cache_read_tokens?, cache_creation_tokens?, reasoning_tokens?}` 直接寫入 `turn_model_usages`，標記 `is_subagent = 1`
- 系統 SHALL NOT 對該 turn 執行 transcript subagent 掃描（`ExtractWindow` 的 `tool_use { name: "Agent" }` 掃描與 `meta.json` 查找），避免重複計算
- 系統 SHALL 依各 subagent element 的 `model` 查詢定價表計算 `estimated_cost_usd`

此需求與既有「提取 subagent token 並合入 turn 成本」需求並存：transcript 掃描適用於 Claude Code / Copilot CLI 等工具，flag 路徑適用於 opencode（event 已含 token）。

#### Scenario: 提供 --subagent-tokens 時直接寫入不掃描 transcript

- **WHEN** `tt record response --session s1 --tokens '<main>' --subagent-tokens '[{"model":"claude-haiku","agent":"build","input_tokens":100,"output_tokens":50,"cache_read_tokens":20}]'` 被呼叫
- **THEN** `turn_model_usages` 寫入一筆 `is_subagent = 1` 明細（`model = "claude-haiku"`, `input_tokens = 100`, `output_tokens = 50`, `cache_read_tokens = 20`）
- **THEN** 系統不掃描主 transcript 的 `tool_use { name: "Agent" }` entries，不查找 `meta.json`

#### Scenario: --subagent-tokens 陣列含多個 subagent 時逐一寫入

- **WHEN** `--subagent-tokens` 為 `[{"model":"claude-haiku","agent":"build","input_tokens":100,"output_tokens":50},{"model":"claude-haiku","agent":"explore","input_tokens":80,"output_tokens":30}]`
- **THEN** `turn_model_usages` 寫入兩筆 `is_subagent = 1` 明細，分別對應 `build` 與 `explore` 的 token
- **THEN** 兩筆明細的 `turn_id` 均關聯至本次更新的 turn

#### Scenario: --subagent-tokens 缺欄位時記錄 NULL

- **WHEN** `--subagent-tokens` 陣列元素缺少 `cache_read_tokens` 欄位
- **THEN** 對應的 `turn_model_usages` 紀錄其 `cache_read_tokens` 寫入 NULL，不報錯，exit code 0
