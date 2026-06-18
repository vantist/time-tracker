## ADDED Requirements

### Requirement: By Project table 顯示 user active time

web dashboard 的 By Project table SHALL 包含 `user_active_time_sec` 欄位，以 `Xh Ym` 格式顯示。

#### Scenario: By Project user time 正確顯示

- **WHEN** 瀏覽器請求 `GET /` 且有多個 project 的 sessions
- **THEN** By Project table 每列 MUST 包含該 project 的 user active time，格式同 agent time

### Requirement: Session 明細 table 顯示 user active time 與 work item

web dashboard 的 Session 明細 table SHALL 包含 `user_time_sec` 欄位及 `work_item` 欄位。

#### Scenario: Session 明細含 work item

- **WHEN** session 有設定 work item
- **THEN** Session 明細 table 該列 MUST 顯示 work item 值

#### Scenario: Session 明細 work item 為空

- **WHEN** session 未設定 work item
- **THEN** work item 欄位顯示空白，不顯示佔位文字

#### Scenario: Session 明細含 user active time

- **WHEN** 瀏覽器請求 `GET /` 且 DB 有 sessions
- **THEN** Session 明細 table 每列 MUST 包含 user active time（`Xh Ym` 格式）

### Requirement: 新增 By Work Item 分組 section

web dashboard SHALL 包含 By Work Item section，依 work item（無則 fallback 至 branch）分組，顯示 sessions、agent time、user time、cost。

#### Scenario: By Work Item section 渲染

- **WHEN** 瀏覽器請求 `GET /` 且 `/api/report` 回傳非空 `groups` 陣列
- **THEN** dashboard MUST 顯示 By Work Item table，每列包含 label、sessions、agent time、user time、cost

#### Scenario: groups 唯一且為 main 時隱藏 section

- **WHEN** `/api/report` 的 `groups` 陣列長度 ≤ 1
- **THEN** By Work Item section SHALL 隱藏（不渲染或 CSS display:none）

#### Scenario: work item 分組優先於 branch

- **WHEN** session 有 work_item 值
- **THEN** By Work Item 以 work_item 值為 label；若 work_item 為空則 fallback 至 branch；若兩者皆空則 label 為 `untagged`

### Requirement: /api/report 永遠回傳 groups

`GET /api/report` SHALL 永遠在回應 JSON 中包含 `groups` 陣列，不依任何 query parameter 控制。

#### Scenario: groups 陣列存在於 JSON

- **WHEN** 請求 `GET /api/report`
- **THEN** JSON 根物件 MUST 含 `groups` 陣列（可為空陣列 `[]`）

### Requirement: ProjectSummary 與 SessionRow 補充欄位

`/api/report` 回傳的 JSON 中，`by_project` 每個元素 SHALL 含 `user_active_time_sec`（int）；`sessions` 每個元素 SHALL 含 `user_time_sec`（int）及 `work_item`（string）。

#### Scenario: by_project 含 user_active_time_sec

- **WHEN** 請求 `GET /api/report`
- **THEN** `by_project[0]` MUST 含 `user_active_time_sec` 欄位（integer）

#### Scenario: sessions 含 user_time_sec 與 work_item

- **WHEN** 請求 `GET /api/report`
- **THEN** `sessions[0]` MUST 含 `user_time_sec`（integer）及 `work_item`（string，可為空）
