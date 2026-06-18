# Brainstorm: Subagent Token Capture

**Date**: 2026-06-18  
**Status**: Converged

## Problem

`tt record response` 只從主 transcript 提取 token。呼叫 subagent（Agent tool、Workflow、/code-review 等）時，subagent 的 token 存在獨立的 `subagents/agent-<id>.jsonl`，完全未被計算。

截圖顯示一次 /code-review 跑 5 個平行 subagent，約 150k tokens——不是邊緣情境。

## Discovery

### 檔案結構

```
~/.claude/projects/<project>/<session_id>.jsonl          ← 主 transcript
~/.claude/projects/<project>/<session_id>/subagents/
    agent-<id>.jsonl       ← subagent transcript (isSidechain=True)
    agent-<id>.meta.json   ← { agentType, description, toolUseId }
```

### Key finding

- 主 transcript 的 `tool_use` entries 中，`id` 欄位與 `meta.json.toolUseId` 完全對應
- 主 transcript 無 sidechain entries（全在獨立 jsonl）
- Subagent jsonl 中的 assistant entries 全部 `isSidechain=True`
- 現行 extract 函式 skip `isSidechain=True` → subagent token 被丟棄

### Verified

```
main transcript: tool_use { id: "toolu_vrtx_01DTUv3yv15HZHfXfM9iXfUv", name: "Agent" }
meta.json:       { "toolUseId": "toolu_vrtx_01DTUv3yv15HZHfXfM9iXfUv", ... }
```

完全對應。

## Design Decision

**合計進現有欄位**（不新增欄位）  
理由：Turn 成本 = 主 agent + subagents 總和，這就是使用者想知道的數字。

## Implementation Plan

### `extractSubagentTokens(transcriptPath string, offset int) usageFields`

1. 掃 transcript 第 `[offset:]` 行，找 `content[].type == "tool_use" && name == "Agent"` → 收集 toolUseIds
2. 推導 subagentsDir：
   ```
   transcript: ~/.claude/projects/foo/<session_id>.jsonl
   subagents:  ~/.claude/projects/foo/<session_id>/subagents/
   ```
3. 讀所有 `*.meta.json`，比對 toolUseId → 對應 `.jsonl`
4. 對每個 `.jsonl`：dedup+sum assistant entries（忽略 `isSidechain`，全讀）
5. 回傳合計

### 修改點

- `cmd/tt/record.go`: `extractFromTranscriptAtOffset` 完成後，加 `extractSubagentTokens` 並合計
- 只改 `extractFromTranscriptAtOffset` 路徑（有 offset 時才嘗試），`extractFromTranscript`（fallback）維持不變

### Known Limitation

Subagent 巢狀（subagent 再呼叫 agent）不處理。subagents/ 目錄下不遞迴。

## Next Steps

- `/spex-ingest` 將此設計記入現有 `ai-tool-time-tracker` change
- 或 `/spex-propose` 建立獨立 change
