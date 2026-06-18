## ADDED Requirements

### Requirement: Per-project work item storage

系統 SHALL 以 project key 為索引儲存和查詢 work item，使不同 project 的 work item 互相獨立。

Project key 解析規則：
1. 若 `dir` 在 git repository 內，使用 `git rev-parse --show-toplevel` 輸出（git root 絕對路徑）
2. 否則使用 `dir` 本身（CWD 絕對路徑）

儲存路徑 SHALL 為 `~/.tt/work-items/<sha256[:16]>`，其中 `sha256[:16]` 為 resolved project key 的 SHA-256 hex digest 前 16 字元。

#### Scenario: 在 git repo 內 Set 後可於同 repo 任意子目錄 Get

- **WHEN** 使用者在 repo 的子目錄執行 `tt work set "JIRA-123"` 然後切換到另一個子目錄執行 `tt work`
- **THEN** 系統顯示 `JIRA-123`

#### Scenario: 不同 repo 的 work item 互相隔離

- **WHEN** 使用者在 repo-A 設定 work item 為 `"TICKET-A"`，然後在 repo-B 設定為 `"TICKET-B"`
- **THEN** 在 repo-A 執行 `tt work` 顯示 `TICKET-A`，在 repo-B 顯示 `TICKET-B`

#### Scenario: 非 git 目錄使用 CWD 作為 key

- **WHEN** 使用者在非 git 目錄 `/tmp/scratch` 設定 work item 為 `"LOCAL"`
- **THEN** 在相同目錄執行 `tt work` 顯示 `LOCAL`；在其他目錄查詢不受影響

### Requirement: Work item key 使用 SHA-256 雜湊

系統 SHALL 使用 resolved project path 的 SHA-256 hex digest 前 16 字元作為儲存檔名，以避免路徑特殊字元問題並保持固定長度。

#### Scenario: 路徑包含空格或特殊字元

- **WHEN** project path 為 `/Users/user/my projects/repo`
- **THEN** 系統正確計算 SHA-256 hash 並儲存，不因空格而失敗

### Requirement: RecordPrompt 附加 per-project work item

系統 SHALL 在 `RecordPrompt` 時，以 `input.Project`（CWD）為 project key 查詢 work item，並將其附加到 session 紀錄。

#### Scenario: 有 work item 時附加到 session

- **WHEN** `input.Project` 對應的 project 有設定 work item `"PROJ-42"`
- **THEN** session 紀錄中 `WorkItem` 欄位為 `"PROJ-42"`

#### Scenario: 無 work item 時 WorkItem 為空

- **WHEN** `input.Project` 對應的 project 沒有 work item
- **THEN** session 紀錄中 `WorkItem` 欄位為空字串

### Requirement: Clear 移除指定 project 的 work item

系統 SHALL 在 `tt work clear` 時僅移除當前 project 的 work item，不影響其他 project。

#### Scenario: Clear 後 Get 回傳空

- **WHEN** 使用者在 repo-A 執行 `tt work clear`
- **THEN** repo-A 的 `tt work` 回傳空，repo-B 的 work item 不受影響
