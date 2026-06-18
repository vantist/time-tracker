## ADDED Requirements

### Requirement: 設定與顯示工作項目

系統 SHALL 透過 `tt work` 命令管理工作項目標記：

- `tt work "<label>"` — 寫入工作項目名稱到 `~/.tt/work-item`（純文字檔，含換行）
- `tt work` — 顯示目前追蹤的工作項目名稱，若無則顯示 "No work item set."

工作項目標記在每次 `tt record prompt` 時讀取，並寫入對應 session 的 `work_item` 欄位。

#### Scenario: 設定工作項目

- **WHEN** `tt work "login-redesign"` 被呼叫
- **THEN** `~/.tt/work-item` 內容為 `"login-redesign\n"`
- **THEN** stdout 輸出 `Work item set: login-redesign`
- **THEN** exit code 0

#### Scenario: 顯示目前工作項目

- **WHEN** `~/.tt/work-item` 存在且內容為 `"login-redesign\n"`，呼叫 `tt work`
- **THEN** stdout 輸出 `Current work item: login-redesign`

#### Scenario: 無工作項目時顯示提示

- **WHEN** `~/.tt/work-item` 不存在，呼叫 `tt work`
- **THEN** stdout 輸出 `No work item set.`

### Requirement: prompt 記錄時自動讀取 work_item

系統 SHALL 在 `tt record prompt` 被呼叫時，讀取 `~/.tt/work-item`（若存在），並將其值寫入對應 session 的 `work_item` 欄位（如果 session 的 `work_item` 尚未設定）。

#### Scenario: 已有 work-item 時記錄 prompt 自動填入

- **WHEN** `~/.tt/work-item` 內容為 `"auth-refactor"`，呼叫 `tt record prompt --session abc123 ...`
- **THEN** `sessions.work_item = "auth-refactor"` 在 session 建立或首次 prompt 時寫入

#### Scenario: 無 work-item 檔案時 work_item 保持 NULL

- **WHEN** `~/.tt/work-item` 不存在，呼叫 `tt record prompt ...`
- **THEN** `sessions.work_item` 保持 NULL，不報錯

### Requirement: 清除工作項目

系統 SHALL 在 `tt work --clear` 被呼叫時，刪除 `~/.tt/work-item` 檔案（若存在）。

#### Scenario: 清除工作項目

- **WHEN** `tt work --clear` 被呼叫
- **THEN** `~/.tt/work-item` 被刪除
- **THEN** stdout 輸出 `Work item cleared.`
- **THEN** exit code 0

#### Scenario: 清除不存在的工作項目不報錯

- **WHEN** `~/.tt/work-item` 不存在，呼叫 `tt work --clear`
- **THEN** exit code 0，stdout 輸出 `Work item cleared.`（idempotent）
