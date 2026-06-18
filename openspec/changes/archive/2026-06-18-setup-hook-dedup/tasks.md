## 1. 測試先行（TDD）

- [x] 1.1 在 `internal/setup/setup_test.go` 新增「重複執行不產生重複條目」測試：呼叫 `SetupClaudeCode()` 兩次，斷言每個 event 只有一個 tt hook 條目
- [x] 1.2 新增「更新後取代舊版本」測試：先寫入帶舊 command 字串的 hook，再呼叫 `SetupClaudeCode()`，斷言舊條目被移除、新條目存在且只有一個
- [x] 1.3 新增「不影響使用者自有 hook」測試：settings.json 預置不帶 `_owner` 的條目，呼叫後斷言該條目仍存在

## 2. 實作

- [x] 2.1 在 `internal/setup/setup.go` 的 `ttHooks` 每個 outer entry 加 `"_owner": "tt"` 欄位
- [x] 2.2 將 `SetupClaudeCode()` 的 merge 邏輯改為：先 filter 掉 `_owner == "tt"` 的舊條目，再 append 新版本

## 3. 驗證

- [x] 3.1 執行 `go test ./internal/setup/...`，確認所有測試通過
