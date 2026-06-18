## 1. 擴展資料結構

- [x] 1.1 在 `transcriptEntry` struct 新增 `Content []contentBlock` 欄位（`json:"content"`），`contentBlock` 包含 `Type string`、`ID string`、`Name string`
- [x] 1.2 新增 `subagentMeta` struct（`json:"toolUseId"`、`json:"agentType"`、`json:"description"`）用於反序列化 meta.json

## 2. 撰寫失敗測試（TDD）

- [x] 2.1 在 `cmd/tt/record_test.go` 新增 `TestExtractSubagentTokens`，建立 fixture 目錄（subagents dir、meta.json、agent jsonl），驗證：無目錄回傳零值、單一 subagent 合計正確、多 subagent 合計、toolUseId 不匹配時不計算、jsonl 不存在時略過
- [x] 2.2 在 `cmd/tt/record_test.go` 新增 `TestExtractFromTranscriptAtOffset_WithSubagents`，驗證最終 tokensJSON 包含 subagent token

## 3. 實作 extractSubagentTokens

- [x] 3.1 實作 `extractSubagentTokens(transcriptPath string, entries []transcriptEntry, offset int) transcriptUsageFields`：
  - 從 `entries[offset:]` 掃 `type == "assistant"` 的 content block，找 `type == "tool_use" && name == "Agent"` 的 ID，收集成 Set
  - 推導 subagentsDir：`filepath.Join(strings.TrimSuffix(transcriptPath, ".jsonl"), "subagents")`
  - 讀 `*.meta.json`，比對 `toolUseId` 是否在 Set 中
  - 對每個匹配的 `.jsonl` 呼叫 `sumWindow(loadTranscript(jsonlPath), 0, n)`
  - 合計所有 subagent 的 usage 回傳
- [x] 3.2 確認所有步驟都 graceful 處理檔案不存在、目錄不存在的情況（靜默略過）

## 4. 整合進 extractFromTranscriptAtOffset

- [x] 4.1 在 `extractFromTranscriptAtOffset` 的 `acc := sumWindow(...)` 之後，呼叫 `extractSubagentTokens(path, all, offset)`，將結果加入 `acc`
- [x] 4.2 確認 `extractFromTranscript`（無 offset 的 fallback）不受影響

## 5. 驗證

- [x] 5.1 執行 `go test ./cmd/tt/...` 確認所有測試通過
- [x] 5.2 手動驗證：執行一次有 subagent 的 Claude Code session，`tt record response` 後檢查 DB 中的 token 數字是否包含 subagent token
