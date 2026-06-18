## 1. Git Remote 設定

- [x] 1.1 執行 `git remote add origin https://github.com/vantist/time-checker.git`
- [ ] 1.2 執行 `git push -u origin main`，確認現有 5 個 commits 上傳成功

## 2. version 命令

- [x] 2.1 新增 `cmd/tt/version.go`：宣告 `var version = "dev"` 及 `versionCmd` cobra command，執行後印出 `version`
- [x] 2.2 在 `cmd/tt/main.go` 的 `init()` 或 `main()` 中將 `versionCmd` 加入 `rootCmd`
- [x] 2.3 本地執行 `go build ./cmd/tt && ./tt version` 驗證輸出 `dev`
- [x] 2.4 本地執行 `go build -ldflags "-X main.version=v0.1.0-test" ./cmd/tt && ./tt version` 驗證輸出 `v0.1.0-test`

## 3. release-please 設定

- [x] 3.1 新增 `release-please-config.json`，設定 `release-type: go`，package path 為 `.`
- [x] 3.2 新增 `.release-please-manifest.json`，內容 `{"." : "0.0.0"}`
- [x] 3.3 新增 `.github/workflows/release-please.yml`：trigger `on: push: branches: [main]`，使用 `googleapis/release-please-action@v4`，token 用 `${{ secrets.GITHUB_TOKEN }}`

## 4. Build workflow

- [x] 4.1 新增 `.github/workflows/build.yml`：trigger `on: push: tags: ['v*']`
- [x] 4.2 設定矩陣策略，include darwin/amd64、darwin/arm64、windows/amd64，各自對應 artifact 名稱
- [x] 4.3 每個 matrix job 執行 `go build -ldflags "-X main.version=${{ github.ref_name }}" -o <artifact> ./cmd/tt`，設定 `CGO_ENABLED=0`
- [x] 4.4 使用 `softprops/action-gh-release` 或 `gh release upload` 將三個 binary 附加至對應 Release

## 5. Install Scripts

- [x] 5.1 新增 `install.sh`：偵測 `uname -s`（Darwin）與 `uname -m`（arm64/x86_64），組出 artifact 名稱；從 GitHub Releases API 取最新 tag；下載 binary 至 `/tmp/tt`；`chmod +x`；`mv /usr/local/bin/tt`；執行 `xattr -d com.apple.quarantine`（macOS only）；印出 `tt version`
- [x] 5.2 不支援的平台（非 darwin）印出 `Unsupported platform: <os>/<arch>` 並 exit 1
- [x] 5.3 新增 `install.ps1`：`Invoke-RestMethod` 取 latest tag；組出 `tt-windows-amd64.exe`；下載至 `$env:USERPROFILE\bin\tt.exe`；若 `$env:USERPROFILE\bin` 不在 PATH 則印出提示；印出 `tt version`

## 6. 整合驗證

- [x] 6.1 commit 所有新增檔案（message: `chore: add GitHub Actions release pipeline and install scripts`）
- [ ] 6.2 push main → 確認 release-please workflow 在 GitHub Actions 成功執行
- [ ] 6.3 在 GitHub Actions 頁面確認 release-please 開出 Release PR（含 v0.1.0 版號）
- [ ] 6.4 merge Release PR → 確認 tag v0.1.0 被打，build workflow 觸發
- [ ] 6.5 確認三個 binary（tt-darwin-amd64、tt-darwin-arm64、tt-windows-amd64.exe）出現在 GitHub Release 頁面
- [ ] 6.6 在 macOS 執行 `curl -fsSL https://raw.githubusercontent.com/vantist/time-checker/main/install.sh | sh`，確認安裝成功並 `tt version` 輸出 `v0.1.0`
