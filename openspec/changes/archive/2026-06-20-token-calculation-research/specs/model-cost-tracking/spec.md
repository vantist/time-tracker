## MODIFIED Requirements

### Requirement: 從 transcript 抽取 model 並補寫至 session

`RecordResponse` (或對應的 `record response` 鉤子命令) SHALL 根據所使用的 `tool` 分流解析對應的本地日誌（JSONL 格式），並提取 model 與精確的 token 數據存入資料庫：
1. **Claude Code** (`tool == "claude-code"`): 解析 `~/.claude/projects/**/*.jsonl`，從 assistant entry 抽取 `message.model`、`inputTokens` 等欄位。
2. **GitHub Copilot CLI** (`tool == "copilot-cli"`): 解析 `~/.copilot/session-state/<sessionId>/events.jsonl`，篩選 `"type":"session.shutdown"` 的 `modelMetrics`，提取該模型的 `inputTokens`、`outputTokens`、`cacheReadTokens`、`cacheWriteTokens` 等。
3. **Antigravity** (`tool == "antigravity"`): 解析 `~/.gemini/antigravity/brain/<sessionId>/.system_generated/logs/transcript.jsonl`，統計主 Agent 的 model usage。

若 `sessions.model` 為空，則以抽取出的 model 值補寫更新。

#### Scenario: model 從 transcript 寫入 session (Claude Code)

- **WHEN** Stop hook 呼叫 `tt record response`，`tool` 為 `claude-code` 且 `sessions.model` 為空
- **THEN** `sessions.model` MUST 被更新為 transcript 中的 model 值

#### Scenario: Copilot CLI 日誌解析 modelMetrics

- **WHEN** Stop hook 呼叫 `tt record response`，`tool` 為 `copilot-cli`，且 `sessionId` 為 `xyz`
- **THEN** `tt` MUST 解析 `~/.copilot/session-state/xyz/events.jsonl`，並正確提取 `session.shutdown` 事件中 `gpt-5.4` 模型的 input/output/cache token 消耗與 model 名稱

#### Scenario: Antigravity 日誌解析

- **WHEN** Stop hook 呼叫 `tt record response`，`tool` 為 `antigravity`，且 `sessionId` 為 `abc`
- **THEN** `tt` MUST 解析 `~/.gemini/antigravity/brain/abc/.system_generated/logs/transcript.jsonl`，統計主 Agent 的模型 input/output token 消耗

#### Scenario: model 已存在時不覆蓋

- **WHEN** `sessions.model` 已有值（非空字串）
- **THEN** UPDATE 不執行，既有 model 值不變

#### Scenario: transcript 無 model 欄位

- **WHEN** transcript 的 assistant entry 無 `message.model` 欄位（空字串或欄位不存在）
- **THEN** `sessions.model` 保持原值，tokens 記錄照常完成

### Requirement: pricing table 更新至最新定價

pricing table SHALL 包含以下 model 及其正確定價（USD / MTok），key 使用不含日期後綴的短 ID：

| Model key | Input | Output | Cache read (0.1×) | Cache write 5m (1.25×) |
|-----------|-------|--------|-------------------|------------------------|
| `claude-fable-5` | $10 | $50 | $1 | $12.50 |
| `claude-opus-4-8` | $5 | $25 | $0.50 | $6.25 |
| `claude-opus-4-7` | $5 | $25 | $0.50 | $6.25 |
| `claude-opus-4-6` | $5 | $25 | $0.50 | $6.25 |
| `claude-opus-4-5` | $5 | $25 | $0.50 | $6.25 |
| `claude-sonnet-4-6` | $3 | $15 | $0.30 | $3.75 |
| `claude-sonnet-4-5` | $3 | $15 | $0.30 | $3.75 |
| `claude-haiku-4-5` | $1 | $5 | $0.10 | $1.25 |
| `claude-haiku-3-5` | $0.80 | $4 | $0.08 | $1.00 |
| `gpt-5.4` | $5 | $15 | $0.50 | $6.25 |
| `gpt-5-mini` | $0.15 | $0.60 | $0.015 | $0.1875 |

Cache write 使用 5-minute TTL 定價（1.25× base input）作為保守估算，因 DB 無從分辨 TTL。

#### Scenario: haiku-4-5 cost 計算正確

- **WHEN** model `claude-haiku-4-5`，input 1,000,000 tokens，其餘 0
- **THEN** cost = $1.00（1 MTok × $1.00）

#### Scenario: opus-4-8 使用新定價

- **WHEN** model `claude-opus-4-8`，input 1,000,000 tokens，其餘 0
- **THEN** cost = $5.00（舊定價為 $15.00，新定價 $5.00）

#### Scenario: gpt-5.4 cost 計算正確

- **WHEN** model `gpt-5.4`，input 1,000,000 tokens，其餘 0
- **THEN** cost = $5.00（1 MTok × $5.00）

#### Scenario: gpt-5-mini cost 計算正確

- **WHEN** model `gpt-5-mini`，input 1,000,000 tokens，其餘 0
- **THEN** cost = $0.15（1 MTok × $0.15）
