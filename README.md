# tt — AI 工作時間追蹤器

透過 Claude Code / Copilot CLI 的 hook 系統，靜默記錄每次 prompt/response 事件，輸出依專案、工作項目分組的時間與費用報表。

單一二進位，無外部 runtime 依賴，資料存於本地 SQLite。

## 安裝

**macOS（curl）**

```sh
curl -fsSL https://raw.githubusercontent.com/vantist/time-checker/main/install.sh | sh
```

**Windows（PowerShell）**

```powershell
irm https://raw.githubusercontent.com/vantist/time-checker/main/install.ps1 | iex
```

安裝至 `%USERPROFILE%\bin\tt.exe`，若該路徑不在 PATH 中，腳本會提示手動加入。

**從原始碼 build**

```sh
go build -o tt ./cmd/tt
```

## 快速開始

1. 安裝 hook（一次性）：

```sh
tt setup --claude-code
```

2. 開始工作，可選標記工作項目：

```sh
tt work "feature-xyz"
```

3. 查看報表：

```sh
tt report
tt report --since 30d --by-work-item
```

## 指令

| 指令 | 說明 |
|------|------|
| `tt setup --claude-code` | 自動寫入 Claude Code hook |
| `tt setup --copilot` | 顯示 Copilot CLI 手動設定說明（尚未實測）|
| `tt work [label]` | 設定 / 顯示 / `--clear` 工作項目標記 |
| `tt report` | 顯示時間與費用報表 |
| `tt config set <key> <val>` | 設定持久化設定值 |
| `tt version` | 顯示版本 |

### `tt report` 選項

| Flag | 說明 | 預設 |
|------|------|------|
| `--since` | 時間範圍：`7d`、`30d`、`YYYY-MM-DD` | `7d` |
| `--project` | 篩選專案名稱 | 全部 |
| `--format` | `text` / `json` | `text` |
| `--by-work-item` | 依工作項目分組 | false |

### `tt config set` 選項

| Key | 說明 | 預設 |
|-----|------|------|
| `idle-threshold` | 使用者閒置判定分鐘數 | `15` |

## Hook 原理

> **注意**：Copilot CLI hook 整合尚未實際測試，設定說明僅供參考。

`tt setup --claude-code` 將以下 hook 合併至 `~/.claude/settings.json`：

```json
{
  "hooks": {
    "UserPromptSubmit": [{"hooks": [{"type": "command", "command": "tt record prompt"}]}],
    "Stop": [{"hooks": [{"type": "command", "command": "tt record response"}]}]
  }
}
```

每次 prompt 觸發 `tt record prompt`（記錄 session、專案、模型），Stop 時觸發 `tt record response`（記錄 token 用量、估算費用）。Hook 失敗靜默處理，不阻擋 AI 工具。

## 資料儲存

- 路徑：`~/.tt/data.db`（SQLite）
- 覆寫：`TT_DB_PATH` env var

```
sessions  — session ID、專案、工具、模型、branch、工作項目
turns     — prompt/response 時間、token 用量、費用
```

## 文件

- [Commands](docs/commands.md) — 完整旗標說明
- [Architecture](ARCHITECTURE.md) — 模組結構與資料流
