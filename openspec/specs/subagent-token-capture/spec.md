# subagent-token-capture Specification

## Purpose
TBD - created by archiving change subagent-token-capture. Update Purpose after archive.
## Requirements
### Requirement: 提取 subagent token 並合入 turn 成本

系統 SHALL 在執行 `tt record response` 或 `reconcile` 時，掃描主 transcript 中 `[from, to)` 範圍內的 `tool_use` entries，找出 `name == "Agent"` 的呼叫，透過對應的 `meta.json` 找到 subagent jsonl，提取 subagent 的 model 與 token 消耗，並以 `is_subagent = 1` 歸屬。

#### Scenario: 無 subagents 目錄時回傳零

- **WHEN** 主 transcript 同層不存在 `<session_id>/subagents/` 目錄
- **THEN** `ExtractWindow` 只回傳主 Agent 的 model usage，不包含 subagent 的 usage

#### Scenario: Agent tool_use 有對應 meta.json 時合計 token

- **WHEN** 主 transcript `[from, to)` 範圍內有 `tool_use { name: "Agent", id: "toulu_xxx" }`，且 `subagents/agent-yyy.meta.json` 的 `toolUseId == "toulu_xxx"`，且 `subagents/agent-yyy.jsonl` 有 assistant entries
- **THEN** `ExtractWindow` 回傳結果中包含一個獨立的 `ModelUsage`，標記為 `is_subagent = true`，記錄該 subagent 的 model 與 token 消耗小計

#### Scenario: 多個 subagent 時合計所有匹配的 token

- **WHEN** 主 transcript `[from, to)` 範圍內有多個 `tool_use { name: "Agent" }`，各自有對應 meta.json 和 jsonl
- **THEN** `ExtractWindow` 對相同 model 的 subagents 進行合併累加，回傳每個 model 對應的 `is_subagent = true` 總計

#### Scenario: to 邊界之後的 Agent tool_use 不被計入

- **WHEN** 主 transcript 第 `to` 行之後存在 Agent tool_use entries（屬於後續 turn）
- **THEN** 提取邏輯不收集第 `to` 行（含）之後的 Agent ID，不計算這些 subagent 的 token

#### Scenario: meta.json 的 toolUseId 不在本 turn 的 Agent 呼叫中

- **WHEN** subagents 目錄有 meta.json，但其 `toolUseId` 對應的 tool_use 在 offset 之前（前一個 turn）或在 `to` 之後
- **THEN** 提取邏輯不計算該 subagent 的 token

#### Scenario: subagent jsonl 不存在時略過

- **WHEN** meta.json 存在且 toolUseId 匹配，但對應的 `.jsonl` 檔案不存在
- **THEN** 提取邏輯略過該 subagent，繼續處理其他 subagent

#### Scenario: subagent token 合入 WindowResult

- **WHEN** `ExtractWindow` 完成主 transcript 提取，且找到匹配的 subagent 消耗
- **THEN** 最終回傳的 `WindowResult.Usages` 同時包含主 Agent (`is_subagent = false`) 與 subagent (`is_subagent = true`) 的詳細列表

### Requirement: ExtractWindow 回傳 typed struct

系統 SHALL 提供 `transcript.WindowResult` struct 與 `ModelUsage` struct 作為 `ExtractWindow` 與 `ExtractLastTurn` 的回傳型別，取代 JSON string 傳遞。

```go
type WindowResult struct {
    Usages []ModelUsage
}

type ModelUsage struct {
    Model               string
    IsSubagent          bool
    InputTokens         int
    OutputTokens        int
    CacheReadTokens     int
    CacheCreationTokens int
    CacheCreation5m     int
    CacheCreation1h     int
}
```

#### Scenario: ExtractWindow 回傳 WindowResult

- **WHEN** `transcript.ExtractWindow(path, from, to)` 被呼叫，transcript 中有 assistant entries
- **THEN** 回傳 `(WindowResult, error)`，所有欄位填入對應值，`error = nil`

#### Scenario: 空 transcript 回傳零值 WindowResult

- **WHEN** `transcript.ExtractWindow` 讀到的 window 無 any assistant entry
- **THEN** 回傳 `(WindowResult{}, nil)`（零值 struct，非 error）

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

