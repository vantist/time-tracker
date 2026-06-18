## ADDED Requirements

### Requirement: BY WORK ITEM 以複合 key 分組

系統 SHALL 在 `groupByWorkItem` 中以 `(project, label)` 複合 key 分組 sessions，
而非純字串 label，以確保不同 repo 的相同 branch name 不被合併計算。

label 優先順序維持不變：`work_item > branch > "untagged"`。
複合 key 格式為 `project + "|" + label`，僅用於內部分組，不暴露於 `GroupResult.Label`。

`GroupResult` SHALL 新增 `Project string` 欄位，值為 `path.Base(project)`。

#### Scenario: 相同 branch 不同 repo 分成不同列

- **WHEN** repo A 和 repo B 均有 branch = "main" 的 sessions，且均未設定 work_item
- **THEN** BY WORK ITEM 報表產生兩列，分別對應 repo A 和 repo B，sessions/cost 各自計算，不合併

#### Scenario: 相同 work_item 不同 repo 分成不同列

- **WHEN** repo A 和 repo B 均有 work_item = "feature-x" 的 sessions
- **THEN** BY WORK ITEM 報表產生兩列，`GroupResult.Label` 均為 "feature-x"，`GroupResult.Project` 分別為各自的 `path.Base(project)`

#### Scenario: GroupResult.Label 不含 project 資訊

- **WHEN** repo A 的 branch = "main" 的 session 被分組
- **THEN** `GroupResult.Label` = "main"（不含 project 路徑）
- **THEN** `GroupResult.Project` = `path.Base(repo A path)`
