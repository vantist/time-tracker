## Why

`tt` 的 subagent token 統計存在競態與邊界計算錯誤，導致 token 數長期偏低或偏高：Stop hook 在 subagent JSONL 未完全 flush 前寫入資料，reconcile 因 `input_tokens IS NULL` 條件跳過而永遠無法修正；`extractSubagentTokens` 缺 `to` 邊界導致跨 turn 重複累計；`record.go` 與 `internal/transcript` 各自維護重複 struct 與函數，維護風險高。此外 cache 5m/1h 費率不同但未區分，per-turn model 未記錄，無法在同 session 切換模型時正確計費。

## What Changes

- **Bug D（最高優先）**：`RecordResponse` 加 `subagent_tokens_settled BOOL DEFAULT 0`；reconcile WHERE 條件改為允許重算未結算 subagent token 的 turn；process 死後 reconcile 重算並寫 `subagent_tokens_settled=1`
- **Bug A**：`extractSubagentTokens` 加 `to int` 參數，限制掃描範圍至當前 turn 邊界，防止跨 turn 重複吸收 subagent token
- **Bug E（struct 合併）**：刪除 `record.go` 中重複的 transcript type 與函數，統一使用 `internal/transcript` package；新增 `transcript.ExtractLastTurn` exported func
- **Bug F**：`recorder.countLines` 改用 `bufio.Scanner` 逐行計數，避免大 transcript 全量讀入記憶體
- **階段 2 — cache 細分**：`usageFields` 加 `CacheCreation5m`、`CacheCreation1h`；DB turns 表加對應欄位；pricing 計算拆分兩種費率
- **階段 2 — per-turn model**：DB turns 表加 `model TEXT`；`RecordResponse` 與 reconcile 寫入 per-turn model
- **階段 2 — typed struct**：`ExtractWindow` 回傳 `WindowResult` typed struct，取代 JSON string 傳遞

## Capabilities

### New Capabilities

（無）

### Modified Capabilities

- `subagent-token-capture`：修正跨 turn 重複計算與競態問題（Bug A、Bug D）；reconcile 改為 process 結束後統一重算 subagent token，並以 `subagent_tokens_settled` 欄位追蹤結算狀態
- `event-recording`：`countLines` 效能修正（Bug F）；turns 表新增 `model TEXT`、`cache_creation_5m_tokens`、`cache_creation_1h_tokens`、`subagent_tokens_settled` 欄位；`record.go` 移除重複 struct，改用 `internal/transcript`（Bug E）
- `interrupt-reconcile`：reconcile WHERE 條件加入 `subagent_tokens_settled=0`；reconcile 完成後寫 `subagent_tokens_settled=1`；改用 `WindowResult` typed struct（Bug D）

## Impact

- Affected specs: `subagent-token-capture`、`event-recording`、`interrupt-reconcile`
- Affected code:
  - Modified: `internal/transcript/extract.go`
  - Modified: `internal/recorder/recorder.go`
  - Modified: `internal/recorder/response.go`
  - Modified: `internal/reconcile/reconcile.go`
  - Modified: `internal/db/schema.go`
  - Modified: `internal/pricing/pricing.go`
  - Modified: `cmd/tt/record.go`

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-19-brainstorm-session-data-redesign.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
