## 1. RC4 — currentModel fallback（parser 層）

- [x] 1.1 寫失敗測試：`copilot_transcript_test.go` 新增 case，`session.shutdown` 的 `mainModel` 空、`currentModel="gpt-5"` → 期望 `WindowResult.Model()=="gpt-5"`、subagent 判定用 `currentModel`
- [x] 1.2 寫失敗測試：`copilot_transcript_test.go` 新增 case，`mainModel="gpt-5"`、`currentModel="claude-3.5"` → 期望 `Model()=="gpt-5"`（mainModel 優先）
- [x] 1.3 修改 `internal/transcript/copilot_transcript.go`：`copilotEvent.Data` 加 `CurrentModel string json:\"currentModel\"`；`mainModel` 為空時 fallback 到 `CurrentModel`
- [x] 1.4 跑 `go test ./internal/transcript/ -run TestParseCopilotLog` 確認 1.1/1.2 通過
- [x] 1.5 寫失敗測試：`vscode_copilot_test.go`（或 `vscode_copilot_debug_test.go`）新增 shutdown case，`mainModel` 空、`currentModel="gpt-5"` → 期望 `Model()=="gpt-5"`
- [x] 1.6 修改 `internal/transcript/vscode_copilot.go`：shutdown struct 加 `CurrentModel` 欄位；`MainModel` 為空時 fallback
- [x] 1.7 跑 `go test ./internal/transcript/ -run TestVscodeCopilot` 確認 1.5 通過

## 2. RC1 — reconcile WHERE 放寬 + path 自推

- [ ] 2.1 寫失敗測試：`reconcile_test.go` 新增 Copilot session 案例——`tool='copilot-cli'`、`transcript_path=NULL`、`prompt_line_offset=NULL`、`input_tokens=NULL`，`events.jsonl` 含 `session.shutdown` → 期望 reconcile 後該 turn 進入處理流程（非 skip）
- [ ] 2.2 修改 `internal/reconcile/reconcile.go` 的 `reconcile` SQL：WHERE 條件新增 `OR s.tool='copilot-cli'`（保留原 `transcript_path IS NOT NULL AND prompt_line_offset IS NOT NULL` 條件）
- [ ] 2.3 修改 `reconcileTurn`：`dt.transcriptPath == ""` 時呼叫 `GetProvider(dt.tool).ResolvePath(dt.sessionID, "")` 推導 path；`os.Stat` 失敗則靜默 skip
- [ ] 2.4 跑 `go test ./internal/reconcile/ -run TestReconcileCopilot` 確認 2.1 通過（此階段尚未做歸因，先驗證 turn 進入流程且 path 推導正確）

## 3. RC1 — Copilot session 級 token 歸因

- [ ] 3.1 寫失敗測試：`reconcile_test.go` 新增 case——Copilot session 3 個 turn、`session.shutdown` 累計 `inputTokens=1000`、`outputTokens=500` → 期望最新 open turn `input_tokens=1000`、其餘兩 turn `input_tokens=0` 且 `subagent_tokens_settled=1`
- [ ] 3.2 寫失敗測試：reconcile 冪等——同一 Copilot session 連續兩次 `MaybeReconcile`，第二次所有 turn token 值不變
- [ ] 3.3 寫失敗測試：跨 shutdown——兩次 shutdown（累計 1000、1500）→ report 加總 = 1500
- [ ] 3.4 在 `reconcile.go` 新增 Copilot 歸因函式：對 `tool='copilot-cli'` session，先清空所有 turn 的 token 欄位與 `turn_model_usages`，再將累計值寫到最新 open turn（`response_at IS NULL` 者；若無則最新 turn），其餘 turn 補 `input_tokens=0`、`output_tokens=0`、`response_at`（`nextPromptAt - 1ms` 或自身 `prompt_at`）、`subagent_tokens_settled=1`
- [ ] 3.5 修改 `reconcile` 主流程：偵測 `tool='copilot-cli'` session 時呼叫歸因函式，跳過一般 `reconcileTurn` 邏輯
- [ ] 3.6 跑 `go test ./internal/reconcile/` 確認 3.1/3.2/3.3 通過

## 4. RC5 — resolveModel provider 分流

- [ ] 4.1 寫失敗測試：`reconcile_test.go` 新增 case——`tool='copilot-cli'` session 的 `turns.model` 為空或 `gemini-3.5-flash`，`events.jsonl` 含 `session.shutdown` 的 `currentModel="gpt-5"` → 期望 `repairSessions` 後 `turns.model="gpt-5"`
- [ ] 4.2 修改 `resolveModel`：改用 `GetProvider(tool).ExtractWindow` 取代寫死 Claude Code parser；移除 Antigravity `settings.json` fallback 對 Copilot session 的觸發
- [ ] 4.3 修改 `repairSessions`：對 `tool='copilot-cli'` session，當 `findExistingTranscriptPath` 回空時改用 `CopilotProvider.ResolvePath(sessionID, "")` 推導 path 餵給 `resolveModel`
- [ ] 4.4 跑 `go test ./internal/reconcile/ -run TestRepairSessions` 確認 4.1 通過

## 5. 整合驗證

- [ ] 5.1 跑 `go test ./internal/transcript/ ./internal/reconcile/` 確認全綠
- [ ] 5.2 跑 `go build ./...` 確認無編譯錯誤
- [ ] 5.3 手動驗證：對真實 DB 的 `61232c8e` / `e0af7acb` session 跑 `tt report`，確認 token 非 0、model 非 `gemini-3.5-flash`
- [ ] 5.4 跑 `go test ./...` 確認無回歸
