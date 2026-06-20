## 1. 實作 Git 分支解析輔助函式

- [x] 1.1 於 `internal/reconcile/reconcile.go` 引入 `"os/exec"` 與 `"strings"` 套件
- [x] 1.2 於 `internal/reconcile/reconcile.go` 實作本地私有函式 `gitBranch(dir string) string`，執行 `git -C <dir> rev-parse --abbrev-ref HEAD` 並回傳清理後的結果，失敗時回傳空字串

## 2. 擴充 `repairSessions` 修補邏輯

- [ ] 2.1 修改 `repairSessions` 中的 SQL 查詢語句，加入 `branch IS NULL OR branch = ''` 條件，並在 `SELECT` 欄位中包含 `branch`
- [ ] 2.2 更新 `sessInfo` 結構以支援 `branch` 欄位
- [ ] 2.3 修改修補迴圈，當發現 `branch` 為空時，若專案路徑不為空，呼叫 `gitBranch` 取得分支。若取得失敗（或非 Git 專案），填入 `"-"` 作為佔位符
- [ ] 2.4 更新 `updateInfo` 結構以支援 `branch` 欄位
- [ ] 2.5 修改 `repairSessions` 中的 `UPDATE` 語句，將 `branch` 更新至 `sessions` 表中

## 3. 新增測試案例與驗證

- [ ] 3.1 於 `internal/reconcile/reconcile_test.go` 引入 `"os/exec"` 套件
- [ ] 3.2 於 `internal/reconcile/reconcile_test.go` 新增單元測試 `TestRepairSessions_Branch`，測試在 Git 專案下能修復正確的分支名稱
- [ ] 3.3 於 `internal/reconcile/reconcile_test.go` 的單元測試 `TestRepairSessions_Branch` 中，測試在非 Git 專案下能將分支修復為 `"-"`
