## MODIFIED Requirements

### Requirement: 記錄 prompt 事件

系統 SHALL 透過 `tt record prompt` 子命令接收 hook 呼叫，並在 SQLite 中建立或更新 session 紀錄，同時寫入 turn 的 `prompt_at` 時間戳。

命令簽章：
```
tt record prompt --session <id> --project <path> --tool <tool> --model <model>
```

- `--session`：hook 提供的 session ID（字串）
- `--project`：git root 路徑，若非 git repo 則為 cwd
- `--tool`：`"claude-code"`、`"copilot-cli"`、`"vscode-copilot"` 或 `"antigravity"`
- `--model`：模型名稱字串，未知時允許任意值

#### Scenario: 首次 prompt 建立 session 與 turn

- **WHEN** `tt record prompt --session abc123 --project /home/user/myproject --tool claude-code --model claude-sonnet-4-6` 被呼叫
- **THEN** `sessions` 表中 `id = "abc123"` 的 session 不存在時，建立新 session（`project = "/home/user/myproject"`, `tool = "claude-code"`, `started_at = 目前 unix ms`）
- **THEN** `turns` 表中插入一筆新 turn（`session_id = "abc123"`, `model = "claude-sonnet-4-6"`, `prompt_at = 目前 unix ms`）
- **THEN** 命令回傳 exit code 0，無輸出到 stdout

#### Scenario: 同 session 第二次 prompt 不重建 session

- **WHEN** `session_id = "abc123"` 的 session 已存在，再次呼叫 `tt record prompt --session abc123 ...`
- **THEN** `sessions` 表中 `id = "abc123"` 的 session 不被重複建立（upsert 不覆蓋 `started_at`）
- **THEN** `turns` 表插入新的 turn 紀錄（同一 session 下的第二個 turn）

#### Scenario: git branch 自動偵測並存入 session

- **WHEN** `--project` 指向的路徑是 git repo，且 `git branch --show-current` 回傳非空字串
- **THEN** `sessions.branch` 存入該 branch 名稱

#### Scenario: 非 git repo 時 branch 為 NULL

- **WHEN** `--project` 指向的路徑不是 git repo，或 `git branch --show-current` 回傳空字串
- **THEN** `sessions.branch` 寫入 NULL，不報錯

#### Scenario: vscode-copilot tool 記錄 prompt

- **WHEN** `tt record prompt --session vscode-session-123 --project /home/user/myproject --tool vscode-copilot --model gpt-5-codex` 被呼叫
- **THEN** `sessions` 表中建立新 session（`tool = "vscode-copilot"`, `started_at = 目前 unix ms`）
- **THEN** `turns` 表中插入新 turn（`model = "gpt-5-codex"`, `prompt_at = 目前 unix ms`）
