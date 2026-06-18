## Context

`tt` 是本機 CLI time tracker，資料存於 SQLite。現有 `report.go` 的 `Result` struct 只帶 input tokens 與 total cost，`FormatText` 輸出欄位不完整，`FormatJSON` 也缺少 output tokens 與 cache 相關欄位。無網頁介面。

現有依賴：`cobra`（CLI）、`mattn/go-sqlite3`（DB）、標準庫。不引入任何新外部套件。

## Goals / Non-Goals

**Goals:**
- `Result` struct 補齊 OutputTokens、CacheReadTokens、CacheCreationTokens、ByProject（[]ProjectSummary）
- `FormatText` 輸出含 tokens breakdown 與 by-project 表格
- `FormatJSON` 輸出補齊上述欄位
- `tt serve` 啟動 `net/http` server，提供 HTML dashboard 與 `/api/report` JSON endpoint
- HTML dashboard 純 CSS bar chart，每 60s 自動 fetch 刷新，無外部 JS 框架
- 跨平台開啟瀏覽器（macOS: `open`、Linux: `xdg-open`、Windows: `cmd /c start`）

**Non-Goals:**
- WebSocket 即時推送（polling 已足夠）
- 使用者認證或 HTTPS
- 資料持久化到遠端
- DB schema 變更

## Decisions

### D1：Daily breakdown 後處理而非額外 query

**決定**：在 `Query()` 回傳的 sessions rows 裡，按 `DATE(prompt_at)` group by，於 Go 層後處理產生 `DailyStats`。

**理由**：避免第二次 DB round-trip；現有 rows 已含所有欄位；資料量小（CLI 工具），記憶體後處理無效能疑慮。

**替代方案**：SQL `GROUP BY DATE(prompt_at)` 單獨 query → 增加 DB 耦合、測試複雜度，不值得。

### D2：HTML template inline string 在 `html.go`

**決定**：HTML/CSS/JS 全部 inline 於 `internal/report/html.go` 的 Go string constant，不使用 `embed.FS`。

**理由**：三個檔案的改動量（proposal Impact）已確定，embed 增加建置複雜度；template 不需要熱更新；單檔便於測試 handler。

**替代方案**：`//go:embed` → 需要獨立靜態檔案目錄，增加目錄結構複雜度。

### D3：開啟瀏覽器用 runtime GOOS check

**決定**：`serve_cmd.go` 用 `runtime.GOOS` switch，分別呼叫 `open`/`xdg-open`/`cmd /c start`，不做條件編譯（build tags）。

**理由**：條件編譯需多個檔案，runtime check 一個函式即可；執行路徑相同，無跨平台編譯需求。

### D4：`/api/report` 重用 `FormatJSON`

**決定**：HTTP handler 直接呼叫現有 `report.Query()` + `json.Marshal(result)`，不另建 API 資料層。

**理由**：DRY；`FormatJSON` 補齊欄位後即可滿足前端需求；避免資料結構分歧。

### D5：model 從 transcript 抽取，補寫至 session

**決定**：`extractFromTranscript`（原 `extractTokensFromTranscript`）同時讀取 assistant entry 的 `message.model`，在 `RecordResponse` 中執行 `UPDATE sessions SET model=? WHERE id=? AND (model='' OR model IS NULL)`。

**理由**：Stop hook 已有 `transcript_path`，改動集中在現有函式，無需新 hook 或新 DB column。`INSERT OR IGNORE` 的既有行為不變。

**替代方案**：`UserPromptSubmit` stdin 也有 `model` 欄位，但 gateway 環境下該欄位可能為空；transcript 是 ground truth。

### D6：pricing normalize — 去除 `/` 前綴

**決定**：`pricing.normalize(model string) string` 用 `strings.LastIndex(model, "/")` 取後段，去除任意 gateway 前綴（`vertex_ai/`、`us.anthropic.`、`bedrock/` 等）。

**理由**：`LastIndex` 覆蓋所有已知前綴格式，不需 regex，stdlib 即可。Pricing table key 統一使用不含日期後綴的短 ID（`claude-haiku-4-5`，非 `claude-haiku-4-5-20251001`）。

### D7：GroupResult 永遠計算，groups 納入 FormatJSON

**決定**：`Query()` 永遠執行 `groupByWorkItem`，不依 `opts.ByWorkItem`。`FormatJSON` 加入 `groups` 欄位。前端若 `groups.length <= 1` 則隱藏 By Work Item section。

**理由**：Web 無 flag 控制，永遠計算比加 query param 更簡單；groups 資料量小，後處理無效能疑慮。

### D8：ProjectSummary / SessionRow 補 UserActiveTimeSec

**決定**：`projMap` 改存 `[]aggregator.Turn`（已有 `sessTurns` 可復用），最後呼叫 `aggregator.UserActiveTime`；SessionRow 加 `UserTimeSec` 從 `aggregator.UserActiveTime(sessTurns[sid], threshold)` 計算。

**理由**：aggregator package 已有現成函式，補欄位是最小改動路徑。

## Risks / Trade-offs

- **Port 衝突** → 使用者可用 `--port` flag 指定；預設 7890 為非常用 port，衝突機率低
- **Windows `cmd /c start` 行為差異** → 無法在 CI 中驗證；加 `// ponytail: runtime GOOS, add build tags if cross-compile needed` 標記
- **CSS bar chart 精度** → 純 % 寬度，極小值（< 1%）可能不可見；屬 UI 限制，不影響功能正確性
- **60s polling 延遲** → 使用者手動重整可立即更新；delay 在 CLI 工具情境可接受
- **cacheCreation 用 5m TTL 定價** → DB 無 TTL 欄位，低估 1h cache write cost；屬已知限制，日後可加欄位區分
- **groups 永遠計算** → 若全部 session 無 work item 且同一 branch，groups 為單一元素；前端以 length ≤ 1 隱藏 section
