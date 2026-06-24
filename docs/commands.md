# Commands

## Build & Run

```bash
go build -o tt ./cmd/tt
./tt --help
```

## CLI 指令

### `tt record prompt`

記錄 prompt 事件（由 AI 工具的 hook 觸發）。

```bash
tt record prompt [flags]
```

| Flag | 說明 | 預設 |
|------|------|------|
| `--session string` | Session ID（覆寫 stdin） | `""` |
| `--project string` | 專案路徑（覆寫 stdin） | `""` |
| `--tool string` | 工具名稱 | `"claude-code"` |
| `--model string` | 模型名稱（覆寫 stdin） | `""` |
| `--transcript-path string` | 對話紀錄 JSONL 路徑（覆寫 stdin） | `""` |

**stdin 格式：**
支援 JSON 物件，可用欄位包括：
- `session_id` / `sessionId` / `conversationId` (適用於 `antigravity`)：Session ID
- `cwd`：目前工作目錄（專案路徑）
- `model`：模型名稱
- `transcript_path` / `transcriptPath`：對話紀錄檔案路徑

---

### `tt record response`

記錄 response 或是停止事件（由 AI 工具的 hook 觸發）。

```bash
tt record response [flags]
```

| Flag | 說明 | 預設 |
|------|------|------|
| `--session string` | Session ID（覆寫 stdin） | `""` |
| `--tokens string` | Token 用量 JSON 字串（覆寫 stdin） | `""` |
| `--subagent-tokens string` | Subagent token JSON 陣列（OpenCode event path） | `""` |
| `--tool string` | 工具名稱 | `"claude-code"` |
| `--model string` | 模型名稱（覆寫 stdin） | `""` |

**stdin 格式：**
支援 JSON 物件，可用欄位包括：
- `session_id` / `sessionId` / `conversationId` (適用於 `antigravity`)：Session ID
- `transcript_path` / `transcriptPath`：對話紀錄檔案路徑（當 `--tokens` 未提供時，會以此路徑解析對話內容並自動計算 token）

---

### `tt work`

管理目前工作項目標記。

```bash
tt work "feature-xyz"   # 設定工作項目
tt work                 # 顯示目前工作項目
tt work --clear         # 清除工作項目
```

| Flag | 說明 | 預設 |
|------|------|------|
| `--clear` | 清除目前的工作項目 | `false` |

儲存於 `~/.tt/work-items/<projectKey>`（由專案路徑 SHA256 決定），下次 `record prompt` 自動帶入。

---

### `tt report`

查詢並顯示使用報表。

```bash
tt report [flags]
```

| Flag | 說明 | 預設 |
|------|------|------|
| `--since string` | 時間範圍：`7d`、`30d` 或 `YYYY-MM-DD` | `"7d"` |
| `--project string` | 篩選專案名稱 | `""` |
| `--format string` | 輸出格式：`text` 或 `json` | `"text"` |
| `--by-work-item` | 依工作項目分組 | `false` |
| `-o, --output string` | 直接將報表內容寫入指定檔案 | `""` |

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

安裝與設定 AI 工具 hook 整合。

```bash
tt setup [flags]
```

| Flag | 說明 | 預設 |
|------|------|------|
| `--claude-code` | 設定 Claude Code hooks | `false` |
| `--copilot` | 設定 GitHub Copilot CLI hooks | `false` |
| `--antigravity` | 設定 Google Antigravity hooks | `false` |
| `--codex` | 設定 OpenAI Codex hooks | `false` |
| `--opencode` | 設定 OpenCode plugin | `false` |
| `--vscode-copilot` | 設定 VS Code Copilot bridge | `false` |

---

### `tt version`

列印當前程式版本。

```bash
tt version
```

此指令無額外 flags。

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
