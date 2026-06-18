# AI 工具時間記錄工具（tt）

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). -->

## Context

記錄與 AI 工具互動的時間與成本。主要場景：Claude Code、Copilot CLI。
兩個工具都有類似的 hook 系統，可以自動計時，不需要手動操作。

## Decision

建置一個 Go CLI binary `tt`，透過 hook 系統自動記錄事件，資料存本機 SQLite，提供報表查詢介面。

## Rationale

- Hook 是唯一可靠的計時切入點，不需要 plugin 或 daemon
- Agent 時間可精確測量（UserPromptSubmit → Stop），User 時間用 idle threshold 近似
- Session 總時長不可信（開著不等於在用），所以改用 idle timeout 切段
- SQLite 查詢有力、單一檔案、可擴展，優於 JSONL

## Approach

`tt` binary 兩個職責：
1. **Hook writer**：被 Claude Code / Copilot CLI 呼叫，寫入 SQLite
2. **報表 CLI**：查詢聚合結果

Claude Code 設定 `~/.claude/settings.json`，Copilot CLI 設定 `~/.copilot/hooks/`。

## Design Notes

### 時間計算模型

```
prompt_sent  ──────────────────── agent_done
                 agent_time        │
                                   │ user_gap
                                   │
                            prompt_sent (next)
```

- **Agent 時間** = `response_at - prompt_at`（精確）
- **User 主動時間** = Σ(gaps < idle_threshold)（近似，預設 15 分鐘，可設定）
- Session 總時長不單獨計算，只用作分組

### Data Model（SQLite）

```sql
sessions (
  id            TEXT PRIMARY KEY,  -- tool session_id
  project       TEXT,              -- git root，fallback cwd
  tool          TEXT,              -- "claude-code" | "copilot-cli"
  started_at    INTEGER,           -- unix ms
  ended_at      INTEGER,
  branch        TEXT,              -- git branch --show-current，自動偵測
  work_item     TEXT               -- tt work 手動覆蓋，報表優先於 branch
)

turns (
  id                      INTEGER PRIMARY KEY,
  session_id              TEXT REFERENCES sessions,
  model                   TEXT,
  prompt_at               INTEGER,
  response_at             INTEGER,
  input_tokens            INTEGER,
  output_tokens           INTEGER,
  cache_read_tokens       INTEGER,
  cache_creation_tokens   INTEGER,
  estimated_cost_usd      REAL
)
```

聚合邏輯在查詢層，原始事件永久保留。

### Hook 對照

| 事件 | Claude Code | Copilot CLI |
|------|-------------|-------------|
| Prompt 送出 | `UserPromptSubmit` | `userPromptSubmitted` |
| Agent 結束 | `Stop` | `agentStop` |
| Tool 前後 | `PreToolUse`/`PostToolUse` | `preToolUse`/`postToolUse` |
| Session | 推算 | `sessionStart`/`sessionEnd` |

### CLI 介面

```bash
# Hook 呼叫（不對使用者）
tt record prompt    --session <id> --project <path> --tool <tool> --model <model>
tt record response  --session <id> --tokens <json>

# 查詢
tt report
tt report --project time-tracker
tt report --since 7d
tt report --since 2026-06-01
tt report --format json

# 工作項目
tt work "login redesign"   # 手動標記，覆蓋 branch 自動偵測
tt work                    # 顯示目前追蹤的工作項目

# 設定
tt config set idle-threshold 20   # 分鐘
```

### 報表範例輸出

```
Project: time-tracker  (last 7d)
  Sessions:     12
  Agent time:   2h 34m
  User active:  1h 12m  (idle threshold: 15m)
  Tokens in:    142,300  (cache hit: 89,200 = 63%)
  Tokens out:   24,100
  Est. cost:    $0.87

tt report --by-work-item

  login-redesign     3h 12m  agent: 1h 40m  $0.52
  auth-token-expiry  0h 45m  agent: 0h 28m  $0.18
  untagged           0h 20m  agent: 0h 10m  $0.05
```

報表 work item 顯示邏輯：`work_item ?? branch ?? "untagged"`

### 技術選型

- 語言：**Go**（啟動快、單一 binary、SQLite 支援好）
- SQLite driver：`modernc.org/sqlite`（pure Go，不需 cgo）
- CLI framework：`cobra`
- 發布：`goreleaser` + GitHub Releases + brew tap

### Token 資料

- Claude Code：`Stop` hook payload 含 usage stats（需實作時確認 payload 結構）
- Copilot CLI：`agentStop` payload 待確認；Chronicle 有記錄 token，可作 fallback
- 定價表 hard-code 在 binary，按 `model` 欄位查詢

### Copilot CLI Token Fallback

若 hook payload 無 token 資料，parse `~/.copilot/` 的 session 記錄（Chronicle 存檔）。

## Insights to Capture

- `design.md`: idle threshold 設計決策（session 開著不等於在使用）
- `tasks.md`: 確認 Stop/agentStop payload 的 token 欄位格式
- `tasks.md`: 實作 `tt record prompt` / `tt record response`
- `tasks.md`: Claude Code hooks 設定
- `tasks.md`: Copilot CLI hooks 設定
- `tasks.md`: 報表聚合查詢

## Open Questions

- `Stop` hook payload 確切欄位名稱（input_tokens? usage.input_tokens?）
- Copilot CLI `agentStop` payload 是否帶 token 資料
- 定價表更新策略（binary 更新 vs 設定檔）
