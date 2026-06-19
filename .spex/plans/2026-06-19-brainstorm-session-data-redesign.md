# Session Data 完整重設計 — Token、時間、Subagent

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). -->

## Context

目標：讓 tt 對單一 session 的資料做完整剖析——token（含 subagent、input/output/cache）、model、時間（精確到 ms）。

---

## 現有架構與問題清單

### 1. Transcript 格式（實際觀察）

位置：`~/.claude/projects/<project-slug>/<uuid>.jsonl`  
Subagents：`~/.claude/projects/<project-slug>/<uuid>/subagents/agent-<id>.{jsonl,meta.json}`

關鍵欄位：
```json
{
  "type": "assistant",
  "isSidechain": false,
  "timestamp": "2026-06-02T06:40:30.950Z",
  "message": {
    "model": "claude-sonnet-4-6",
    "usage": {
      "input_tokens": 1,
      "output_tokens": 192,
      "cache_read_input_tokens": 44760,
      "cache_creation_input_tokens": 35976,
      "cache_creation": {
        "ephemeral_5m_input_tokens": 35976,
        "ephemeral_1h_input_tokens": 0
      },
      "server_tool_use": {
        "web_search_requests": 0,
        "web_fetch_requests": 0
      }
    }
  }
}
```

Subagent JSONL 中：`isSidechain=True`（全部）。

### 2. 已確認 Bug

**Bug D：Stop hook 競態 + reconcile 無 subagent re-check（最高嚴重）**

根本設計問題，造成 subagent token 永遠無法修正。

三條漏算路徑：

| # | 情境 | 症狀 |
|---|------|------|
| D1 | offset 不存在 → fallback `extractFromTranscript` → 從不呼叫 `extractSubagentTokens` | Subagent token 完全沒有 |
| D2 | Stop hook 在 subagent JSONL 未完全 flush 前執行 → 寫入部分值 | Subagent token 偏低 |
| D3 | Stop hook 寫了非零 input_tokens → reconcile WHERE 條件跳過這一行 | 永遠無法重算 |

機制：

```
Stop hook 時機（subagent JSONL 可能未完全 flush）
     ↓
input_tokens 寫入（可能只有 main tokens）
     ↓
reconcile WHERE (response_at IS NULL OR input_tokens IS NULL) → 跳過
     ↓
永遠錯了
```

修法方向：
- reconcile 條件改成允許重算（加 `input_tokens = 0` 或新增 `subagent_tokens_settled BOOL`）
- 或 Stop hook 不寫 subagent token，統一由 reconcile 在 process 結束後重算

**Bug A：extractSubagentTokens 忽略 `to` 邊界（高嚴重）**

```go
// extract.go:65
sub := extractSubagentTokens(path, all, from)  // to 從未傳入

// extract.go:134
for i := offset; i < len(entries); i++ {  // 掃到 EOF，不尊重 to
```

效果：Turn 1 會把 Turn 2+ 的 subagent token 全部重複吸收。  
修法：`extractSubagentTokens(path, all, from, to int)` 並限制掃描範圍。

**Bug E（原 B）：duplicate struct（中）**

`cmd/tt/record.go` 和 `internal/transcript/extract.go` 各自定義了 `transcriptEntry/entry`、`usageFields`、`extractSubagentTokens`、`sumWindow`、`loadTranscript`。兩份實作邏輯不完全同步，維護風險高。

**Bug F（原 C）：`countLines` 讀整個 transcript（效能）**

```go
data, err := io.ReadAll(f)  // transcript 可能 >1MB
return bytes.Count(data, []byte("\n"))
```

大 session 每次 prompt hook 都讀一遍整個 transcript。應改成 `bufio.Scanner` 計行。

### 3. 遺漏的資料

| 資料 | 現況 | 影響 |
|------|------|------|
| `timestamp` 欄位 | 完全未使用 | 無法得知精確 AI 回應時間 |
| `cache_creation.ephemeral_5m/1h` | 未擷取 | 成本計算無法區分 5min vs 1h cache |
| `server_tool_use.web_search_requests` | 未擷取 | 未來計費可能需要 |
| per-turn model | 只記 session 層級 | 同 session 切換模型時計費錯誤 |
| user prompt 文字 | 完全沒存 | 無法在 report 裡顯示 prompt 內容 |
| subagent 開始/結束時間 | 完全沒有 | 無法顯示 subagent 耗時 |

### 4. 現有設計的正確之處（不要動）

- subagent JSONL 裡 `isSidechain=True` → `sumSubagentWindow` 不過濾正確
- `seen` map 以 usageKey 去重複（防止 Claude Code 對同一 API call 寫多條相同 usage）
- reconcile `WHERE (response_at IS NULL OR input_tokens IS NULL)` 邏輯正確

---

## 設計方向

### 方向 A：最小修補（只修 Bug）

修 Bug A（加 `to` 參數）、合併重複 struct（移 `record.go` 的 → `internal/transcript`）、修 `countLines`。

- 代價小，一個 PR
- 遺漏資料問題留著不動

### 方向 B：中等重設計（推薦）

在 A 的基礎上：
1. 移除 `record.go` 中重複的 transcript 邏輯，改用 `internal/transcript`
2. `usageFields` 加入 `CacheCreation5m`、`CacheCreation1h`
3. DB turns 表加 `model TEXT` 欄位（per-turn）
4. `ExtractWindow` 回傳結構改為 typed struct 而非 JSON string

不動：user prompt 文字、subagent 時間、`server_tool_use`

### 方向 C：完整重設計

B + per-turn model + timestamp-based response_at（從 transcript `timestamp` 而非 `time.Now()`）+ subagent 時間統計。

DB schema 需加欄位。實作量大。

---

## 設計結論

**決策**：方向 B，分兩階段

**理由**：Bug A 是真實的重複計算問題，必須修。struct 重複是維護地雷。Cache 細分 5m/1h 對成本計算有實際影響（兩者單價不同）。

**階段 1（Bug fix + cleanup）**：
- 修 Bug D：Stop hook 不寫 subagent token；reconcile 統一在 process 結束後重算 subagent（最高優先）
- 修 Bug A：`extractSubagentTokens` 加 `to` 參數
- 合併重複 struct：`record.go` 使用 `internal/transcript`
- 修 `countLines` 改 bufio

**階段 2（Data enrichment）**：
- per-turn model（DB migration + ExtractWindow 修改）
- `cache_creation.ephemeral_5m/1h` 欄位
- `ExtractWindow` 回傳 typed struct

**未納入**（YAGNI）：user prompt 文字、subagent 時間、server_tool_use

**開放問題**：
- per-turn model 需要 DB migration — `addTurnColumns` 加 `model TEXT`？還是之後另一個 spex change？
- `ExtractWindow` 改 typed struct 會破壞 reconcile 呼叫方，需一起改

---

## 行動選項

- `/spex-propose` — 建立正式 change，從 Bug fix 開始
- 直接 `/spex-apply` 到現有 change（若已有相關 change）
