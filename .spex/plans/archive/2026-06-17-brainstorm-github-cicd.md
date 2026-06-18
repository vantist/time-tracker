---
name: github-cicd
description: GitHub remote + release-please 自動 tag + GitHub Actions 跨平台 build + install script
metadata:
  type: project
---

## 目標

- push 到 GitHub (https://github.com/vantist/time-checker.git)
- 依 Conventional Commits 自動 bump 版本 → tag → GitHub Release
- Release 自動附上三平台 binary
- install script 抓最新版，裝完可 `tt version` 確認

## 現況

- 本地：已有 `git init`，5 個 commits，無 remote
- Commits 已遵循 Conventional Commits（fix:, feat: 等）
- 語言：Go 1.26，`modernc/sqlite`（無 CGo），可直接交叉編譯

## 架構

```
push main
    │
    ▼
release-please Action
    ├─ 分析 commit type (fix/feat/feat!)
    ├─ bump MAJOR.MINOR.PATCH
    └─ 開 Release PR（更新 CHANGELOG, version bump）

merge Release PR
    │
    ▼
release-please 打 tag (v1.2.3)
    │
    ▼
build Action (triggered by tag push)
    ├─ go build -ldflags "-X main.version=v1.2.3" GOOS=darwin GOARCH=amd64
    ├─ go build -ldflags "-X main.version=v1.2.3" GOOS=darwin GOARCH=arm64
    ├─ go build -ldflags "-X main.version=v1.2.3" GOOS=windows GOARCH=amd64
    └─ gh release upload → 附到 GitHub Release
```

## 版本注入

```go
// cmd/tt/main.go
var version = "dev"  // 由 ldflags 覆寫

// tt version 命令印出 version
```

```makefile
go build -ldflags "-X main.version=${VERSION}" -o tt ./cmd/tt
```

CI 用 `${{ github.ref_name }}` 取 tag 名稱。

## GitHub Actions 檔案

- `.github/workflows/release-please.yml`：release-please Action
- `.github/workflows/build.yml`：on tag push → matrix build → upload to release

## Conventional Commits → SemVer 規則

| Commit | bump |
|--------|------|
| `fix:` | patch |
| `feat:` | minor |
| `feat!:` / `BREAKING CHANGE:` | major |
| `chore:`, `docs:`, `test:` | 不 bump（不觸發 Release PR） |

## Install Script

`install.sh`（macOS/Linux）：
1. `curl` GitHub Releases API 取最新 tag
2. 偵測 `$(uname -s)` + `$(uname -m)` 選對應 binary
3. `chmod +x` + `mv /usr/local/bin/tt`
4. `tt version` 驗證

`install.ps1`（Windows）：類似邏輯，`$env:PROCESSOR_ARCHITECTURE` 偵測，放到 `$env:USERPROFILE\bin\`。

## 實作步驟

1. `git remote add origin https://github.com/vantist/time-checker.git`
2. 新增 `cmd/tt/version.go`（version var + cobra command）
3. 新增 `.github/workflows/release-please.yml`
4. 新增 `.github/workflows/build.yml`
5. 新增 `install.sh` + `install.ps1`
6. `git push -u origin main`

## Open Questions

- [ ] GitHub repo 是 public 還是 private？（影響 install script curl 是否需要 token）
- [ ] release-please 需要 GITHUB_TOKEN（自動提供）或 PAT？若 Actions 有 branch protection rule 需 PAT
