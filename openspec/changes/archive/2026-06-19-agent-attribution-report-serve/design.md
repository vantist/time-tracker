## Context

目前 `tt` 在資料庫中已包含 `sessions.tool` 欄位來記錄 AI Agent 的名稱（例如 `claude-code` 或 `copilot-cli`）。然而，現有的 `tt report` 與 `tt serve` 指令並未讀取或顯示此欄位，導致所有統計數據與 Session 清單混在一起。我們需要在此變更中實現 Agent 的區分、正規化以及依 Agent 的統計彙整。

## Goals / Non-Goals

**Goals:**
- 在 SQL 查詢中讀取 `s.tool` 並將其掃描至內部的 `SessionRow` 與 `rowData` 中。
- 實現 Agent 名稱正規化（例如將 `claude-code` 正規化為 `Claude Code`，空值轉換為 `unknown`）。
- 更新 `tt report` 文字輸出：
  - 在 Sessions 表格中新增 `Agent` 欄位（對齊欄寬 `%-12s`）。
  - 在 `─── By Project ───` 下方新增 `─── By Agent ───` 統計表格。
- 更新 `tt serve` 網頁儀表板：
  - 在 Sessions 明細表格中新增 `Agent` 欄位。
  - 新增 `By Agent` 統計卡片與表格。
- 更新 `/api/report` JSON API，加入 `by_agent` 彙整數據。

**Non-Goals:**
- 本變更不修改 SQLite 資料庫 Schema，因為 `sessions.tool` 欄位已存在。
- 不調整 CLI 命令參數或設定檔結構。

## Decisions

### 1. SQL 查詢與資料結構擴充
在 `internal/report/report.go` 中，將 `Query` 內 SQL 的 `SELECT` 欄位擴充，加入 `COALESCE(s.tool, '')`。
擴充結構體：
- `SessionRow` 新增 `Tool string` 欄位。
- `rowData` 新增 `tool string` 欄位。
- `Result` 新增 `ByAgent []AgentSummary`。
- 新增 `AgentSummary` 結構體定義：
  ```go
  type AgentSummary struct {
      Agent     string  `json:"agent"`
      Sessions  int     `json:"sessions"`
      AgentTime string  `json:"agent_time"`
      UserTime  string  `json:"user_time"`
      Tokens    string  `json:"tokens"`
      Cost      float64 `json:"cost"`
  }
  ```

### 2. 名稱正規化實作
在 `internal/report/report.go` 中實作 `normalizeAgentName(tool string) string`。
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

### 3. By Agent 獨立時間計算
為避免多個 Agent 交錯工作時，其 User Active Time 被錯誤合併（或跨 Agent 重疊），我們必須針對各個 Agent 獨立建立其時間區間陣列，並分別調用 `aggregator.MergeAndSum` 來計算該 Agent 專屬的 User Active Time。

### 4. 前端與 API 調整
- 調整 `internal/report/html.go` 中的 `dashboardHTML`：
  - 於 UI 中新增 By Agent 表格與卡片。
  - Sessions 表格的 header 與 body 新增 `Agent` 欄位。
  - 調整 JS 渲染邏輯，於 Sessions 渲染時加入對 `esc(s.tool)` 的顯示。
  - 將 API 回傳的 `by_agent` 渲染至 By Agent 區域。

## Risks / Trade-offs

- **[Risk]** 歷史遺留的 NULL 或空字串 `tool` 值會被解析為空字串，可能在 UI 顯示不美觀。
  - **Mitigation**：透過 `normalizeAgentName` 將空字元轉換成 `"unknown"` 以作防呆與明確識別。
- **[Risk]** 跨 Agent 時間重疊導致總 Active Time 重複計算。
  - **Mitigation**：這在 By Agent 維度上是預期的，因為每個 Agent 分開計算各自的 Active Time，但在 Project 總計或全域總計中依然要保持原始的不重複時間計算。
