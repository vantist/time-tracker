## MODIFIED Requirements

### Requirement: User 主動時間 Idle Threshold 近似計算

系統 SHALL 將 user 主動時間計算改為 interval-based 方式：

1. 每個 turn 產生 interval `[response_at[i-1], prompt_at[i]]`（見 `user-time` spec）
2. 過濾掉長度 ≥ idleThreshold 的 interval
3. 跨 sessions 收集所有有效 intervals，merge 重疊後加總

idle_threshold 預設值為 15 分鐘（900000 毫秒），可透過設定覆蓋。

#### Scenario: Gap 小於 idle threshold 計入 user 時間

- **WHEN** idleThreshold = 15 分鐘，session 有 2 個 turns：
  - turn 1：`response_at = T+60s`
  - turn 2：`prompt_at = T+180s`
  - interval 長度 = 120 秒 = 2 分鐘 < 15 分鐘
- **THEN** 此 interval 保留，user 主動時間累加 2 分鐘

#### Scenario: Interval 長度大於等於 idle threshold 丟棄

- **WHEN** idleThreshold = 15 分鐘，interval 長度 = 20 分鐘（1200000 ms）
- **THEN** 該 interval 丟棄，不計入 user 主動時間

#### Scenario: 多個 session 並行時 merge 去重

- **WHEN** 查詢範圍內有 session A 和 session B 時間重疊
- **THEN** 兩個 session 的 user intervals 合併去重後加總，重疊時段不重複計算

#### Scenario: Session 只有一個 turn 時 user 時間來自 session start

- **WHEN** session 只有一個 turn，`session_start` 不為零值
- **THEN** user 時間 = `turns[0].prompt_at - session_start`（若長度 < idleThreshold）

#### Scenario: Session 只有一個 turn 且 session_start 為零值

- **WHEN** session 只有一個 turn，`session_start` 為零值
- **THEN** user 主動時間 = 0
