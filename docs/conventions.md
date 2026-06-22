# Conventions — 專案開發與 Commit 規範

本文件彙整了 `tt` 專案中的程式碼風格規範、Git 工作流、Commit 訊息格式以及 CI/CD 檢查流程，供開發人員與 AI Agent 參考遵循。

## 程式碼開發風格

1. **開發環境與工具**：
   - **Go 語言版本**：使用 Go `1.26.4`（詳見 [go.mod](file:///Users/vantist_lin/Documents/playground/time-tracker/go.mod)）。
   - **本地建置指令**：`go build -o tt ./cmd/tt`。
   - **本地測試指令**：`go test ./...`。

2. **模組架構與職責單一 (Single Responsibility)**：
   - `internal/` 下的套件必須專注於單一職責：
     - `db`：Database Schema 管理與連線。
     - `recorder`：負責 prompt/response 事件寫入。
     - `report`：負責報表數據讀取與查詢。
     - `aggregator`：負責時間區間與活動度計算。
     - `pricing`：負責 Token 費用估算。
   - 避免跨套件循環依賴。底層工具庫（如 `pricing`、`process`、`transcript`）必須保持完全獨立。

3. **強健性與靜默處理 (Graceful Failure)**：
   - **Hook 靜默失敗**：Hook 程式執行出錯或連線失敗時，必須以 `exit 0` 退出，僅將錯誤訊息輸出至 `stderr`，絕對不能阻擋或中斷呼叫端（如 Claude Code/Copilot CLI）的正常開發工作（詳見 [silent.go](file:///Users/vantist_lin/Documents/playground/time-tracker/internal/recorder/silent.go) 的 `RecordPromptSilent` 與 `RecordResponseSilent` 實作）。
   - **輸入優先級**：`stdin` 傳入的 JSON 優先於 CLI flag；flag 僅保留作為測試與手動執行時的備用手段。
   - **SQLite 資料庫**：使用純 Go 的 `modernc.org/sqlite`（無 CGO 依賴），確保跨平台交叉編譯（Cross-compilation）順暢（詳見 [schema.go](file:///Users/vantist_lin/Documents/playground/time-tracker/internal/db/schema.go)）。
   - **測試隔離**：測試時必須使用 `TT_DB_PATH` 環境變數指定獨立的測試資料庫檔案，避免干擾本機資料（測試時可利用 `t.TempDir()` 動態建立獨立目錄，詳見 [schema_test.go](file:///Users/vantist_lin/Documents/playground/time-tracker/internal/db/schema_test.go)）。

---

## Git 工作流與 Commit 規範

### 1. Commit 訊息格式
專案嚴格遵守 [Conventional Commits](https://www.conventionalcommits.org/) 格式。所有的 commit 訊息必須符合以下結構：
```
<type>[optional scope][optional !]: <description>
```

- **允許的 `<type>`**：
  - `feat` (新功能)
  - `fix` (修正問題)
  - `docs` (文件變更)
  - `style` (格式調整，如空白、縮排，不影響代碼邏輯)
  - `refactor` (代碼重構，無功能新增或修正)
  - `perf` (提升效能)
  - `test` (新增或修正測試代碼)
  - `chore` (構建工具或輔助套件之變更)
  - `ci` (CI/CD 流程設定變更)
  - `build` (建置系統或相依套件變更)
  - `revert` (還原先前的 commit)

- **自動化發布**：
  - 本專案使用 `release-please` 自動管理語義化版本（Semantic Versioning）與自動生成 CHANGELOG。
  - **重要**：不合規的 commit 訊息會導致 `release-please` 在自動發佈時直接跳過，不計入版本號與 Changelog。

---

## CI/CD 檢查流程

所有的 CI/CD 工作流定義於 `.github/workflows/`：

1. **PR 標題檢查 ([pr-lint.yml](file:///Users/vantist_lin/Documents/playground/time-tracker/.github/workflows/pr-lint.yml))**：
   - 當 Pull Request 進行開立、編輯、重新開啟或同步時，會觸發 `amannn/action-semantic-pull-request@v5`。
   - 檢查 PR 標題是否符合 Conventional Commits 格式，若不符合則阻擋 PR 合併。
   - 採用 `RELEASE_PLEASE_TOKEN` 作為存取憑證。

2. **自動釋出管理 ([release-please.yml](file:///Users/vantist_lin/Documents/playground/time-tracker/.github/workflows/release-please.yml))**：
   - 當 Commit 被合併至 `main` 分支後自動觸發，利用 `googleapis/release-please-action@v4` 產生或更新 Release PR。
   - 若為 `release-please` 的發布 PR（分支名稱為 `release-please--branches--main`），CI 會藉由 GitHub CLI 自動啟用 `--auto --merge`，於檢查通過後自動完成發布 PR 合併。

3. **自動建置與釋出二進位檔 ([build.yml](file:///Users/vantist_lin/Documents/playground/time-tracker/.github/workflows/build.yml))**：
   - 當推送 `v*` 格式的版號 tag 時觸發。
   - 在 `ubuntu-latest` 環境中編譯並打包為以下三個平台的二進位檔：
     - `tt-darwin-amd64` (macOS Intel)
     - `tt-darwin-arm64` (macOS Apple Silicon)
     - `tt-windows-amd64.exe` (Windows 64-bit)
   - 建置時設定 `CGO_ENABLED: 0` 以確保跨平台移植性，並使用 `-ldflags "-X main.version=$VERSION"` 將當前 tag 版本注入二進位檔。
   - 最終藉由 `softprops/action-gh-release@v2` 將二進位附件自動上傳至 GitHub Release。

---
Related: [Architecture](../ARCHITECTURE.md) | [Commands](commands.md) | [Configuration](configuration.md)
