## 1. Test and Setup for Pricing Table

- [x] 1.1 新增或修改 `internal/pricing/pricing_test.go`，寫入對 `gpt-5.4` 與 `gpt-5-mini` 定價計算的測試案例（TDD 第一步）
- [x] 1.2 在 `internal/pricing/pricing.go` 中，於定價表 table 加入 `gpt-5.4` ($5.00/$15.00/$0.50/$6.25) 與 `gpt-5-mini` ($0.15/$0.60/$0.015/$0.1875) 的定價，使測試通過

## 2. GitHub Copilot CLI log parser

- [x] 2.1 於 `internal/transcript/` 下建立測試檔 `copilot_transcript_test.go`，設計 Copilot CLI 日誌（events.jsonl）解析的單元測試
- [x] 2.2 於 `internal/transcript/` 下建立 `copilot_transcript.go`，實作 `ParseCopilotLog` 函數以讀取 `events.jsonl`，過濾出 `"type": "session.shutdown"` 事件，並提取 `modelMetrics` 的 Token 消耗及模型名稱
- [x] 2.3 執行測試並驗證 Copilot CLI 日誌解析功能

## 3. Antigravity log parser

- [x] 3.1 於 `internal/transcript/` 下建立測試檔 `antigravity_transcript_test.go`，設計 Antigravity 日誌（transcript.jsonl）解析的單元測試
- [x] 3.2 於 `internal/transcript/` 下建立 `antigravity_transcript.go`，實作 `ParseAntigravityLog` 函數以讀取 `transcript.jsonl`，統計主 Agent 的 model usage
- [x] 3.3 執行測試並驗證 Antigravity 日誌解析功能

## 4. Integration with Record Response

- [x] 4.1 在 `cmd/tt/record.go` 或相關 recorder 模組中，新增測試案例（TDD），模擬 `tool == "copilot-cli"` 與 `tool == "antigravity"` 時的 record response 行為
- [x] 4.2 修改 `cmd/tt/record.go` 中的 `record response` 指令執行邏輯，使其根據傳入的 `tool` 分流：讀取對應的日誌檔案並調用新實作的 parser，提取數據存入資料庫
- [x] 4.3 進行整合測試，確認 hook 失敗時能靜默處理（exit 0），且能正確補寫 `sessions.model` 與更新 `turns.estimated_cost_usd` 等資料庫欄位
