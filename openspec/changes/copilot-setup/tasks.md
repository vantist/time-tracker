## 1. 撰寫測試 (TDD - 測試先行)

- [x] 1.1 在 `internal/setup/setup_test.go` 中新增 `TestSetupCopilot` 系列測試案例（包含首次安裝、重複執行冪等性、不影響使用者自有 hook，以及更新 hook 取代舊版本）。此時執行測試應因尚未實作 `SetupCopilot` 或是實作空殼而編譯失敗或測試失敗。

## 2. 實作核心邏輯

- [x] 2.1 在 `internal/setup/setup.go` 中實作 `SetupCopilot() error`，實作自動將 Copilot CLI hook 合併寫入 `~/.copilot/hooks/tt.json`。
- [x] 2.2 執行並通過 `TestSetupCopilot` 與 `TestSetupClaudeCode` 等 `internal/setup` 的所有測試。

## 3. CLI 整合與驗證

- [x] 3.1 修改 `cmd/tt/setup_cmd.go` 中的 `--copilot` 旗標處理邏輯，將原本列印 `CopilotInstructions` 的行為替換為呼叫 `setup.SetupCopilot()`。
- [ ] 3.2 刪除/清理 `internal/setup/setup.go` 中不再使用的 `CopilotInstructions` 常數。
- [ ] 3.3 執行完整的 `go test ./...` 確保專案內所有測試皆通過。
