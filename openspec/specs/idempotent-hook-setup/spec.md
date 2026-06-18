# idempotent-hook-setup Specification

## Purpose
TBD - created by archiving change setup-hook-dedup. Update Purpose after archive.
## Requirements
### Requirement: SetupClaudeCode 為 idempotent 操作

`tt setup --claude-code` 的 `SetupClaudeCode()` 函數 SHALL 為 idempotent：多次執行的最終結果與執行一次相同。每個由 tt 所有的 hook 條目 SHALL 帶有 `"_owner": "tt"` 標記欄位。merge 邏輯 SHALL 先移除所有 `_owner == "tt"` 的舊條目，再插入新版本，確保不產生重複條目且更新後舊版本被移除。非 tt 所有的 hook 條目 SHALL NOT 受影響。

#### Scenario: 重複執行不產生重複條目

- **WHEN** `SetupClaudeCode()` 在同一個 `settings.json` 上執行兩次
- **THEN** 每個 event（`UserPromptSubmit`、`Stop`）下各自只有一個 tt hook 條目，不出現重複

#### Scenario: 更新 hook 內容後重新 setup 取代舊版本

- **WHEN** `ttHooks` 的 command 字串更新後，`SetupClaudeCode()` 在已含舊版本 hook 的 `settings.json` 上執行
- **THEN** 舊版本 hook 條目被移除，新版本條目被插入，event 下只存在新版本

#### Scenario: 不影響使用者自有 hook

- **WHEN** `settings.json` 的某個 event 已存在 `_owner` 不為 `"tt"` 的 hook 條目，`SetupClaudeCode()` 執行
- **THEN** 該使用者自有條目仍保留在結果中，未被移除或修改

#### Scenario: 首次安裝

- **WHEN** `settings.json` 不存在或 hooks 為空，`SetupClaudeCode()` 執行
- **THEN** `UserPromptSubmit` 與 `Stop` 各新增一個帶 `"_owner": "tt"` 的 hook 條目

