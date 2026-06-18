## 1. workitem package — 失敗測試先行（TDD）

- [x] 1.1 在 `internal/workitem/workitem_test.go` 新增測試：`TestGetSetClearPerProject`，驗證兩個不同 project key 的 Get/Set/Clear 互相獨立（此時測試應失敗，因為 API 尚未更改）
- [x] 1.2 在 `internal/workitem/workitem_test.go` 新增測試：`TestResolveProjectGitRoot`，驗證在 git repo 子目錄時 resolve 結果等於 git root（可 mock exec 或使用臨時 git repo）
- [x] 1.3 在 `internal/workitem/workitem_test.go` 新增測試：`TestResolveProjectNonGit`，驗證非 git 目錄時 resolve 結果等於傳入的 dir

## 2. workitem package — 實作

- [x] 2.1 在 `internal/workitem/workitem.go` 新增 `resolveProject(dir string) string`：執行 `git -C dir rev-parse --show-toplevel`，失敗則回傳 `dir`
- [x] 2.2 新增 `projectKey(project string) string`：對 `resolveProject(project)` 結果取 SHA-256，回傳前 16 hex 字元
- [x] 2.3 新增 `workItemPath(project string) (string, error)`：回傳 `~/.tt/work-items/<hash>` 完整路徑，並 `os.MkdirAll` 確保目錄存在
- [x] 2.4 將 `Get` / `Set` / `Clear` 簽名改為 `(project string)` 版本，內部使用 `workItemPath(project)` 取代原本的 `~/.tt/work-item`
- [x] 2.5 確認 1.1–1.3 所有測試通過

## 3. cmd/tt/work.go — 傳入 CWD

- [ ] 3.1 在 `cmd/tt/work.go` 的 `get` / `set` / `clear` handler 中，以 `os.Getwd()` 取得 CWD 並傳入對應的 `workitem.Get` / `Set` / `Clear`

## 4. recorder — 傳入 input.Project

- [ ] 4.1 在 `internal/recorder/recorder.go` 的 `RecordPrompt` 中，將 `workitem.Get()` 呼叫改為 `workitem.Get(input.Project)`

## 5. 驗證

- [ ] 5.1 執行 `go test ./internal/workitem/...` 全部通過
- [ ] 5.2 執行 `go build ./...` 無編譯錯誤
- [ ] 5.3 手動驗證：在兩個不同 git repo 分別 `tt work set`，確認切換 repo 後各自顯示對應 work item
