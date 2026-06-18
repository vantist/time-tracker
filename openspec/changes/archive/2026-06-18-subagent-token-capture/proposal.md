## Why

`tt record response` 只從主 transcript 提取 token，忽略 subagent（Agent tool、Workflow、/code-review 等）的獨立 `subagents/agent-<id>.jsonl`。一次 /code-review 跑 5 個平行 subagent 約 150k tokens，造成 turn 成本嚴重低估。

## What Changes

- 新增 `extractSubagentTokens(transcriptPath, offset)` 函式：掃描主 transcript 的 `tool_use` entries 找 Agent 呼叫，讀取對應 subagent jsonl，合計 token
- 修改 `extractFromTranscriptAtOffset`：完成主 transcript 提取後，加入 subagent token 合計
- Subagent token 合入現有欄位（input/output/cache），不新增欄位

## Capabilities

### New Capabilities

- `subagent-token-capture`: 提取並合計 subagent（Agent tool、Workflow 等）產生的 token，納入 turn 成本計算

### Modified Capabilities

(none)

## Impact

- Affected code:
  - Modified: `cmd/tt/record.go`

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-18-brainstorm-subagent-token-capture.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
