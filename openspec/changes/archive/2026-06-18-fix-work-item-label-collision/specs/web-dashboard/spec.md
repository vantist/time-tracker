## ADDED Requirements

### Requirement: By Work Item table 顯示 Project 欄

dashboard 的 By Work Item table SHALL 包含 Project 欄，顯示 `GroupResult.Project`（即 `path.Base(project)`），位於 Label 欄右側。

#### Scenario: By Work Item table 包含 Project 欄

- **WHEN** 瀏覽器請求 `GET /` 且 BY WORK ITEM 報表有資料
- **THEN** By Work Item table thead 包含 `Project` 欄位標題
- **THEN** 每列對應的 `<td>` 顯示該 group 的 `path.Base(project)` 值

#### Scenario: 相同 label 不同 project 顯示為獨立列

- **WHEN** 兩個 repo 均有 branch = "main" 的 sessions
- **THEN** By Work Item table 顯示兩列，Label 欄均為 "main"，Project 欄各自顯示不同的 repo 名稱
