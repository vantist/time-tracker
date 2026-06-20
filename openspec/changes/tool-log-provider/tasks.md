## 1. 修正 Copilot CLI prompt 錄製資料

- [x] 1.1 撰寫單元測試驗證 Copilot CLI prompt 錄製時，`transcriptPath` 與 `cwd` 能被寫入資料庫
- [x] 1.2 修改 `cmd/tt/record.go` 中的 stdin JSON 解析，確保正確儲存 Copilot CLI 的 `transcriptPath` 與 `cwd`

## 2. 定義 LogProvider 介面與 Registry

- [x] 2.1 建立 `internal/transcript/provider.go`，定義 `LogProvider` 介面及 `Register`/`GetProvider` 機制
- [x] 2.2 撰寫測試驗證 Provider Registry 能正確註冊與取得對應的 Provider

## 3. 實作 JSONL 共享 Provider (JSONLProvider)

- [x] 3.1 撰寫單元測試以模擬 JSONL 日誌測試 `JSONLProvider` 的 `ExtractLastTurn`、`ExtractWindow` 與 subagent 遞迴解析
- [x] 3.2 重構並封裝 `internal/transcript/extract.go` 的 JSONL 解析邏輯至 `JSONLProvider`，使 Claude, Antigravity, Codex 共享實作

## 4. 實作 Copilot CLI Provider (CopilotProvider)

- [x] 4.1 撰寫單元測試，模擬 `events.jsonl` 日誌驗證 `CopilotProvider` 能夠正確擷取主模型及子代理模型 (非 `mainModel`) 的 token 消耗
- [x] 4.2 實作 `CopilotProvider` 解析 `events.jsonl` 中的 `session.shutdown` 事件，並支援 `ExtractLastTurn` 與 `ExtractWindow` 介面

## 5. 整合與驗證

- [ ] 5.1 修改 `cmd/tt/record.go` 中的 stop 錄製邏輯，改為透過 `GetProvider(tool)` 獲取 Provider 擷取 token
- [ ] 5.2 修改 `internal/reconcile/reconcile.go`，使用 `GetProvider(tool)` 重構 `reconcileTurn` 中的日誌擷取邏輯，確保所有工具的增量窗口比對邏輯一致
- [ ] 5.3 執行所有單元測試與整合測試 (`go test ./...`)，確保無 regression
