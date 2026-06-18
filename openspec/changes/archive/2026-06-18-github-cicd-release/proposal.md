## Why

`tt` 目前無法分發給其他使用者，因為沒有 git remote、自動版本管理或跨平台 binary 發布流程。需要建立完整的 GitHub 發布鏈，讓使用者能透過一條指令安裝最新版本，並讓 `tt version` 確認安裝成功。

## What Changes

- 設定 git remote 指向 `https://github.com/vantist/time-checker.git`
- 新增 `tt version` 子命令，顯示由 ldflags 注入的版本號（預設 `dev`）
- 新增 `release-please` GitHub Action：每次 push main 分析 Conventional Commits，自動 bump SemVer 並開 Release PR
- 新增 `build` GitHub Action：merge Release PR 觸發 tag push → 矩陣編譯三平台 binary → 上傳至 GitHub Release
- 新增 `install.sh`（macOS/Linux）和 `install.ps1`（Windows）安裝腳本，自動偵測平台、下載最新 Release binary、放進 PATH

## Capabilities

### New Capabilities

- `version-command`：`tt version` 命令，印出 build 時注入的版本字串
- `github-release-automation`：Conventional Commits → SemVer 自動 tag + GitHub Release 流程
- `cross-platform-build`：GitHub Actions 矩陣編譯 darwin/amd64、darwin/arm64、windows/amd64 三平台執行檔
- `install-scripts`：macOS/Linux `install.sh` 和 Windows `install.ps1`，一鍵安裝最新版至 PATH

### Modified Capabilities

（無，此為純新增功能）

## Impact

- Affected code:
  - New: `cmd/tt/version.go`
  - New: `.github/workflows/release-please.yml`
  - New: `.github/workflows/build.yml`
  - New: `install.sh`
  - New: `install.ps1`
  - New: `release-please-config.json`
  - New: `.release-please-manifest.json`
  - Modified: `cmd/tt/main.go`（注入 version var 給 cobra root）

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-17-brainstorm-github-cicd.md`
