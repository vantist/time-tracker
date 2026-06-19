# Brainstorm: 區分 report 與 serve 中的 Agent 名稱

**日期**: 2026-06-19  
**目標**: 目前 `report` 與 `serve` 沒有標示 Agent 名稱，導致 Claude Code 與 Copilot CLI 的數據全部混在一起。需要在 CLI 報表與網頁 Dashboard 中新增 Agent (Tool) 欄位與 By Agent 彙整統計，並將 Agent 名稱進行正規化（如 `claude-code` 轉為 `Claude Code`，空值轉換為 `unknown`）。

---

## 決定

1. **資料讀取**：SQL 查詢中新增讀取 `s.tool`，並使用 `Scan` 寫入 internal struct。
2. **名稱正規化**：實作 `normalizeAgentName` 函數，將 `claude-code` 轉為 `Claude Code`，`copilot-cli` 轉為 `Copilot CLI`，空值轉為 `unknown`。
3. **終端報表強化（`tt report`）**：
   * 在 Sessions 表格中插入 `Agent` 欄位（對齊欄寬 `%-12s`）。
   * 在 `─── By Project ───` 下方新增 `─── By Agent ───` 統計表格。
4. **網頁 Dashboard（`tt serve`）**：
   * 新增 `By Agent` 統計卡片/表格（欄位：Agent, Sessions, Agent time, User time, Tokens, Cost）。
   * 在 Sessions 明細表格中，新增 `Agent` 欄位。
5. **資料結構調整**：
   * `Result` 結構體新增 `ByAgent []AgentSummary`。
   * `SessionRow` 結構體新增 `Tool` 欄位。

---

## 涉及的檔案

| 動作 | 檔案 | 說明 |
|------|------|------|
| 修改 | `internal/report/report.go` | 1. 調整 `rowData`、`SessionRow`、`Result` struct，並新增 `AgentSummary` struct。<br>2. 修改 SQL 查詢與 `rows.Scan` 邏輯以讀取 `tool`。<br>3. 實作 `normalizeAgentName`。<br>4. 在 `Query` 中新增 `ByAgent` 彙整計算邏輯。<br>5. 更新 `FormatText` 輸出，加入 `By Agent` 表格與 Sessions 欄位。 |
| 修改 | `internal/report/html.go` | 1. 調整 `dashboardHTML` 加入 `By Agent` 區塊與 Sessions `Agent` 欄位。<br>2. 修改 JS `render` 函數渲染 `by_agent` 並在 Sessions 列中補上 `esc(s.tool)`。 |

---

## 設計細節

### 1. SQL 查詢擴充
```sql
SELECT s.id, s.project, s.branch, s.work_item,
       COALESCE(s.model, ''), s.started_at,
       t.prompt_at, t.response_at,
       COALESCE(t.input_tokens, 0),
       COALESCE(t.output_tokens, 0),
       COALESCE(t.cache_read_tokens, 0),
       COALESCE(t.cache_creation_tokens, 0),
       t.estimated_cost_usd,
       COALESCE(s.tool, '') -- 新增
FROM turns t
JOIN sessions s ON s.id = t.session_id
WHERE t.prompt_at >= ?
ORDER BY s.id, t.id
```

### 2. 名稱正規化
```go
func normalizeAgentName(tool string) string {
	tool = strings.TrimSpace(strings.ToLower(tool))
	if tool == "" {
		return "unknown"
	}
	switch tool {
	case "claude-code", "claudecode", "claude":
		return "Claude Code"
	case "copilot-cli", "copilotcli", "copilot":
		return "Copilot CLI"
	default:
		return tool
	}
}
```

### 3. ByAgent 統計計算
比照 `ByProject`，建立內部 `agentMap`，並用 `aggregator.MergeAndSum` 分別計算各個 Agent 獨立且不重疊的 User Active Time。

---

## Open Questions

無。此變更不涉及 DB Schema 異動或設定檔調整，僅為唯讀報表呈現層的擴充。

---

## 結論

**決策**: 採用方案 C  
**原因**: 能在展示 By Agent 總彙整資料的同時，透過正規化顯示更為專業的工具名稱，並對歷史遺留空值（`unknown`）進行防呆處理。  
**選定做法**: 擴充 `internal/report/report.go` 與 `internal/report/html.go`，修改 CLI 輸出與 Web dashboard JS。
