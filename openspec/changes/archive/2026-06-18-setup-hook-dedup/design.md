## Context

`internal/setup/setup.go` 的 `SetupClaudeCode()` 對 `~/.claude/settings.json` 進行 hook merge，目前邏輯對已存在的 event key 直接 append，導致重複與殘留問題。

## Goals / Non-Goals

**Goals:**
- `SetupClaudeCode()` 變為 idempotent
- hook 更新後重新 setup 移除舊版本

**Non-Goals:**
- 支援多個 tt owner（`_owner` 值固定為 `"tt"`）
- 處理 Copilot 或其他工具的 hook

## Decisions

**Marker ownership**：每個 tt hook outer entry 加 `"_owner": "tt"`。Merge 時先 filter 掉 `_owner == "tt"` 的舊條目，再 append 新版本。

理由：command string dedup 無法解決更新後舊版本殘留；完全取代 event array 會清除使用者自有 hook。

## Risks / Trade-offs

- Claude Code 忽略 JSON 未知欄位是預期行為，目前版本已確認如此；未來版本若嚴格驗證 schema 需移除 `_owner` 改用其他機制。
