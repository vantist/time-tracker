## ADDED Requirements

### Requirement: opencode plugin 將 user prompt 對應至 tt record prompt

系統 SHALL 提供一個 opencode plugin（由 `tt setup --opencode` 產生），監聽 `message.updated` event，當訊息 `role = "user"` 且 `time.created` 存在時，透過 `exec` 呼叫：

```
tt record prompt --tool opencode --session <info.sessionID> --project <plugin directory> --model <info.model?.modelID ?? "">
```

`--tool` SHALL 為常數 `"opencode"`。`--project` SHALL 為 plugin context 提供的 `directory`（opencode 啟動時工作目錄）。`--model` 於 user 訊息未帶 model 時 SHALL 傳空字串。

#### Scenario: user prompt 觸發 tt record prompt

- **WHEN** opencode plugin 收到 `message.updated` event，其 `role = "user"` 且 `time.created` 存在
- **THEN** plugin 以 `exec` 呼叫 `tt record prompt --tool opencode --session <info.sessionID> --project <directory> --model <info.model?.modelID ?? "">`
- **THEN** `--tool` 值為 `"opencode"`

#### Scenario: user 訊息未帶 model 時傳空字串

- **WHEN** `message.updated` event 的 `role = "user"` 且 `info.model` 為 `undefined`
- **THEN** plugin 呼叫 `tt record prompt` 時 `--model` 傳空字串 `""`

### Requirement: opencode plugin 將 assistant response 對應至 tt record response

系統 SHALL 在 plugin 收到 `message.updated` event 且同時符合下列條件時，透過 `exec` 呼叫 `tt record response`：

- `role = "assistant"`
- `info.model.modelID` 存在
- `tokens` 存在
- `time.completed` 或 `finish` 存在（串流結束，非中間態）

flag 對應：

| Flag | 值 |
|------|----|
| `--tool` | `"opencode"`（常數） |
| `--session` | `info.sessionID` |
| `--model` | `info.model.modelID` |
| `--tokens` | JSON 字串 `{input_tokens, output_tokens, reasoning_tokens?, cache_read_tokens?, cache_write_tokens?}` |
| `--subagent-tokens` | 當該 session 有暫存 subagent token 時帶入 JSON 陣列；否則省略 |

#### Scenario: assistant response 完成時觸發 tt record response

- **WHEN** plugin 收到 `message.updated` event，`role = "assistant"`、`info.model.modelID = "claude-sonnet-4-6"`、`tokens` 存在、`time.completed` 存在
- **THEN** plugin 呼叫 `tt record response --tool opencode --session <info.sessionID> --model claude-sonnet-4-6 --tokens <json>`
- **THEN** `--tokens` JSON 包含 `input_tokens` 與 `output_tokens` 欄位

#### Scenario: 串流中間態不觸發記錄

- **WHEN** plugin 收到 `message.updated` event，`role = "assistant"`、`modelID` 存在、`tokens` 存在，但 `time.completed` 與 `finish` 均不存在
- **THEN** plugin 不呼叫 `tt record response`，等待下一幀

### Requirement: opencode plugin subagent token 暫存與隨主 response flush

opencode subagent 訊息與主 agent 訊息位於同一 event stream，差異為 `info.agent` 欄位。系統 SHALL 在 plugin 記憶體維護 `pendingSubTokens[sessionId][messageId]` 暫存表：

- 當 `info.agent` 為 `undefined` 或空字串時，視為主 agent，依前一需求觸發 `tt record response`
- 當 `info.agent` 為非空字串（如 `"build"`、`"explore"`）時，視為 subagent，將其 `{model, provider, tokens: {input, output, ...}}` 暫存至 `pendingSubTokens[sessionId][messageId]`

主 agent response 完成觸發 `tt record response` 時，plugin SHALL 將該 session 暫存的所有 subagent token 組成 JSON 陣列帶入 `--subagent-tokens` flag，並於 exec 完成後清空該 session 的 `pendingSubTokens`。

#### Scenario: subagent token 暫存並隨主 response 一起送出

- **WHEN** session `s1` 中先後收到 subagent 訊息（`info.agent = "build"`，tokens = X）與主 agent response 完成訊息
- **THEN** 主 agent response 觸發 `tt record response --tool opencode --session s1 --model <main> --tokens <main json> --subagent-tokens <json 陣列含 X>`
- **THEN** exec 完成後 `pendingSubTokens["s1"]` 被清空

#### Scenario: 無 subagent 時省略 --subagent-tokens flag

- **WHEN** session `s1` 中主 agent response 完成，且 `pendingSubTokens["s1"]` 為空
- **THEN** plugin 呼叫 `tt record response` 時省略 `--subagent-tokens` flag

### Requirement: opencode plugin 串流去重

系統 SHALL 在 plugin 維護一個 `seen` 集合，對每筆 assistant event 計算 `dedupKey = ${messageId}-${input_tokens}-${output_tokens}-${cache_read_tokens}`。當 `seen.has(dedupKey)` 為真時，plugin SHALL 跳過該 event 不重複觸發 `tt record response`；否則將 `dedupKey` 加入 `seen` 後繼續處理。

#### Scenario: 重複的串流 frame 不重複記錄

- **WHEN** plugin 連續收到兩筆 `message.updated` event，其 `messageId`、`input_tokens`、`output_tokens`、`cache_read_tokens` 完全相同
- **THEN** 僅第一筆觸發 `tt record response`，第二筆被 `seen` 集合攔截跳過

### Requirement: tt setup --opencode 冪等產生 plugin 檔

系統 SHALL 提供 `tt setup --opencode` 命令，於 `~/.config/opencode/plugins/tt-bridge.ts` 產生 opencode plugin 檔。該命令 SHALL 為冪等：當目標檔已存在時，SHALL NOT 覆蓋或重複產生，並輸出提示訊息告知檔案已存在。當目標檔不存在時，系統 SHALL 建立必要父目錄後寫入 plugin 內容，並輸出 `OpenCode plugin configured in ~/.config/opencode/plugins/tt-bridge.ts`。

#### Scenario: 首次設定產生 plugin 檔

- **WHEN** 執行 `tt setup --opencode` 且 `~/.config/opencode/plugins/tt-bridge.ts` 不存在
- **THEN** 系統建立 `~/.config/opencode/plugins/` 目錄（若不存在）並寫入 plugin 檔
- **THEN** stdout 輸出 `OpenCode plugin configured in ~/.config/opencode/plugins/tt-bridge.ts`
- **THEN** 命令回傳 exit code 0

#### Scenario: 檔案已存在時不覆蓋

- **WHEN** 執行 `tt setup --opencode` 且 `~/.config/opencode/plugins/tt-bridge.ts` 已存在
- **THEN** 系統不覆蓋該檔案內容
- **THEN** stdout 輸出檔案已存在的提示訊息
- **THEN** 命令回傳 exit code 0

### Requirement: opencode 專案目錄解析為 git root

系統 SHALL 將 plugin context 提供的 `directory`（opencode 啟動時工作目錄）透過 `workitem.ResolveProject(dir)` 解析為 git root 作為 `--project` 傳入 `tt record prompt`。`ResolveProject` SHALL 執行 `git -C dir rev-parse --show-toplevel`，成功時回傳 git root，失敗時（非 git repo）回傳原 `dir`。

#### Scenario: directory 為 git repo 時回傳 git root

- **WHEN** plugin context `directory = "/Users/foo/time-tracker"` 且該路徑為 git repo
- **THEN** `--project` 傳入 `ResolveProject` 回傳的 git root 路徑

#### Scenario: directory 非 git repo 時回傳原路徑

- **WHEN** plugin context `directory = "/tmp/scratch"` 且該路徑非 git repo
- **THEN** `--project` 傳入 `/tmp/scratch`，不報錯
