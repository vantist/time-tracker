## ADDED Requirements

### Requirement: User 主動時間語義定義

系統 SHALL 將每個 turn 的 user 主動時間定義為 `response_at[i-1] → prompt_at[i]` 的間隔，代表使用者閱讀回覆、思考並輸入下一個 prompt 的時間。

#### Scenario: 正常相鄰 turn 產生一個 interval

- **WHEN** session 有 2 個 turns：turn 1 `response_at = T1`，turn 2 `prompt_at = T2`（T2 > T1）
- **THEN** 產生一個 user interval `[T1, T2]`，長度 = `T2 - T1`

#### Scenario: 前一個 turn response_at 為 nil 時略過

- **WHEN** turn i-1 的 `response_at` 為 nil（agent 未回應）
- **THEN** turn i 不產生 user interval

#### Scenario: Session start 產生第一段 interval

- **WHEN** session 的 `session_start` 不為零值，且 turns[0].prompt_at 存在
- **THEN** 產生 interval `[session_start, turns[0].prompt_at]`

#### Scenario: Session start 為零值時略過第一段 interval

- **WHEN** session 的 `session_start` 為零值（未記錄）
- **THEN** 不產生第一段 interval，從 turn 1 開始計算

### Requirement: Idle Threshold 套用於單一 Interval

系統 SHALL 對每個 user interval 套用 idle threshold 過濾：若 `interval.End - interval.Start >= idleThreshold`，則丟棄該 interval，不計入 user 主動時間。

#### Scenario: Interval 長度小於 idle threshold 保留

- **WHEN** idleThreshold = 15 分鐘，interval 長度 = 10 分鐘
- **THEN** 該 interval 保留並計入 user 時間（貢獻 10 分鐘）

#### Scenario: Interval 長度大於等於 idle threshold 丟棄

- **WHEN** idleThreshold = 15 分鐘，interval 長度 = 20 分鐘
- **THEN** 該 interval 丟棄，不計入 user 時間（貢獻 0 分鐘）

### Requirement: 多 Session 重疊 Interval Merge

系統 SHALL 在每個聚合層級（總計、ByProject、ByWorkItem）收集所有相關 sessions 的 user intervals，執行 merge 去重後再加總，避免並行 session 重疊時段被重複計算。

#### Scenario: 兩個 session 無重疊時間直接加總

- **WHEN** session A 有 interval `[10:00, 10:05]`（5 分鐘），session B 有 interval `[10:10, 10:15]`（5 分鐘）
- **THEN** merge 結果為 `[10:00, 10:05]` 和 `[10:10, 10:15]`，total user time = 10 分鐘

#### Scenario: 兩個 session 有重疊時間合併去重

- **WHEN** session A 有 interval `[10:00, 10:10]`（10 分鐘），session B 有 interval `[10:05, 10:15]`（10 分鐘）
- **THEN** merge 結果為 `[10:00, 10:15]`（15 分鐘），total user time = 15 分鐘（非 20 分鐘）

#### Scenario: 三個 session 多重重疊正確合併

- **WHEN** session A `[10:00, 10:08]`，session B `[10:03, 10:12]`，session C `[10:10, 10:20]`
- **THEN** merge 結果為 `[10:00, 10:20]`，total user time = 20 分鐘
