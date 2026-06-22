## 1. 整合測試基礎腳手架

- [x] 1.1 實作 TestMain，在測試開始前編譯 tt 二進位檔，並在測試結束後清理臨時二進位檔與環境
- [x] 1.2 實作 runTT 輔助函式，封裝 `exec.Command` 並設定隔離的環境變數 `HOME` 與 `TT_DB_PATH`
- [x] 1.3 實作 SQLite 資料庫查詢與驗證的斷言輔助函式，便於直接比對資料表中的 sessions 與 turns 欄位

## 2. 各整合測試案例實作

- [x] 2.1 實作 TestIntegration_GitBranchRepair 測試案例，驗證 `reconcile` 自動修復無 branch 資訊的 session
- [x] 2.2 實作 TestIntegration_ActiveTurnPreemption 測試案例，驗證連續錄製 prompt 時，前一個 turn 被 preempt 關閉
- [x] 2.3 實作 TestIntegration_IdleThresholdReconcile 測試案例，驗證超時 15 分鐘的 dangling turn 會被自動 reconcile 關閉
- [x] 2.4 實作 TestIntegration_FallbackDefaultModel 測試案例，驗證缺省欄位時自動 fallback 當前路徑與預設模型
- [ ] 2.5 實作 TestIntegration_MultiToolIntegration 測試案例，模擬 Claude Code、Copilot CLI、Google Antigravity 的 stdin 與 log 檔案，驗證寫入與 token 解析

## 3. 測試驗證

- [ ] 3.1 執行 `go test -v ./cmd/tt/...` 確保所有新增的整合測試案例皆順利通過
