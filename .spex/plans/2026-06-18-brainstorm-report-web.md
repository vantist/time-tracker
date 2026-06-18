# Brainstorm: 豐富 report 資訊 + `tt serve` 網頁介面

**日期**: 2026-06-18  
**目標**: 現有 `tt report` 資訊太少，補齊所有 DB 欄位；新增 `tt serve` 提供可長時間閱讀的網頁 dashboard。

---

## 決定

**文字輸出（`tt report`）強化**：補 output tokens、cache read、cache creation、依 Project 分組。  
**網頁（`tt serve`）**：啟動本機 HTTP server，自動開啟瀏覽器，前端每 60s 自動刷新。  
**Daily timeline**：預設 7 天，與 `--since 7d` 一致。  
**Auto-open browser**：是。  

---

## 文字輸出改版（`tt report`）

```
Sessions:        11
Agent time:      1h 14m
User active:     0h 28m
─── Tokens ─────────────────
  Input:         9,000
  Output:        12,345
  Cache read:    5,678
  Cache create:  1,234
─── Cost ────────────────────
  Est. cost:     $0.0234
─── By Project ──────────────
  my-project     8  1h 10m  $0.02
  other          3  0h 05m  N/A
```

---

## `tt serve` 架構

```
tt serve [--port 7890] [--since 7d]
   │
   ├─ 啟動 HTTP server
   ├─ 印出 "Serving at http://localhost:7890"
   ├─ open http://localhost:7890 (macOS: open, Linux: xdg-open)
   └─ handlers:
       GET /           → HTML dashboard (embedded Go template)
       GET /api/report → JSON（擴充現有 FormatJSON）
```

前端：原生 fetch + DOM，每 60s 刷新，不引入外部 JS 框架或 chart 套件。  
Bar chart：純 CSS（div 寬度 %，顏色漸層），不用 canvas/SVG。

---

## Dashboard 區塊

1. **Summary 卡片** — Sessions, Agent time, User active, Input/Output/Cache tokens, Est. cost
2. **Daily timeline** — 7 天 bar chart（橫軸日期，縱軸 sessions count + token 數）
3. **By Project** — table（project, sessions, agent time, tokens, cost）
4. **Session 明細** — table（時間, project, branch, model, turns, agent time, cost）

---

## 涉及的檔案

| 動作 | 檔案 | 說明 |
|------|------|------|
| 修改 | `internal/report/report.go` | `Result` 加 OutputTokens / CacheCreationTokens 顯示；`FormatText` 改版；`FormatJSON` 補齊欄位；`Query` 補 daily breakdown 資料 |
| 新增 | `internal/report/html.go` | HTML template（inline string）+ HTTP handler |
| 新增 | `cmd/tt/serve_cmd.go` | `tt serve` subcommand，port flag，open browser |

不需要新的外部套件（標準庫 `net/http` + 現有 `cobra`）。

---

## Open Questions

- `serve_cmd.go` 的 `--port` 預設值：7890（可調整）
- Daily breakdown：query 時按 `DATE(prompt_at)` GROUP BY，另開一個 query 或在現有 rows 裡後處理？→ 建議後處理（不加新 query）
- Windows 的 `open browser`：用 `cmd /c start` → 要條件編譯或 runtime GOOS check

---

## 結論

最小實作路徑：
1. 改 `report.go`（Result struct + FormatText + FormatJSON + daily grouping 後處理）
2. 新增 `html.go`（template + handler）
3. 新增 `serve_cmd.go`（cobra subcommand）

不需要新套件，不需要 DB schema 改動，不需要 config 改動。
