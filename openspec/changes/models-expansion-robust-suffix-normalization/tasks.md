## 1. 撰寫失敗測試 (TDD 階段)

- [x] 1.1 在 `internal/pricing/pricing_test.go` 中新增後綴常態化（如 `gemini-1.5-pro-002`、`claude-3-5-sonnet-latest`、`gpt-4o-preview`）的失敗測試案例
- [x] 1.2 在 `internal/pricing/pricing_test.go` 中新增 2026 最新主流模型（如 `gemini-3.5-flash`、`claude-3-5-sonnet` 等）的費率計算失敗測試案例
- [x] 1.3 執行測試並驗證新增的測試案例皆失敗，確認 TDD 測試防護建立完成

## 2. 核心邏輯與費率表實作

- [x] 2.1 在 `internal/pricing/pricing.go` 中更新 `normalize` 邏輯，實作動態裁切版本與預覽後綴的正規表示式
- [x] 2.2 在 `internal/pricing/pricing.go` 中擴充 `table` 費率表以涵蓋 2026 年最新主流模型與專屬模型定價

## 3. 測試與驗證

- [x] 3.1 執行 `go test ./internal/pricing/...` 確保所有新增與既有測試皆通過
- [x] 3.2 執行 `go build -o tt ./cmd/tt` 確保專案可正常建置
