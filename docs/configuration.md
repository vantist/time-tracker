# Configuration — 系統變數、設定檔與配置鍵

本文件彙整了 `tt` 專案中的環境變數、配置檔案儲存路徑以及全域配置鍵值，供開發人員與 AI Agent 參考遵循。

## 系統環境變數

在執行、測試或進行 Hook 呼叫時，系統支援或需要以下環境變數：

| 環境變數 | 資料型態 | 預設值 | 說明 |
|----------|----------|--------|------|
| `TT_DB_PATH` | `String` (檔案路徑) | `~/.tt/data.db` | 覆寫本地 SQLite 資料庫檔案的儲存路徑。主要用於開發、測試時隔離資料，或供使用者自訂路徑。 |
| `PROCESS_PID` | `Integer` | 無 (自動偵測) | 供 AI 工具 Hook 呼叫時傳入，代表呼叫端 Process 的 PID。若與 `PROCESS_START` 共同提供，會覆寫預設自動偵測的父行程 PID（即 `os.Getppid()`），用於精確對應工作會話。 |
| `PROCESS_START` | `Integer` | 無 (自動偵測) | 供 AI 工具 Hook 呼叫時傳入，代表呼叫端 Process 的啟動時間戳記（Unix Nano）。配合 `PROCESS_PID` 用於防止 PID 重用（PID reuse）造成的 Session 誤判。若未提供，則退回以 `process.StartTime(ppid)` 自動取得父行程啟動時間。 |
| `ORCA_WORKSPACE_ID` | `String` (格式為 `owner::path`) | 無 | 專案目錄後備解析變數。當前專案目錄未指定或判定為無效路徑（如路徑包含 `.gemini`、`.claude`、`.copilot`）時，系統會以 `::` 分割此變數並提取後半部的 `path`。若其為有效目錄，則覆寫當前 `project` 工作目錄。 |
| `ORCA_WORKTREE_ID` | `String` (格式為 `owner::path`) | 無 | 專案目錄後備解析變數。當 `ORCA_WORKSPACE_ID` 未能成功解析出有效目錄時，系統將此變數作為 fallback 進行與上述相同的解析及目錄檢查。 |

---

## 設定檔與儲存路徑

`tt` 將配置、資料與運行時狀態儲存於使用者的家目錄之下（預設為 `~/.tt/`），並與多個 AI 編輯工具的設定檔進行整合：

### 1. 本地全域配置與資料
* **配置檔案：`~/.tt/config.json`**
  - **格式**：JSON Key-Value
  - **說明**：持久化儲存全域配置（如 `idle-threshold`）。若檔案不存在，讀取時會傳回預設值；寫入配置時會自動建立此檔案（以 `0o644` 權限）。
* **SQLite 資料庫：`~/.tt/data.db`**
  - **格式**：SQLite Database
  - **說明**：存放所有工作 Session、Turn 記錄、Token 消耗以及估算費用的主要資料庫。若檔案不存在，程式首次啟動時會自動建立資料庫並進行 Schema 初始化。可透過環境變數 `TT_DB_PATH` 自訂其位置。
* **Reconcile 鎖定檔：`~/.tt/reconcile.lock`**
  - **格式**：排他性鎖定標記檔
  - **說明**：用於在執行 `tt reconcile` 調和時防止多實例併發衝突。
* **專案工作標籤：`~/.tt/work-items/<projectKey>`**
  - **格式**：純文字（檔名為專案 Key，內容為標籤加換行符 `\n`）
  - **說明**：儲存個別專案目前標記的工作項目標籤。執行 `tt work [label]` 時寫入，執行 `tt work --clear` 時刪除。
  - **`<projectKey>` 生成演算法**：以該專案的 Git Root 路徑（若非 Git 專案則為當前工作目錄）經 SHA256 雜湊後，取前 8 個位元組（16 字元）的十六進位字串作為檔名。

### 2. AI 編輯器與工具整合
* **Claude Code 整合設定：`~/.claude/settings.json`**
  - **格式**：JSON
  - **說明**：執行 `tt setup --claude-code` 時，系統會自動在該檔案的 `"hooks"` 區段中合併寫入 `"UserPromptSubmit"` 與 `"Stop"` 事件的 Hook 指令。
  - **結構範例**：
    ```json
    {
      "hooks": {
        "UserPromptSubmit": [
          {
            "_owner": "tt",
            "hooks": [
              { "type": "command", "command": "tt record prompt" }
            ]
          }
        ],
        "Stop": [
          {
            "_owner": "tt",
            "hooks": [
              { "type": "command", "command": "tt record response" }
            ]
          }
        ]
      }
    }
    ```
* **Google Antigravity 整合設定：`~/.gemini/config/hooks.json`**
  - **格式**：JSON
  - **說明**：執行 `tt setup --antigravity` 時自動合併寫入 `"tt"` 區段中的 `"PreInvocation"` 與 `"Stop"` 事件。
  - **結構範例**：
    ```json
    {
      "tt": {
        "PreInvocation": [
          { "_owner": "tt", "type": "command", "command": "tt record prompt --tool antigravity" }
        ],
        "Stop": [
          { "_owner": "tt", "type": "command", "command": "tt record response --tool antigravity" }
        ]
      }
    }
    ```
* **Antigravity CLI 工具模型設定：`~/.gemini/antigravity-cli/settings.json` 或 `~/.gemini/antigravity/settings.json`**
  - **格式**：JSON
  - **說明**：儲存 Antigravity 所使用的模型設定。在 `reconcile` 調和與 Prompt 輸入解析時，系統會嘗試讀取並清理/常態化該檔案中的 `"model"` 欄位，若皆無法讀取，則 fallback 預設為 `gemini-3.5-flash`。
* **Codex 整合設定：`~/.codex/hooks.json`**
  - **格式**：JSON
  - **說明**：執行 `tt setup --codex` 時自動合併寫入 `"hooks"` 區段中的 `"UserPromptSubmit"` 與 `"Stop"` 事件。
  - **結構範例**：
    ```json
    {
      "hooks": {
        "UserPromptSubmit": [
          { "_owner": "tt", "type": "command", "command": "tt record prompt --tool codex" }
        ],
        "Stop": [
          { "_owner": "tt", "type": "command", "command": "tt record response --tool codex" }
        ]
      }
    }
    ```
* **Copilot CLI 整合設定：`~/.copilot/hooks/tt.json`**
  - **格式**：JSON
  - **說明**：執行 `tt setup --copilot` 時自動建立或更新該檔案，包含 `"version": 1` 與 `"hooks"` 區段中 `"userPromptSubmitted"` 及 `"agentStop"` 事件。
  - **結構範例**：
    ```json
    {
      "version": 1,
      "hooks": {
        "userPromptSubmitted": [
          { "_owner": "tt", "type": "command", "command": "tt record prompt --tool copilot-cli" }
        ],
        "agentStop": [
          { "_owner": "tt", "type": "command", "command": "tt record response --tool copilot-cli" }
        ]
      }
    }
    ```

---

## 全域配置鍵值 (Configuration Keys)

目前儲存於 `~/.tt/config.json` 中的全域配置鍵值如下：

### `idle-threshold`
- **資料型態**：`String` (內容為整數分鐘數)
- **預設值**：`"15"` (即 15 分鐘)
- **說明**：使用者閒置判定的時間閾值。在查詢或計算使用者實際活動時間（`UserActiveTime`）時，若兩次 Prompt 之間的時間間隔小於此值，則視為連續工作；若大於或等於此值，則視為閒置，該間隔時間將不予計入工時。
- **設定指令**：`tt config set idle-threshold <value>`。

### `model` (Antigravity 相關)
- **資料型態**：`String`
- **預設值**：`"gemini-3.5-flash"` (若設定檔無配置時 fallback 使用)
- **說明**：儲存於 `~/.gemini/antigravity-cli/settings.json` 或 `~/.gemini/antigravity/settings.json` 中，指定預設使用的 AI 模型名稱。系統解析時會進行常態化處理（如轉換為小寫、移除括號與空白）。

---
Related: [Architecture](../ARCHITECTURE.md) | [Commands](commands.md) | [Conventions](conventions.md)
