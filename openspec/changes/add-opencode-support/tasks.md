## 1. Transcript provider registration

- [x] 1.1 新增 `internal/transcript/opencode.go`：為 `"opencode"` 註冊空實作 LogProvider（不解析 token，僅滿足 provider 註冊表完整性）。TDD：先寫失敗測試驗證 provider lookup 回傳 opencode provider 且不執行任何 token 提取。

## 2. record.go flag handling

- [x] 2.1 `cmd/tt/record.go` `resolvePromptInput`：當 `--tool opencode` 時不要求 `--transcript-path`，`prompt_line_offset` 寫入 NULL。TDD：先寫失敗測試（opencode prompt 不帶 transcript-path 不報錯且 offset 為 NULL）。
- [x] 2.2 `cmd/tt/record.go` `resolveResponseInput`：當 `--tool opencode` 時直接採用 `--tokens` flag 值，跳過 `ExtractWindow` / `ExtractLastTurn` fallback。TDD：先寫失敗測試驗證不讀取 transcript 檔案。
- [x] 2.3 `cmd/tt/record.go`：為 `tt record response` 新增 `--subagent-tokens` flag 並解析進 input 結構。TDD：先寫失敗測試驗證 flag 被解析且未提供時為空。

## 3. recorder subagent-tokens 持久化

- [ ] 3.1 `internal/recorder/response.go`：解析 `--subagent-tokens` JSON 陣列為 `ModelUsage` list（`is_subagent = 1`，容錯缺欄位寫 NULL）。TDD：先寫失敗測試涵蓋單一/多個 subagent、缺欄位情境。
- [x] 3.2 `internal/recorder/response.go`：將 subagent usages INSERT 進 `turn_model_usages`（依各 element model 查定價算 `estimated_cost_usd`，`turn_id` 關聯本 turn）；當 `--subagent-tokens` 已提供時 SHALL NOT 執行 transcript subagent 掃描。TDD：先寫失敗測試驗證 DB 明細與「不掃描 transcript」行為。

## 4. opencode setup

- [x] 4.1 `internal/setup/opencode.go`：實作 `SetupOpencode`——產生 `~/.config/opencode/plugins/tt-bridge.ts`，冪等（已存在則不覆蓋並輸出提示，不存在則建父目錄後寫入並輸出 `OpenCode plugin configured in ~/.config/opencode/plugins/tt-bridge.ts`）。TDD：先寫失敗測試涵蓋首次產生與已存在跳過。
- [x] 4.2 `cmd/tt/setup_cmd.go`：新增 `--opencode` flag 並接線至 `SetupOpencode`。TDD：先寫失敗測試驗證 flag 觸發 setup。
- [x] 4.3 `internal/setup/setup.go`：將 opencode 納入多工具 setup 分派、`~/.config/opencode` 自動偵測、以及 no-tool-warning 檢查清單。TDD：先寫失敗測試涵蓋自動偵測 opencode 與無工具時提示。

## 5. report 工具名正規化

- [x] 5.1 `cmd/tt/report_cmd.go` `normalizeAgentName`：新增 `"opencode"` → `"OpenCode"` 分支（置於其餘名稱 fallback 之前）。TDD：先寫失敗測試驗證 `opencode` / `OpenCode` 正規化為 `OpenCode`。

## 6. plugin template 與整合

- [x] 6.1 撰寫 `tt-bridge.ts` plugin 範本內容：`message.updated` event → `tt record prompt/response --tool opencode` flag 對應、subagent token 暫存（`pendingSubTokens`）與隨主 response flush、串流中間態防護（`time.completed`/`finish` 條件）、`dedupKey` 去重。
- [ ] 6.2 整合測試：`tt setup --opencode` 產生合法 TS 檔，並以模擬的 `tt record prompt` + `tt record response --tokens --subagent-tokens` 呼叫驗證 `sessions` / `turns` / `turn_model_usages` 寫入正確（含 `is_subagent = 1` 明細）。

## 7. 開放問題驗證與建置

- [ ] 7.1 實測開放問題 1（user message 是否帶 model）與開放問題 2（subagent 相對於主 response 的事件順序） against 真實 opencode；記錄結論，若 subagent 晚於主 response 完成則評估是否另開 change 補償。
- [ ] 7.2 `go build ./...` 與 `go test ./...` 全綠。
