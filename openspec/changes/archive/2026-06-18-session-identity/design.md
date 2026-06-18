## Context

`tt record prompt` 透過 Claude Code `UserPromptSubmit` hook 執行，stdin 攜帶 `session_id`（UUID）。每次 `/clear` 都會產生新 UUID，導致同一個 Claude Code 進程的多段對話在 DB 中被記錄為不同 session。

Hook 執行環境中，`$PPID` = Claude Code 進程 PID，跨所有 `/clear` 不變，直到 Claude Code 關閉。

## Goals / Non-Goals

**Goals:**

- 以 `(process_pid, process_start)` 組合作為跨 `/clear` 穩定的工作 session key
- 保留 `conversation_id`（原 session_id UUID）記錄對話段落數
- 現有 sessions 資料不丟失（新欄位 NULL = 舊資料）
- session 工作時間計算跨所有 conversation_id

**Non-Goals:**

- 跨機器或跨 tmux session 的 session 合併
- PID reuse 的完美防護（`process_start` 解決大多數情境，極端 edge case 不處理）
- 修改 token / cost 計算邏輯（範疇在 enrich-report-and-tt-serve）

## Decisions

### D1：穩定 session key = `(process_pid, process_start)`

只用 `$PPID` 有 PID reuse 風險（重開機後）。加上 `process_start`（進程啟動時間戳）可消除絕大多數 reuse 情境。

**取得 `process_start` 的方式：**
- macOS：`ps -p $PPID -o lstart=`（輸出人類可讀字串，需 parse）
- Linux：讀 `/proc/$PPID/stat` 第 22 欄（jiffies，需轉 epoch）
- 簡化版（本次採用）：hook 傳入 Unix epoch 秒，recorder 接收並存 INTEGER

實作：shell hook 中執行 `ps -p $PPID -o etimes=`（進程已活多少秒），以 `$(date +%s) - etimes` 估算啟動時間戳。跨平台可行，誤差 ±1 秒（足夠辨識 reuse）。

**備選：** 只用 `$PPID` + 當天日期。缺點：若同天重開 Claude Code 且 PID 碰巧相同，會合併成同一 session。選擇加 `process_start` 更精確。

### D2：env var 傳遞 `$PPID` 給 `tt`

Hook 命令改為：

```bash
PROCESS_PID=$PPID PROCESS_START=$(( $(date +%s) - $(ps -p $PPID -o etimes= | tr -d ' ') )) tt record prompt
```

`tt record prompt` 讀取 `$PROCESS_PID`、`$PROCESS_START` env var。

**備選：** CLI flag `--pid`、`--start`。env var 對 hook 腳本更簡潔，採用。

### D3：DB schema 改動

```sql
ALTER TABLE sessions ADD COLUMN process_pid     INTEGER;
ALTER TABLE sessions ADD COLUMN process_start   INTEGER;  -- Unix epoch seconds
ALTER TABLE sessions ADD COLUMN conversation_id TEXT;     -- 原 session_id UUID
```

Upsert key 改為 `(process_pid, process_start)`。舊資料這三欄皆為 NULL。

### D4：Recorder upsert 邏輯

```
收到 (process_pid, process_start, conversation_id):
  SELECT * FROM sessions WHERE process_pid = ? AND process_start = ?
  若找到：
    UPDATE last_seen, conversation_id（若換了則更新）
  若找不到：
    INSERT 新 session，conversation_id = 收到的值
```

`session_id` 欄位保留（向後相容），但不再用於查詢。

## Risks / Trade-offs

- **`ps` 跨平台差異** → 在 macOS 與 Linux 兩種環境各自測試 hook 指令；recorder 對 `process_start = 0` 或 NULL 做防禦
- **PID reuse（同秒）** → 極端情境（同秒重啟 Claude Code 且 PID 相同）理論上可能合併，接受此 trade-off
- **現有 sessions 資料** → 新欄位 NULL，不影響舊查詢；時間計算邏輯需對 NULL 做 fallback

## Migration Plan

1. 寫 DB migration：`ALTER TABLE sessions ADD COLUMN ...`（三欄）
2. 更新 hook 設定（`settings.json`）加入 env var 傳遞
3. 更新 recorder upsert 邏輯
4. 更新 `tt record prompt` 讀取新 env var
5. 現有 sessions 無需 backfill（NULL 為有效舊資料標記）

Rollback：回退 `tt` binary 即可，DB 欄位留著不影響舊版讀取。

## Open Questions

1. Linux 上 `ps -p $PPID -o etimes=` 是否可用（或需改讀 `/proc/$PPID/stat`）？
2. 若 hook 取不到 `process_start`（ps 失敗），recorder 應以 `process_pid` 單獨 upsert，還是降級為舊行為（用 session_id UUID）？
