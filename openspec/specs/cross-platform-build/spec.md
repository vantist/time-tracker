# cross-platform-build Specification

## Purpose
TBD - created by archiving change github-cicd-release. Update Purpose after archive.
## Requirements
### Requirement: tag push 觸發三平台矩陣編譯

推入符合 `v*` 格式的 tag 時，GitHub Actions SHALL 以矩陣策略同時編譯三個 binary：`darwin/amd64`、`darwin/arm64`、`windows/amd64`。

#### Scenario: tag v0.1.0 push

- **WHEN** git tag `v0.1.0` 被 push 至 GitHub
- **THEN** build workflow 啟動，三個 matrix job 各自以對應 GOOS/GOARCH 編譯 binary

### Requirement: binary 注入版本號

每個編譯出的 binary SHALL 以 `-ldflags "-X main.version=<tag>"` 注入版本，其中 `<tag>` 取自 `github.ref_name`。

#### Scenario: v0.1.0 tag build

- **WHEN** workflow 以 `GOOS=darwin GOARCH=arm64` 編譯，tag 為 `v0.1.0`
- **THEN** 產出的 `tt-darwin-arm64` 執行 `tt version` 輸出 `v0.1.0`

### Requirement: artifact 命名規則

編譯產出的檔案 SHALL 依平台命名：
- `tt-darwin-amd64`
- `tt-darwin-arm64`
- `tt-windows-amd64.exe`

#### Scenario: 三個 artifact 上傳至 Release

- **WHEN** 矩陣編譯完成
- **THEN** 三個 binary 以上述名稱附加至對應的 GitHub Release，可從 Release 頁面直接下載

### Requirement: 使用 golang Docker image 編譯

Build workflow SHALL 使用官方 `golang:1.21` 以上版本的 Docker image（或 actions/setup-go），不依賴 CGo 或平台特定 toolchain。

#### Scenario: 無 CGo 交叉編譯

- **WHEN** workflow 設定 `CGO_ENABLED=0`，`GOOS=darwin`，`GOARCH=arm64`，於 Linux runner 執行
- **THEN** 成功產出可在 darwin/arm64 執行的 binary

