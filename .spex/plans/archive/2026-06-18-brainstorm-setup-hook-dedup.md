# setup hook dedup

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

`tt setup --claude-code` 寫 hook 到 `~/.claude/settings.json` 時，沒有去重機制：
1. 重複執行 setup 會在同一個 event 下 append 出重複的 hook 條目
2. 更新 hook 內容後重新 setup，舊版本不會被移除

問題來源：`internal/setup/setup.go` 的 merge 邏輯對已存在的 event key 直接 `append`。

## Decision

採用 marker ownership 策略：在每個 tt hook group 加 `"_owner": "tt"` 欄位，merge 時先過濾掉 `_owner == "tt"` 的舊條目，再 append 新版本。

## Rationale

- **B（command string dedup）**：只能防重複安裝，無法清除舊版本 command 字串
- **C（完全取代 event array）**：會清掉使用者在同一 event 的其他 hook，破壞性不可接受
- **A（marker ownership）**：idempotent，支援 command 更新，只動 tt 自己的條目，不影響其他 hook

Claude Code 應忽略 JSON 中不認識的欄位，`_owner` 不會影響功能。

## Approach

1. `ttHooks` 的每個 outer entry 加 `"_owner": "tt"`
2. merge 邏輯改為：先 filter 掉 `_owner == "tt"` 的舊條目，再 append 新版本

```
first install:  []               → [tt-v1]
reinstall:      [tt-v1]          → [tt-v1]      ← filter old, append new
update:         [tt-v1]          → [tt-v2]      ← filter old, append new
other hooks:    [user, tt-v1]    → [user, tt-v2] ← user hooks untouched
```

## Design Notes

只改 `internal/setup/setup.go`，改動約 5–10 行：

```go
// ttHooks - 每個 outer entry 加 _owner
var ttHooks = map[string]interface{}{
    "UserPromptSubmit": []interface{}{
        map[string]interface{}{
            "_owner": "tt",
            "hooks": []interface{}{ ... },
        },
    },
    ...
}

// merge 邏輯
for event, hookVal := range ttHooks {
    newEntries, _ := hookVal.([]interface{})
    existing, _ := hooks[event].([]interface{})

    // filter out tt-owned entries
    filtered := existing[:0]
    for _, e := range existing {
        m, _ := e.(map[string]interface{})
        if m["_owner"] != "tt" {
            filtered = append(filtered, e)
        }
    }
    hooks[event] = append(filtered, newEntries...)
}
```

`setup_test.go` 需補一個 idempotency 測試：連續兩次 SetupClaudeCode()，結果和一次相同。

## Insights to Capture

- `tasks.md`: 補 idempotency test for SetupClaudeCode

## Open Questions

無。
