# Commands

## Build & Run

```bash
go build -o tt ./cmd/tt
./tt --help
```

## CLI 指令

### `tt record prompt`

記錄 prompt 事件（由 Claude Code `UserPromptSubmit` hook 觸發）。

```bash
tt record prompt [flags]
```

| Flag | 說明 |
|------|------|
| `--session string` | Session ID（stdin 優先） |
| `--project string` | 專案路徑（stdin 優先） |
| `--tool string` | 工具名稱（預設：`claude-code`） |
| `--model string` | 模型名稱（stdin 優先） |

stdin 格式：`{"session_id":"...","cwd":"/path","model":"claude-sonnet-4-6"}`

---

### `tt record response`

記錄 response 事件（由 Claude Code `Stop` hook 觸發）。

```bash
tt record response [flags]
```

| Flag | 說明 |
|------|------|
| `--session string` | Session ID（stdin 優先） |
| `--tokens string` | Token 用量 JSON |
| `--tool string` | 工具名稱（預設：`claude-code`） |

---

### `tt work`

管理目前工作項目標記。

```bash
tt work "feature-xyz"   # 設定工作項目
tt work                 # 顯示目前工作項目
tt work --clear         # 清除工作項目
```

儲存於 `~/.tt/work-item`，下次 `record prompt` 自動帶入。

---

### `tt report`

查詢並顯示使用報表。

```bash
tt report [flags]
```

| Flag | 說明 | 預設 |
|------|------|------|
| `--since string` | 時間範圍：`7d`、`30d`、`YYYY-MM-DD` | `7d` |
| `--project string` | 篩選專案名稱 | （全部） |
| `--format string` | 輸出格式：`text` 或 `json` | `text` |
| `--by-work-item` | 依工作項目分組 | false |

---

### `tt serve`

啟動 Web dashboard，在瀏覽器呈現互動式報表。

```bash
tt serve [flags]
```

| Flag | 說明 | 預設 |
|------|------|------|
| `--port int` | HTTP server 埠號 | `7890` |
| `--since string` | 時間範圍：`7d`、`30d`、`YYYY-MM-DD` | `7d` |

啟動後自動開啟瀏覽器至 `http://localhost:7890`，每 60 秒自動重新整理。

---

### `tt setup`

安裝 AI 工具 hook 整合。

```bash
tt setup --claude-code   # 自動寫入 ~/.claude/settings.json
tt setup --copilot       # 顯示 Copilot CLI 手動設定說明
```

---

### `tt config set`

設定持久化設定值。

```bash
tt config set <key> <value>
```

| Key | 說明 | 預設 |
|-----|------|------|
| `idle-threshold` | 使用者閒置判定分鐘數 | `15` |

設定儲存於 `~/.tt/config.json`。

---

## Hook 設定範例

### Claude Code（`~/.claude/settings.json`）

```json
{
  "hooks": {
    "UserPromptSubmit": [{"hooks": [{"type": "command", "command": "tt record prompt"}]}],
    "Stop": [{"hooks": [{"type": "command", "command": "tt record response"}]}]
  }
}
```

### Copilot CLI（`~/.copilot/settings.json`）

```json
{
  "hooks": {
    "userPromptSubmitted": [{"command": "tt record prompt --tool copilot-cli"}],
    "agentStop": [{"command": "tt record response --tool copilot-cli"}]
  }
}
```

---
Related: [Architecture](../ARCHITECTURE.md) | [Conventions](conventions.md) | [Configuration](configuration.md)
