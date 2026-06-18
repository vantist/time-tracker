## ADDED Requirements

### Requirement: 定價表內建於 binary

系統 SHALL 在 binary 中 hard-code 定價表，每個 model 包含：
- `input_price_per_1m_tokens`：USD
- `output_price_per_1m_tokens`：USD
- `cache_read_price_per_1m_tokens`：USD
- `cache_creation_price_per_1m_tokens`：USD

初始定價表 SHALL 至少包含以下 models（以 2026 年 6 月 Anthropic 定價為準）：

| model | input | output | cache_read | cache_creation |
|-------|-------|--------|------------|----------------|
| `claude-haiku-4-5-20251001` | $0.80 | $4.00 | $0.08 | $1.00 |
| `claude-sonnet-4-6` | $3.00 | $15.00 | $0.30 | $3.75 |
| `claude-opus-4-8` | $15.00 | $75.00 | $1.50 | $18.75 |

#### Scenario: 已知 model 計算 estimated_cost_usd

- **WHEN** turn 的 `model = "claude-sonnet-4-6"`，`input_tokens = 1000`, `output_tokens = 200`, `cache_read_tokens = 500`, `cache_creation_tokens = 0`
- **THEN** `estimated_cost_usd = (1000/1000000 * 3.00) + (200/1000000 * 15.00) + (500/1000000 * 0.30) + (0/1000000 * 3.75)`
- **THEN** `estimated_cost_usd ≈ 0.00300 + 0.00300 + 0.00015 + 0.00000 = 0.00615`

### Requirement: 未知 model 的 estimated_cost_usd 寫 NULL

系統 SHALL 在 `model` 欄位不在定價表中時，將 `estimated_cost_usd` 寫入 NULL，不報錯。

#### Scenario: 未知 model 不中斷記錄

- **WHEN** turn 的 `model = "gpt-5-unknown"`，不在定價表中
- **THEN** `turns.estimated_cost_usd = NULL`
- **THEN** 其餘 token 欄位（`input_tokens` 等）正常寫入
- **THEN** 命令 exit code 0

### Requirement: 報表成本加總

系統 SHALL 在報表中加總查詢範圍內所有 `estimated_cost_usd` 不為 NULL 的 turns，並顯示總預估成本。若所有 turns 的成本皆為 NULL，顯示 "N/A"。

#### Scenario: 混合已知與未知 model 的成本加總

- **WHEN** 查詢範圍內有 3 個 turns：cost = $0.006, NULL, $0.003
- **THEN** 報表顯示 Est. cost: $0.009（跳過 NULL）

#### Scenario: 全部 model 未知時顯示 N/A

- **WHEN** 查詢範圍內所有 turns 的 `estimated_cost_usd` 皆為 NULL
- **THEN** 報表顯示 `Est. cost: N/A`
