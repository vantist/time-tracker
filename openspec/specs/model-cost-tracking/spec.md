# model-cost-tracking Specification

## Purpose
TBD - created by archiving change enrich-report-and-tt-serve. Update Purpose after archive.
## Requirements
### Requirement: 從 transcript 抽取 model 並補寫至 session

`RecordResponse` SHALL 從 transcript JSONL 的 assistant entry 抽取 `message.model`，若 `sessions.model` 為空則以 UPDATE 補寫。

#### Scenario: model 從 transcript 寫入 session

- **WHEN** Stop hook 呼叫 `tt record response`，transcript JSONL 含 assistant entry 且 `sessions.model` 為空
- **THEN** `sessions.model` MUST 被更新為 transcript 中的 model 值

#### Scenario: model 已存在時不覆蓋

- **WHEN** `sessions.model` 已有值（非空字串）
- **THEN** UPDATE 不執行，既有 model 值不變

#### Scenario: transcript 無 model 欄位

- **WHEN** transcript 的 assistant entry 無 `message.model` 欄位（空字串或欄位不存在）
- **THEN** `sessions.model` 保持原值，tokens 記錄照常完成

### Requirement: pricing normalize 去除 gateway 前綴

`pricing.Calculate` SHALL 在查詢 pricing table 前對 model ID 執行 normalize：去除最後一個 `/` 之前的所有字元（gateway 前綴如 `vertex_ai/`）。

#### Scenario: vertex_ai 前綴 model 正確計算 cost

- **WHEN** model 為 `vertex_ai/claude-sonnet-4-6`
- **THEN** `pricing.Calculate` MUST 以 `claude-sonnet-4-6` 查詢 pricing table，回傳非 nil cost

#### Scenario: 無前綴 model 維持正確

- **WHEN** model 為 `claude-sonnet-4-6`（無前綴）
- **THEN** normalize 後仍為 `claude-sonnet-4-6`，pricing 查詢結果不變

#### Scenario: 未知 model 回傳 nil

- **WHEN** normalize 後的 model 不在 pricing table 中
- **THEN** `pricing.Calculate` 回傳 nil（不影響現有行為）

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

Cache write 使用 5-minute TTL 定價（1.25× base input）作為保守估算，因 DB 無從分辨 TTL。

#### Scenario: haiku-4-5 cost 計算正確

- **WHEN** model `claude-haiku-4-5`，input 1,000,000 tokens，其餘 0
- **THEN** cost = $1.00（1 MTok × $1.00）

#### Scenario: opus-4-8 使用新定價

- **WHEN** model `claude-opus-4-8`，input 1,000,000 tokens，其餘 0
- **THEN** cost = $5.00（舊定價為 $15.00，新定價 $5.00）

