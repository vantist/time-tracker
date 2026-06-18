## Context

`tt work` 用於記錄當前正在處理的工作項目標籤，讓 `recorder.RecordPrompt` 可以把該標籤附加到 session 紀錄。目前所有 repo 共用同一個檔案 `~/.tt/work-item`，切換 repo 時會互相覆蓋。

`internal/workitem/workitem.go` 現有 API：
```go
func Get() (string, error)
func Set(label string) error
func Clear() error
```

`recorder.RecordPrompt` 透過 `input.Project`（hook 傳入的 CWD）辨識來源目錄，但 work item 查詢沒有用到這個資訊。

## Goals / Non-Goals

**Goals:**
- work item 以 git repo root 為 key 獨立儲存
- 非 git 目錄以 CWD 為 key（不強制要 git）
- API 向後不相容（新增 project 參數）但在同一個 package 內修改

**Non-Goals:**
- 遷移舊全域 `~/.tt/work-item`（無法安全對應 project）
- 支援手動指定 project key（YAGNI）
- 跨 project 共用 work item

## Decisions

### 1. 儲存路徑：`~/.tt/work-items/<sha256[:16]>`

用 SHA-256 前 16 hex 字元作為目錄項目名稱，輸入為 resolved project path（絕對路徑）。

**Why SHA-256 hash over raw path:**
- 避免路徑特殊字元問題（空格、/）
- 固定長度，目錄列舉清楚
- 16 hex 字元 = 64-bit prefix，碰撞機率可忽略（一台機器不會有 2^32 個 project）

**Alternatives considered:**
- base64 encode raw path → 可讀但長度不固定，含 `/`
- 直接用 path 做子目錄 → 需要 mkdir -p，且 `~` 本身路徑很長

### 2. Project key resolution：git root 優先

```go
func resolveProject(dir string) string {
    out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
    if err != nil { return dir }
    return strings.TrimSpace(string(out))
}
```

**Why git root over sub-directory:**
- git root 是自然的 project 邊界
- 在 repo 內任何子目錄執行，key 一致
- 非 git 目錄 fallback 到 CWD，不中斷功能

### 3. API 簽名：新增 `project string` 參數

```go
func Get(project string) (string, error)
func Set(label, project string) error
func Clear(project string) error
```

Resolution（`resolveProject`）在 package 內部進行，呼叫端只需傳 CWD。

**Why pass CWD vs. resolve at call site:**
- 封裝 git 查詢邏輯在 workitem package，不洩漏到 recorder / cmd
- 測試時可注入任意 project key（不依賴真實 git repo）

### 4. 無遷移策略

舊 `~/.tt/work-item` 繼續存在但被忽略。遷移條件：
- 無法知道該 work item 屬於哪個 project
- 使用者 re-set 一次成本低

## Risks / Trade-offs

- **git binary 不存在** → `exec.Command` 失敗，fallback 到 CWD，行為正確
- **CWD 在 git submodule 內** → `rev-parse --show-toplevel` 回傳 submodule root，而非 parent repo root。這是預期行為，submodule 通常是獨立 project。
- **sha256 前 16 字元碰撞** → 機率極低（birthday bound ≈ 2^32 個 project），發生時兩個 project 共用同一 work item，不影響資料正確性（只是 label 串到另一個 project）

## Migration Plan

無需部署步驟。舊 `~/.tt/work-item` 不刪除、不讀取，靜默忽略。
