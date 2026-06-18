## Context

`cmd/tt/record.go` 的 `extractFromTranscriptAtOffset` 從主 transcript 提取 token 後回傳。主 transcript 的 `tool_use` entries 記錄 Agent 呼叫（`name: "Agent"`），ID 對應 `subagents/<session_id>/subagents/agent-<id>.meta.json` 的 `toolUseId`。Subagent 的實際 token 在各自的 `agent-<id>.jsonl`，現行程式完全未讀取。

檔案結構：
```
~/.claude/projects/<project>/<session_id>.jsonl          ← 主 transcript
~/.claude/projects/<project>/<session_id>/subagents/
    agent-<id>.jsonl       ← subagent transcript
    agent-<id>.meta.json   ← { toolUseId, agentType, description }
```

## Goals / Non-Goals

**Goals:**
- 讀取與主 transcript 同一 turn（offset 之後）的 Agent tool_use entries
- 對應到 subagent jsonl，合計 token
- 合入現有 input/output/cache 欄位，不改 DB schema

**Non-Goals:**
- 巢狀 subagent（subagent 再呼叫 Agent）不遞迴處理
- `extractFromTranscript`（fallback 路徑，無 offset）不修改
- Workflow tool 的 subagent 若也用 Agent tool 呼叫則自然涵蓋，無需特判

## Decisions

### 決定一：合入現有欄位而非新增欄位

Turn 成本 = 主 agent + subagents 總和，這是使用者關心的單一數字。不新增欄位保持 DB schema 穩定，不需 migration。

替代方案：新增 `subagent_tokens` 欄位分開顯示。拒絕理由：分開顯示增加複雜度，使用者想知道的是總成本。

### 決定二：以 tool_use id 配對 meta.json，而非掃目錄所有 subagent

只計算本 turn（offset 之後）產生的 Agent 呼叫，避免把前一個 turn 的 subagent token 重複計入。

替代方案：掃 subagents/ 目錄所有檔案取最新的 N 個。拒絕理由：與 turn 邊界脫鉤，易重複計算。

### 決定三：新函式 `extractSubagentTokens`，呼叫點在 `extractFromTranscriptAtOffset` 結束後

與主 transcript 提取分離，各自可獨立測試。

## Risks / Trade-offs

- [風險] subagents/ 目錄不存在或 meta.json 缺失 → 靜默略過，回傳零值，不影響主流程
- [風險] subagent jsonl 有重複 entry → 沿用現有 sumWindow dedup 邏輯
- [限制] 巢狀 subagent token 不計算 → 已知限制，文件記錄即可
- [trade-off] 需擴展 `transcriptEntry` 加入 `Content []ContentBlock` → 增加反序列化成本，但 loadTranscript 已讀整個檔案，影響可忽略
