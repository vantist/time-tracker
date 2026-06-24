# tt — AI Tool Time Tracker

Go CLI，透過 Claude Code / Copilot CLI / Antigravity / Codex / OpenCode / VS Code Copilot hook 自動記錄 AI 工作時間與 token 費用。本地 SQLite，單一二進位，零 runtime 依賴。

## Quick Start

```bash
go build -o tt ./cmd/tt   # 建置
go test ./...             # 執行測試
tt setup                  # 自動偵測並安裝 hooks
tt report --since 7d      # 查看過去 7 天報表
tt work "feature-xyz"     # 標記目前工作項目
```

## 文件

- [ARCHITECTURE.md](ARCHITECTURE.md) — 模組結構、資料流、DB schema、設計決策
- [docs/commands.md](docs/commands.md) — 完整指令參考、flag 說明、hook 設定範例
- [docs/conventions.md](docs/conventions.md) — 專案開發與 Commit 規範
- [docs/configuration.md](docs/configuration.md) — 系統變數、設定檔與配置鍵
- [design.md](design.md) — Hook 整合設計筆記（Claude Code / Copilot CLI stdin 格式）

## Commit 規範

所有 commit 訊息必須遵守 [Conventional Commits](https://www.conventionalcommits.org/) 格式：

```
<type>[optional scope][optional !]: <description>
```

允許的 type：`feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `ci`, `build`, `revert`

範例：
- `feat: add session export command`
- `fix(db): handle null timestamp on migration`
- `chore: update go dependencies`

**不合規的 commit 會導致 release-please 跳過該 commit，不計入版本號與 changelog。**

## 核心慣例

- `internal/` 所有 package 職責單一：db（schema/連線）、recorder（寫入）、report（讀取）、aggregator（時間及工時計算）、pricing（費用）、setup（hook 安裝）、transcript（日誌解析）等
- Hook 失敗靜默處理（exit 0），不阻擋 AI 工具
- stdin JSON 優先於 CLI flag；flag 保留供測試用
- `TT_DB_PATH` env var 覆寫 DB 路徑（測試隔離用）
