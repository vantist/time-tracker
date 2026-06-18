## Context

`report.go:groupByWorkItem` 以純字串 label（branch name 或 work_item）作為分組 key。
當使用者未設定 work_item 時，fallback 為 branch name。
不同 repo 的 branch name 可能相同（如 "main"），導致跨 repo sessions 被合併計算。

`rowData.project` 欄位已存在於查詢結果，`GroupResult` 目前未帶入 project 資訊。

## Goals / Non-Goals

**Goals:**
- groupBy key 改為 `(project, label)` 複合 key，確保跨 repo 隔離
- `GroupResult` 新增 `Project` 欄位，供 UI 顯示
- By Work Item HTML table 新增 Project 欄

**Non-Goals:**
- 不修改 DB schema
- 不修改 recorder 端邏輯
- 不影響 By Work Item 以外的報表模式
- 手動設定的 work_item 行為不變（仍以 work_item 為 label，project 加入複合 key）

## Decisions

### D1：複合 key 格式為 `project + "|" + label`

`"|"` 不會出現在合法的 project path 或 branch name 中，safe 作為分隔符。
不用 struct key（`map[struct]`）是因為現有 map 操作以字串為主，改動最小。

替代方案：使用 `map[struct{ project, label string }]` — 型別更嚴謹，但需重構所有 map 操作，改動量較大。

### D2：`GroupResult.Project` 儲存 `path.Base(project)`

完整 project path 在 UI 過長。`path.Base` 取最後一段目錄名，可識別且簡短。

替代方案：儲存完整 path — UI 可自行截斷，但邏輯分散。

### D3：維持兩個輔助 map：`projectOf`、`displayOf`

原本只有 `labelOf`（sessionID → label）。新增：
- `projectOf`（sessionID → project）：建構 `GroupResult.Project`
- `displayOf`（sessionID → display label）：`GroupResult.Label` 仍顯示原始 branch/work_item，不含 project

複合 key 只用於 `groupStates` map，不暴露到 `GroupResult.Label`。

## Risks / Trade-offs

- **現有快照測試**：HTML table 欄位數變動，現有測試需更新 → 先寫失敗測試再修正（TDD）
- **相同 work_item 跨 repo**：複合 key 確保隔離，但 UI 會出現多列相同 label — 使用者需透過 Project 欄區分（可接受，符合設計目標）

## Migration Plan

純程式碼修改，無 DB schema 變動，無需 migration。部署即生效。
