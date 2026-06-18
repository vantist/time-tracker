## ADDED Requirements

### Requirement: 提取 subagent token 並合入 turn 成本

系統 SHALL 在執行 `tt record response` 時，掃描主 transcript 中 offset 之後的 `tool_use` entries，找出 `name == "Agent"` 的呼叫，透過對應的 `meta.json` 找到 subagent jsonl，合計其 token 並加入主 transcript 的 token 總計中。

#### Scenario: 無 subagents 目錄時回傳零

- **WHEN** 主 transcript 同層不存在 `<session_id>/subagents/` 目錄
- **THEN** extractSubagentTokens 回傳 `transcriptUsageFields{}` 零值，不回傳錯誤

#### Scenario: Agent tool_use 有對應 meta.json 時合計 token

- **WHEN** 主 transcript offset 之後有 `tool_use { name: "Agent", id: "toolu_xxx" }`，且 `subagents/agent-yyy.meta.json` 的 `toolUseId == "toolu_xxx"`，且 `subagents/agent-yyy.jsonl` 有 assistant entries
- **THEN** extractSubagentTokens 回傳該 subagent 的 input/output/cache token 總計

#### Scenario: 多個 subagent 時合計所有匹配的 token

- **WHEN** 主 transcript offset 之後有多個 `tool_use { name: "Agent" }`，各自有對應 meta.json 和 jsonl
- **THEN** extractSubagentTokens 回傳所有匹配 subagent 的 token 總和

#### Scenario: meta.json 的 toolUseId 不在本 turn 的 Agent 呼叫中

- **WHEN** subagents 目錄有 meta.json，但其 `toolUseId` 對應的 tool_use 在 offset 之前（前一個 turn）
- **THEN** extractSubagentTokens 不計算該 subagent 的 token

#### Scenario: subagent jsonl 不存在時略過

- **WHEN** meta.json 存在且 toolUseId 匹配，但對應的 `.jsonl` 檔案不存在
- **THEN** extractSubagentTokens 略過該 subagent，繼續處理其他 subagent

#### Scenario: subagent token 合入最終回傳值

- **WHEN** `extractFromTranscriptAtOffset` 完成主 transcript 提取，且 `extractSubagentTokens` 回傳非零值
- **THEN** 最終回傳的 tokensJSON 包含主 transcript + subagent 的合計 token
