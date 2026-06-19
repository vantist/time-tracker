## MODIFIED Requirements

### Requirement: 提取 subagent token 並合入 turn 成本

系統 SHALL 在執行 `tt record response` 時，掃描主 transcript 中 `[from, to)` 範圍內的 `tool_use` entries，找出 `name == "Agent"` 的呼叫，透過對應的 `meta.json` 找到 subagent jsonl，合計其 token 並加入主 transcript 的 token 總計中。

`to` 參數代表本 turn 的上界行數（即下一個 turn 的 `prompt_line_offset`）；`to = -1` 表示讀到 EOF。掃描範圍僅限 `[from, to)`，不得超出本 turn 邊界。

#### Scenario: 無 subagents 目錄時回傳零

- **WHEN** 主 transcript 同層不存在 `<session_id>/subagents/` 目錄
- **THEN** `extractSubagentTokens` 回傳零值 `WindowResult{}`，不回傳錯誤

#### Scenario: Agent tool_use 有對應 meta.json 時合計 token

- **WHEN** 主 transcript `[from, to)` 範圍內有 `tool_use { name: "Agent", id: "toulu_xxx" }`，且 `subagents/agent-yyy.meta.json` 的 `toolUseId == "toulu_xxx"`，且 `subagents/agent-yyy.jsonl` 有 assistant entries
- **THEN** `extractSubagentTokens` 回傳該 subagent 的 input/output/cache token 總計

#### Scenario: 多個 subagent 時合計所有匹配的 token

- **WHEN** 主 transcript `[from, to)` 範圍內有多個 `tool_use { name: "Agent" }`，各自有對應 meta.json 和 jsonl
- **THEN** `extractSubagentTokens` 回傳所有匹配 subagent 的 token 總和

#### Scenario: to 邊界之後的 Agent tool_use 不被計入

- **WHEN** 主 transcript 第 `to` 行之後存在 Agent tool_use entries（屬於後續 turn）
- **THEN** `extractSubagentTokens` 不收集第 `to` 行（含）之後的 Agent ID，不計算這些 subagent 的 token

#### Scenario: meta.json 的 toolUseId 不在本 turn 的 Agent 呼叫中

- **WHEN** subagents 目錄有 meta.json，但其 `toolUseId` 對應的 tool_use 在 offset 之前（前一個 turn）或在 `to` 之後
- **THEN** `extractSubagentTokens` 不計算該 subagent 的 token

#### Scenario: subagent jsonl 不存在時略過

- **WHEN** meta.json 存在且 toolUseId 匹配，但對應的 `.jsonl` 檔案不存在
- **THEN** `extractSubagentTokens` 略過該 subagent，繼續處理其他 subagent

#### Scenario: subagent token 合入 WindowResult

- **WHEN** `ExtractWindow` 完成主 transcript 提取，且 `extractSubagentTokens` 回傳非零值
- **THEN** 最終回傳的 `WindowResult` 包含主 transcript + subagent 的合計 token

## ADDED Requirements

### Requirement: Subagent token 結算旗標

系統 SHALL 在 `turns` 表中維護 `subagent_tokens_settled BOOL DEFAULT 0` 欄位，追蹤每個 turn 的 subagent token 是否已在 process 結束後完整重算。

- Stop hook 寫入 token 時，`subagent_tokens_settled` 設為 0（待重算）
- Reconcile 在 process 結束後重算 token（含 subagent）成功後，將 `subagent_tokens_settled` 設為 1

#### Scenario: Stop hook 寫 token 時設旗標為 0

- **WHEN** `RecordResponse` 更新 turn 的 token 欄位
- **THEN** `turns.subagent_tokens_settled` 設為 0

#### Scenario: Reconcile 重算後設旗標為 1

- **WHEN** `MaybeReconcile` 對某個 turn 完成 `ExtractWindow` 呼叫並成功寫回 DB
- **THEN** `turns.subagent_tokens_settled` 設為 1

#### Scenario: subagent_tokens_settled=0 的 turn 被 reconcile 重算

- **WHEN** `reconcile` 掃描 turn，turn 的 `response_at IS NOT NULL` 且 `input_tokens IS NOT NULL` 但 `subagent_tokens_settled = 0`，且對應 process 已不存活
- **THEN** `reconcile` 對該 turn 執行 `ExtractWindow` 並覆蓋 token 欄位

### Requirement: ExtractWindow 回傳 typed struct

系統 SHALL 提供 `transcript.WindowResult` struct 作為 `ExtractWindow` 與 `ExtractLastTurn` 的回傳型別，取代 JSON string 傳遞。

```go
type WindowResult struct {
    InputTokens         int
    OutputTokens        int
    CacheReadTokens     int
    CacheCreationTokens int
    CacheCreate5m       int
    CacheCreate1h       int
    Model               string
}
```

#### Scenario: ExtractWindow 回傳 WindowResult

- **WHEN** `transcript.ExtractWindow(path, from, to)` 被呼叫，transcript 中有 assistant entries
- **THEN** 回傳 `(WindowResult, error)`，所有欄位填入對應值，`error = nil`

#### Scenario: 空 transcript 回傳零值 WindowResult

- **WHEN** `transcript.ExtractWindow` 讀到的 window 無任何 assistant entry
- **THEN** 回傳 `(WindowResult{}, nil)`（零值 struct，非 error）

### Requirement: cache_creation 5m/1h 細分欄位

系統 SHALL 從 transcript `usage.cache_creation.ephemeral_5m_input_tokens` 與 `ephemeral_1h_input_tokens` 欄位提取數值，並分別存入 `turns.cache_creation_5m_tokens` 與 `turns.cache_creation_1h_tokens`。

#### Scenario: transcript 含 cache_creation 細分欄位時正確提取

- **WHEN** transcript entry 的 `usage.cache_creation.ephemeral_5m_input_tokens = 1000`，`ephemeral_1h_input_tokens = 500`
- **THEN** `WindowResult.CacheCreate5m = 1000`，`WindowResult.CacheCreate1h = 500`

#### Scenario: transcript 無 cache_creation 欄位時填零

- **WHEN** transcript entry 的 `usage` 無 `cache_creation` 子物件（舊格式）
- **THEN** `WindowResult.CacheCreate5m = 0`，`WindowResult.CacheCreate1h = 0`，不報錯
