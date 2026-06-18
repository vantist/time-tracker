## Context

`tt` 是 Go CLI 工具，使用 `modernc/sqlite`（純 Go，無 CGo）。本地已有 5 個 Conventional Commits，git init 完成但無 remote。目標：GitHub public repo + 全自動版本發布鏈。

## Goals / Non-Goals

**Goals:**
- 建立 git remote 連結 GitHub
- Conventional Commits → SemVer 自動 tag（release-please）
- GitHub Actions 矩陣編譯三平台 binary 並附加至 Release
- `tt version` 顯示注入版本號
- `install.sh` / `install.ps1` 一鍵安裝

**Non-Goals:**
- Homebrew / Scoop / 其他 package manager 支援
- Code signing / notarization（macOS Gatekeeper 警告需用戶手動允許）
- Linux binary（現階段不需要）

## Decisions

### D1：版本管理用 release-please（非 semantic-release）

**選擇**：[googleapis/release-please-action](https://github.com/googleapis/release-please-action)

**理由**：
- 純 GitHub Action，不需要 Node.js runtime 或 npm CI step
- PR-based 流程：每次 push main → 更新 Release PR → 合 PR 才打 tag，給 maintainer 控制時機
- 自動維護 `CHANGELOG.md`

**替代方案棄用**：
- `semantic-release`：需要 npm install，CI 較重
- 手動 tag：不自動化，違背目標

**release-please 所需設定檔**：
```
release-please-config.json     ← 定義 release type（go）和 package 路徑
.release-please-manifest.json  ← 追蹤目前版本，初始值 {"." : "0.0.0"}
```

### D2：版本注入用 ldflags

Build 時：
```bash
go build -ldflags "-X main.version=v1.2.3" -o tt ./cmd/tt
```

`cmd/tt/version.go` 宣告 package-level var：
```go
var version = "dev"
```

CI 用 `${{ github.ref_name }}` 取 tag（形如 `v1.2.3`）。
本地開發不設定 ldflags → 印出 `dev`。

### D3：Build Action 觸發條件

觸發：`on: push: tags: ['v*']`

release-please merge Release PR 時自動打 tag，build action 被觸發。
**不在 push main 時 build**（避免每次 push 都產生未 versioned 的 binary）。

### D4：矩陣編譯設定

```yaml
strategy:
  matrix:
    include:
      - goos: darwin
        goarch: amd64
        artifact: tt-darwin-amd64
      - goos: darwin
        goarch: arm64
        artifact: tt-darwin-arm64
      - goos: windows
        goarch: amd64
        artifact: tt-windows-amd64.exe
```

`modernc/sqlite` 純 Go → 無需 CGO_ENABLED=1 或 cross-compiler toolchain。

### D5：Install Script 策略

**install.sh**（macOS/Linux）：
1. `curl -s https://api.github.com/repos/vantist/time-checker/releases/latest | grep tag_name` 取最新版
2. `uname -s`（Darwin）+ `uname -m`（arm64/x86_64）組合 artifact 名稱
3. 下載至 `/tmp/tt`，`chmod +x`，`mv /usr/local/bin/tt`
4. 確認 `tt version` 輸出

**install.ps1**（Windows）：
1. `Invoke-RestMethod` 取 latest release tag
2. `$env:PROCESSOR_ARCHITECTURE`（AMD64）選 artifact
3. 下載至 `$env:USERPROFILE\bin\tt.exe`
4. 提示加入 PATH（若 `$env:USERPROFILE\bin` 不在 PATH）

**Public repo → 不需要 token**，curl 直接打 GitHub API 即可。

## Risks / Trade-offs

- **macOS Gatekeeper**：用戶第一次執行時會被 quarantine 阻擋。緩解：install.sh 加 `xattr -d com.apple.quarantine /usr/local/bin/tt` 步驟，或文件說明手動允許。
- **release-please 第一次 PR**：因為 `.release-please-manifest.json` 初始設 `0.0.0`，合 PR 後會打 `v0.1.0`（5 個現有 feat commits）。預期行為，無需處理。
- **ldflags 變數路徑**：`version` var 在 `cmd/tt/version.go` package `main`，ldflags 路徑必須是 `main.version`（不是 `github.com/user/tt/cmd/tt.version`）。

## Migration Plan

1. 設 git remote → push main（現有 5 commits）
2. 加設定檔和 workflow 檔案 → commit（`chore: add GitHub Actions release pipeline`）
3. Push main → release-please 偵測現有 feat commits → 開第一個 Release PR（v0.1.0）
4. Review PR → merge → 觸發 tag v0.1.0 → build action 跑矩陣編譯 → 三個 artifact 附加至 GitHub Release
5. 測試 install.sh 安裝流程

**Rollback**：刪除 workflow 檔案即可停用 CI，不影響 binary 本身。
