## 1. 歷史 Session 自動修補

- [x] 1.1 在 `internal/reconcile/reconcile_test.go` 中寫入對 `repairSessions` 的測試案例（包含缺失 `project` 或 `model` 欄位，從 transcript 檔案路徑解析，並遞迴尋找專案根目錄的預期行為）。測試目前應失敗。
- [x] 1.2 在 `internal/reconcile/reconcile.go` 中實作 `repairSessions(db *sql.DB)` 函數（解析 transcript_full.jsonl、過濾系統路徑、遞迴尋找專案根目錄、更新 session 欄位）。
- [x] 1.3 執行測試並驗證 `repairSessions` 單元測試通過。
- [x] 1.4 在 `internal/reconcile/reconcile.go` 中修改 `reconcile` 流程，使其在最開始呼叫 `repairSessions`，並更新相關整合測試以驗證歷史 Session 自動修補整合成功。

## 2. 進程存活超時自動關閉 (idle-threshold)

- [ ] 2.1 在 `internal/reconcile/reconcile_test.go` 中寫入對 dangling turn 逾期強制 reconcile 的測試案例。測試目前應失敗。
- [ ] 2.2 在 `internal/reconcile/reconcile.go` 中新增 `idle-threshold` 超時判斷邏輯（15 分鐘），並在 `reconcileTurn` 中修改對已逾期的進行中 turn 的 skip 判斷。
- [ ] 2.3 執行測試並驗證超時關閉測試通過。

## 3. RecordPrompt 自動搶佔逾期懸空 Turn

- [ ] 3.1 在 `internal/recorder/recorder_test.go` 中寫入對 `RecordPrompt` 在遇到超過 15 分鐘 active turn 時強制更新/搶佔的測試案例。測試目前應失敗。
- [ ] 3.2 在 `internal/recorder/recorder.go` 中實作 `RecordPrompt` 針對 `antigravity` 工具的逾時 active turn 判斷，並執行 SQL `UPDATE` 自動關閉該懸空 turn。
- [ ] 3.3 執行測試並驗證 `RecordPrompt` 搶佔測試通過。

## 4. 整體驗證

- [ ] 4.1 執行專案所有單元與整合測試 `go test ./...` 驗證皆通過。
