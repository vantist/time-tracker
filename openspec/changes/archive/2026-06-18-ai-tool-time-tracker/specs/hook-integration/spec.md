## ADDED Requirements

### Requirement: Claude Code Hook 設定

系統 SHALL 在 `tt setup --claude-code` 被呼叫時，在 `~/.claude/settings.json` 中加入以下 hooks（不覆蓋現有 hooks，以 merge 方式加入）：

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "tt record prompt --session $CLAUDE_SESSION_ID --project $CLAUDE_PROJECT_PATH --tool claude-code --model $CLAUDE_MODEL"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "tt record response --session $CLAUDE_SESSION_ID --tokens \"$CLAUDE_USAGE_JSON\""
          }
        ]
      }
    ]
  }
}
```

#### Scenario: 首次設定 Claude Code hooks

- **WHEN** `tt setup --claude-code` 被呼叫，且 `~/.claude/settings.json` 不存在或不含 `hooks` 欄位
- **THEN** 在 `~/.claude/settings.json` 中加入 `UserPromptSubmit` 與 `Stop` hooks
- **THEN** stdout 輸出 `Claude Code hooks configured in ~/.claude/settings.json`

#### Scenario: 設定時保留現有 hooks

- **WHEN** `~/.claude/settings.json` 已存在其他 hooks（例如 caveman mode hook）
- **THEN** 現有 hooks 不被覆蓋或刪除，tt hooks 以追加方式加入

### Requirement: Claude Code Hook 事件欄位對照

系統 SHALL 正確解析 Claude Code hook 呼叫時的環境變數或 stdin payload：

| 事件 | 資料來源 | 欄位 |
|------|----------|------|
| `UserPromptSubmit` | 環境變數 | `CLAUDE_SESSION_ID`, `CLAUDE_PROJECT_PATH`, `CLAUDE_MODEL` |
| `Stop` | 環境變數 | `CLAUDE_SESSION_ID`, `CLAUDE_USAGE_JSON`（JSON 字串） |

**注意**：實際欄位名稱需在實作時確認 Claude Code hook 文件，若與上述不符，以實際文件為準。

#### Scenario: Stop hook 帶有完整 token 資訊

- **WHEN** Claude Code `Stop` hook 觸發，`CLAUDE_USAGE_JSON = '{"input_tokens":5000,"output_tokens":800,"cache_read_tokens":3000,"cache_creation_tokens":0}'`
- **THEN** `tt record response` 正確解析並寫入所有 token 欄位

#### Scenario: Stop hook 不帶 token 資訊

- **WHEN** Claude Code `Stop` hook 觸發，`CLAUDE_USAGE_JSON` 為空或不存在
- **THEN** `tt record response` 寫入 `response_at` 並更新 `ended_at`，token 欄位寫入 NULL

### Requirement: Copilot CLI Hook 設定說明

系統 SHALL 透過 `tt setup --copilot` 輸出 Copilot CLI hook 設定的指引（不自動寫入，因 Copilot CLI hook 路徑因版本而異）。

#### Scenario: 輸出 Copilot CLI 設定指引

- **WHEN** `tt setup --copilot` 被呼叫
- **THEN** stdout 輸出包含 Copilot CLI hooks 目錄路徑、事件名稱（`userPromptSubmitted`, `agentStop`）、以及對應的 `tt record` 命令

### Requirement: Copilot CLI Hook 事件欄位對照

系統 SHALL 支援 Copilot CLI hook 呼叫格式，事件欄位對照如下：

| Copilot CLI 事件 | 對應 Claude Code 事件 | 備註 |
|-----------------|----------------------|------|
| `userPromptSubmitted` | `UserPromptSubmit` | 觸發 `tt record prompt` |
| `agentStop` | `Stop` | 觸發 `tt record response` |

#### Scenario: Copilot CLI agentStop 無 token 資料時不報錯

- **WHEN** Copilot CLI `agentStop` hook 觸發，payload 不包含 token 資訊
- **THEN** `tt record response --tokens '{}'` 被呼叫，token 欄位寫入 NULL
- **THEN** exit code 0
