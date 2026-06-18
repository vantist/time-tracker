## ADDED Requirements

### Requirement: Agent 時間精確計算

系統 SHALL 將 agent 時間定義為每個 turn 的 `response_at - prompt_at`（毫秒），並以秒或分鐘顯示。只有 `response_at` 不為 NULL 的 turn 才計入。

#### Scenario: 單 session 的 agent 時間加總

- **WHEN** session `abc123` 有 3 個 turns：
  - turn 1：`prompt_at = 0`, `response_at = 30000`（30 秒）
  - turn 2：`prompt_at = 60000`, `response_at = 75000`（15 秒）
  - turn 3：`prompt_at = 90000`, `response_at = NULL`（進行中）
- **THEN** 該 session 的 agent 時間 = 45 秒（30 + 15，turn 3 不計入）

#### Scenario: 跨 session 的 agent 時間加總

- **WHEN** 查詢範圍內有多個 sessions
- **THEN** 各 session 的 agent 時間加總為報表的總 agent 時間

### Requirement: User 主動時間 Idle Threshold 近似計算

系統 SHALL 將 user 主動時間定義為：同一 session 內，相鄰兩個 turns 之間的 gap（第 N 個 turn 的 `prompt_at` 減去第 N-1 個 turn 的 `response_at`），若 gap < idle_threshold，則累加為 user 主動時間；若 gap ≥ idle_threshold，則不計入。

idle_threshold 預設值為 15 分鐘（900000 毫秒），可透過設定覆蓋。

#### Scenario: Gap 小於 idle threshold 計入 user 時間

- **WHEN** idle_threshold = 15 分鐘，session 有 2 個 turns：
  - turn 1：`response_at = 60000`（1 分鐘）
  - turn 2：`prompt_at = 180000`（3 分鐘）
  - gap = 120000 ms = 2 分鐘 < 15 分鐘
- **THEN** user 主動時間累加 2 分鐘

#### Scenario: Gap 大於等於 idle threshold 不計入

- **WHEN** idle_threshold = 15 分鐘，turn 之間 gap = 20 分鐘（1200000 ms）
- **THEN** 該 gap 不計入 user 主動時間

#### Scenario: Session 只有一個 turn 時 user 時間為 0

- **WHEN** session 只有一個 turn，無相鄰 turn
- **THEN** user 主動時間 = 0

### Requirement: Idle Threshold 設定

系統 SHALL 讀取 `tt config` 中的 `idle-threshold`（分鐘，整數），並在聚合計算時使用該值。未設定時預設 15 分鐘。

#### Scenario: 自訂 idle threshold 影響 user 時間計算

- **WHEN** `tt config set idle-threshold 30` 已執行，且 turn gap = 20 分鐘
- **THEN** 20 分鐘 < 30 分鐘，該 gap 計入 user 主動時間
- **WHEN** idle threshold 為預設 15 分鐘，且 turn gap = 20 分鐘
- **THEN** 20 分鐘 ≥ 15 分鐘，該 gap 不計入
