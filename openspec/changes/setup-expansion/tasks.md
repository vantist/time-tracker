## 1. Helper 重構與測試 (TDD)

- [x] 1.1 在 `internal/setup/setup_test.go` 中為即將抽取出的 `mergeHooksFile` Helper 撰寫測試用例（包括目錄不存在建立、舊 `_owner == "tt"` 項目清理、合併新項目、0o600 權限等行為驗證）
- [x] 1.2 在 `internal/setup/setup.go` 中重構並實作 `mergeHooksFile` Helper，使上述測試通過
- [x] 1.3 使用 `mergeHooksFile` 重構 `SetupClaudeCode()` 函式，並確保現有 Claude Code 測試依然通過

## 2. Antigravity 及 Codex 整合 (TDD)

- [x] 2.1 在 `internal/setup/setup_test.go` 中為 `SetupAntigravity()` 與 `SetupCodex()` 撰寫單元測試用例，驗證其寫入對應的 hooks 結構與冪等性
- [x] 2.2 在 `internal/setup/setup.go` 中實作 `SetupAntigravity()` 與 `SetupCodex()`，呼叫 `mergeHooksFile` 實現相應邏輯，並使測試通過
- [x] 2.3 在 `cmd/tt/setup_cmd.go` 中加入 `--antigravity` 與 `--codex` flag 參數與對接，並為 CLI 參數撰寫/執行整合測試

## 3. Record Stdin Payload 解析支援 (TDD)

- [x] 3.1 在 `cmd/tt/record_test.go` 中為 stdin JSON 包含 `conversationId` 與 `transcriptPath` 的解析對照與邏輯增加單元測試
- [x] 3.2 在 `cmd/tt/record.go` 的 `hookPayload` 中加入對應欄位，並在 `readStdinJSON()` 中進行正規化，確保測試通過
- [ ] 3.3 更新 `design.md` 中的 Hook 整合與 Payload 欄位規格說明
