# Work Item Label 跨 Repo 衝突修正

<!-- Brainstorm plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). -->

## Context

BY WORK ITEM 報表中，label 優先順序為 `work_item > branch > "untagged"`。
當使用者未手動設定 `work_item` 時，fallback 到 branch name（如 "main"）。
問題：不同 repo 的 "main" branch 會被 `groupByWorkItem` 合併成同一列，
造成跨 repo 的 sessions/cost 混算，無法辨別來源。

根本原因：`report.go:332-338` 的 groupBy key 純字串 label，不含 project 資訊。

## Decision

**選項 B**：groupBy key 改為 `(project, label)` 複合 key。
UI 畫面在 By Work Item table 新增 Project 欄，顯示 `path.Base(project)`。

## Rationale

- branch name 不具跨 repo 唯一性，複合 key 是最根本的修法
- `rowData.project` 已存在，改動極小（不動 DB schema、不動 recorder）
- 手動設定的 `work_item` 仍正常運作，只是 groupBy key 多帶 project
- Project 欄讓使用者在畫面上直接辨別來源，無需點進 session 明細

## Approach

### `internal/report/report.go`

1. `GroupResult` struct 新增 `Project string` 欄位
2. `groupByWorkItem` 修改 key 邏輯：

```go
// before
labelOf := map[string]string{} // sessionID → label

// after
labelOf   := map[string]string{} // sessionID → composite key (project|label)
projectOf := map[string]string{} // sessionID → project
displayOf := map[string]string{} // sessionID → display label (branch/workItem only)

for _, r := range rows {
    if _, seen := labelOf[r.sessionID]; !seen {
        label := r.workItem
        if label == "" {
            label = r.branch
        }
        if label == "" {
            label = "untagged"
        }
        key := r.project + "|" + label
        labelOf[r.sessionID]   = key
        projectOf[r.sessionID] = r.project
        displayOf[r.sessionID] = label
    }
    ...
}
```

3. `GroupResult` 建構時帶入 `Project: path.Base(g.project)`
4. `groupState` 加 `project string` 欄位

### `internal/report/html.go`

By Work Item table：
- thead 加 `<th>Project</th>`（Label 欄右側）
- tbody 渲染加對應欄位（`d.project`）

## Design Notes

- `path.Base(project)` 取最後一段目錄名作為 Project 欄顯示，避免完整路徑太長
- 若 `work_item` 有值，label = workItem，project 仍帶入複合 key，確保跨 repo workItem 同名時也能區隔
- 不影響 `GroupResult.Label` 欄（仍顯示原始 branch/workItem，不含 project 路徑）

## Insights to Capture

無新的 insight（純 groupBy key 修正）

## Open Questions

無
