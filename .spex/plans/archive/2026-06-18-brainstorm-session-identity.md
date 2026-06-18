# Session 識別：跨 /clear 的穩定 session 追蹤

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). -->

## Context

`tt record` 靠 Claude Code hook stdin 的 `session_id`（UUID）識別 session。
但 `/clear` 每次產生新 UUID，導致同一次工作被切割成多個 session，成本與時間估算失真。

## 問題

```
開啟 Claude Code → session_id: UUID-A
/clear           → session_id: UUID-B  ← tt 以為是新 session
/clear           → session_id: UUID-C  ← 又一個新 session
```

使用者意圖：UUID-A/B/C 都是「同一個工作 session」，直到 Claude Code 進程關閉。

## 識別依據分析

| 識別依據 | 跨 /clear 穩定？ | 唯一性 | 取得方式 |
|---------|----------------|-------|---------|
| `session_id` (UUID) | ✗ — 每次 /clear 換新 | 高 | stdin JSON |
| `$PPID` | ✓ — 進程不變 | 高（同機器） | hooks 環境變數 |
| `tmux_session` | ✓ | 中 | `$TMUX` |
| `cwd` | ✓ | 低 | stdin JSON |

## 決策

**以 `$PPID` 作為「工作 session」穩定識別符**，`session_id` UUID 降為「對話段落」欄位。

`$PPID` 語義完全符合：
- Hook 是 Claude Code 子進程，`$PPID` = Claude Code PID
- 跨所有 `/clear` 不變
- Claude Code 關掉 → 進程結束 → 自然是 session 結束

## 設計方案

### DB 改動

```
sessions 表新增欄位：
  - process_pid    INTEGER  — $PPID，工作 session 主 key
  - process_start  INTEGER  — 進程啟動時間戳（解決 PID reuse）
  - conversation_id TEXT    — 原 session_id UUID（/clear 後換，記錄段數）
```

工作 session key = `(process_pid, process_start)` 組合，避免重開機 PID reuse。

### Hook 改動

`UserPromptSubmit` hook (tt record prompt) 需傳入 `$PPID`：

```bash
# 現有：tt record prompt  ← stdin 有 session_id
# 改為：透過 --pid flag 或在 stdin JSON 外另傳
PPID_VAL=$PPID tt record prompt
```

或修改 `tt record` 讀取 `$TT_PROCESS_PID` env var。

### Recorder 改動

`RecordPromptSilent` upsert 邏輯：
1. 以 `(process_pid, process_start)` 查 sessions
2. 找到 → 更新 last_seen、conversation_id（如果換了）
3. 找不到 → 新建 session

### 時間計算改動

session 工作時間 = 從第一個 prompt 到最後一個 response，跨所有對話段落。

## Open Questions

1. **PID reuse**: 加 `process_start`（讀 `/proc/$PPID/stat` 或 macOS `ps -p $PPID -o lstart`）實作複雜度中等。簡單版：只用 `$PPID` + 當天日期組合，夠用嗎？
2. **hooks 怎麼拿到 process_start**: macOS 上 `ps -p $PPID -o lstart=` 可行，但跨平台性待確認。
3. **現有 DB migration**: 現有 sessions 要怎麼處理（`process_pid = NULL` = 舊資料）。

## Next Steps

→ `/spex-ingest` 加入現有 `enrich-report-and-tt-serve` change，或 `/spex-propose` 獨立 change。
