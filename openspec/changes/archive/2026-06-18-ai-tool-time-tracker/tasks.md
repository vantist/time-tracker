## 1. 專案初始化

- [x] 1.1 確認 Claude Code `Stop` hook 的 token payload 確切欄位名稱（查閱 Claude Code hook 文件或測試 hook），記錄至 `design.md` Open Questions
- [x] 1.2 確認 Copilot CLI `agentStop` hook payload 格式，記錄欄位名稱或 Chronicle fallback 策略
- [x] 1.3 建立 `go.mod`（module `github.com/user/tt`），加入 `cobra`、`modernc.org/sqlite` 依賴，執行 `go mod tidy`
- [x] 1.4 建立 `cmd/tt/main.go` — cobra root command 骨架，`--help` 可正常輸出

## 2. 資料庫 Schema

- [x] 2.1 **[TDD]** 寫 `internal/db/schema_test.go`：測試首次執行自動建立 `sessions` 和 `turns` 表，使用 `TT_DB_PATH` 環境變數指向 temp 檔
- [x] 2.2 實作 `internal/db/schema.go`：建立 `sessions`、`turns` 表（`CREATE TABLE IF NOT EXISTS`），讀取 `TT_DB_PATH` 環境變數覆蓋預設路徑 `~/.tt/data.db`，通過 2.1 測試
- [x] 2.3 **[TDD]** 寫測試：session upsert 保留 `started_at`（INSERT OR IGNORE 語義）
- [x] 2.4 實作 session upsert 邏輯，通過 2.3 測試

## 3. 事件記錄（`tt record`）

- [x] 3.1 **[TDD]** 寫 `internal/recorder/recorder_test.go`：測試 `RecordPrompt` — 首次建立 session + turn，`prompt_at` 正確，git branch 自動偵測（mock git 或使用 fixture）
- [x] 3.2 實作 `internal/recorder/recorder.go`：`RecordPrompt(sessionID, project, tool, model string)`，含 git branch 偵測，通過 3.1 測試
- [x] 3.3 **[TDD]** 寫測試：`RecordPrompt` 同 session 第二次不重建 session
- [x] 3.4 通過 3.3 測試
- [x] 3.5 **[TDD]** 寫 `internal/recorder/recorder_test.go`：測試 `RecordResponse` — 更新最新 turn 的 `response_at`、token 欄位、`estimated_cost_usd`；token JSON 扁平格式與巢狀格式兩種
- [x] 3.6 實作 `RecordResponse(sessionID string, tokensJSON string)`，解析 token JSON（含容錯），呼叫定價計算，通過 3.5 測試
- [x] 3.7 **[TDD]** 寫測試：hook 呼叫失敗（SQLite 鎖定模擬）時 `RecordPrompt`/`RecordResponse` 回傳 nil error（不 panic）
- [x] 3.8 實作錯誤靜默處理（stderr 輸出，exit code 0），通過 3.7 測試
- [x] 3.9 在 cobra 中加入 `tt record prompt` 和 `tt record response` 子命令，接受所有 flag，呼叫 recorder

## 4. 定價計算（`internal/pricing`）

- [x] 4.1 **[TDD]** 寫 `internal/pricing/pricing_test.go`：測試 `claude-sonnet-4-6` 的 `estimated_cost_usd` 計算（參照 spec cost-estimation Scenario 1 的具體數值）
- [x] 4.2 實作 `internal/pricing/pricing.go`：hard-code 定價表（含 haiku、sonnet、opus），`Calculate(model, inputTokens, outputTokens, cacheRead, cacheCreation int) *float64`，通過 4.1 測試
- [x] 4.3 **[TDD]** 寫測試：未知 model 回傳 `nil`（NULL）
- [x] 4.4 通過 4.3 測試

## 5. 工作項目標記（`tt work`）

- [x] 5.1 **[TDD]** 寫 `internal/workitem/workitem_test.go`：測試 `Set("login-redesign")` 寫入檔案；`Get()` 讀取；`Clear()` 刪除；Get 找不到檔案時回傳空字串
- [x] 5.2 實作 `internal/workitem/workitem.go`，通過 5.1 測試
- [x] 5.3 在 cobra 中加入 `tt work` 子命令（`tt work "<label>"`, `tt work`, `tt work --clear`）
- [x] 5.4 在 `RecordPrompt` 中加入讀取 `~/.tt/work-item` 並寫入 `sessions.work_item` 的邏輯（session 尚未設定 work_item 時才寫入）

## 6. 時間聚合與報表（`tt report`）

- [x] 6.1 **[TDD]** 寫 `internal/aggregator/aggregator_test.go`：測試 agent 時間計算（3 turns，第 3 個 response_at 為 NULL，預期 45 秒）
- [x] 6.2 實作 `internal/aggregator/aggregator.go`：`AgentTime(turns []Turn) time.Duration`，通過 6.1 測試
- [x] 6.3 **[TDD]** 寫測試：user 主動時間計算 — gap < threshold 計入，gap ≥ threshold 不計入，idle_threshold 可設定
- [x] 6.4 實作 `UserActiveTime(turns []Turn, idleThreshold time.Duration) time.Duration`，通過 6.3 測試
- [x] 6.5 **[TDD]** 寫 `internal/report/report_test.go`：測試 `--since 7d` 篩選（turns 在範圍內/外）；`--project` 篩選；無資料時輸出 "No data for the selected period."
- [x] 6.6 實作 `internal/report/report.go`：查詢 SQLite，組裝聚合結果，通過 6.5 測試
- [x] 6.7 實作 text 格式輸出（含 Sessions, Agent time, User active, Tokens in, Est. cost 欄位）
- [x] 6.8 實作 `--format json` 輸出（含所有必要欄位，可被 `jq` 解析）
- [x] 6.9 實作 `--by-work-item` 分組報表（`work_item ?? branch ?? "untagged"` 優先順序）
- [x] 6.10 在 cobra 中加入 `tt report` 子命令，接受 `--project`, `--since`, `--format`, `--by-work-item` flags

## 7. 設定管理（`tt config`）

- [x] 7.1 **[TDD]** 寫 `internal/config/config_test.go`：測試 `Set("idle-threshold", "30")` 寫入；`Get("idle-threshold")` 讀取；未設定時回傳預設值 15
- [x] 7.2 實作 `internal/config/config.go`（設定存於 `~/.tt/config.json`），通過 7.1 測試
- [x] 7.3 在 cobra 中加入 `tt config set <key> <value>` 子命令
- [x] 7.4 在報表聚合時讀取 `idle-threshold` 設定（若未設定使用預設 15 分鐘）

## 8. Hook 整合設定（`tt setup`）

- [x] 8.1 **[TDD]** 寫 `internal/setup/setup_test.go`：測試 `SetupClaudeCode` — 首次設定正確寫入 hooks；已有其他 hooks 時不覆蓋
- [x] 8.2 實作 `internal/setup/setup.go`：`SetupClaudeCode()` merge hooks 到 `~/.claude/settings.json`，通過 8.1 測試
- [x] 8.3 在 cobra 中加入 `tt setup --claude-code` 子命令
- [x] 8.4 實作 `tt setup --copilot` — 輸出 Copilot CLI 設定指引（含事件名稱、hook 路徑、命令範例）

## 10. Transcript-anchored token capture

- [x] 10.1 Migration：`turns` 表加入 `transcript_path TEXT` 與 `prompt_line_offset INTEGER` 欄位（`addTurnColumns` 比照 `addSessionColumns` 模式）
- [x] 10.2 `cmd/tt/record.go`：`hookPayload` 加入 `TranscriptPath` 欄位；`resolvePromptInput` 把 `transcript_path` 傳進 `PromptInput`；`recordPromptCmd` 加入 `--transcript-path` flag（可選，覆蓋 stdin 值）
- [x] 10.3 `internal/recorder/recorder.go`：`PromptInput` 加入 `TranscriptPath string`；`RecordPrompt` 計算當下行數並寫入 `transcript_path`, `prompt_line_offset`；檔案不存在時 offset 寫 0
- [x] 10.4 **[TDD]** 寫測試：`RecordPrompt` 傳入 transcript path → `prompt_line_offset` 等於測試 JSONL 行數；transcript 不存在 → `prompt_line_offset = 0`
- [x] 10.5 `cmd/tt/record.go`：實作 `extractFromTranscriptAtOffset(path string, offset int)` — 跳過前 offset 行，只對第 offset 行之後的 assistant entries 做 dedup + sum；model 仍從整份 transcript 最後一個 non-sidechain assistant entry 取得
- [x] 10.6 **[TDD]** 寫測試：`extractFromTranscriptAtOffset` — offset 正確切割；offset = 0 時行為與全文掃描相同；offset > 行數時回傳空
- [x] 10.7 `internal/recorder/response.go`：`RecordResponse` 從最新 turn SELECT `transcript_path`, `prompt_line_offset`；若 `prompt_line_offset NOT NULL` 則呼叫 `extractFromTranscriptAtOffset`；若 NULL 則 fallback 原有 `extractFromTranscript` 邏輯
- [x] 10.8 手動驗證：在含有 `/clear` 後立即下指令的 session 確認 token 欄位正確填入（不為 NULL）

## 11. Subagent Token Capture

- [x] 11.1 實作 `extractSubagentTokens(transcriptPath string, offset int) usageFields`：掃描 transcript 第 `[offset:]` 行，收集 `content[].type == "tool_use" && name == "Agent"` 的 toolUseId 清單
- [x] 11.2 `extractSubagentTokens` 推導 subagentsDir：transcript `~/.claude/projects/<proj>/<session>.jsonl` → `~/.claude/projects/<proj>/<session>/subagents/`
- [x] 11.3 `extractSubagentTokens` 讀取所有 `*.meta.json`，比對 toolUseId → 取得對應 `.jsonl` 路徑
- [x] 11.4 `extractSubagentTokens` 對每個 subagent `.jsonl`：dedup + sum assistant entries（忽略 `isSidechain` 欄位，全部讀取），回傳合計 `usageFields`
- [x] 11.5 **[TDD]** 寫測試：`extractSubagentTokens` — 給定主 transcript 含一個 `Agent` tool_use，subagents/ 目錄有對應 meta.json + jsonl，驗證回傳正確合計 token；subagents 目錄不存在時回傳零值 `usageFields`
- [x] 11.6 `cmd/tt/record.go`：`extractFromTranscriptAtOffset` 執行後呼叫 `extractSubagentTokens(path, offset)`，將結果合計至 main transcript 提取的 `usageFields`；`extractFromTranscript`（無 offset fallback）不修改
- [x] 11.7 手動驗證：執行含 `/code-review` 或多 subagent 的 session，確認 `turns` 表 `tokens_input` + `tokens_output` 包含 subagent 消耗量（對照 Claude Code UI 顯示的 token 數）

## 9. End-to-End 驗證

- [x] 9.1 手動測試：在本機設定 Claude Code hooks，驗證 `UserPromptSubmit` → `tt record prompt` 呼叫正確（查 SQLite 確認）
- [x] 9.2 手動測試：`Stop` hook → `tt record response` 正確更新 token 欄位
- [x] 9.3 手動測試：`tt report` 輸出正確的時間與成本
- [x] 9.4 手動測試：`tt work "test-task"` → 下次 prompt 記錄自動帶入 work_item
