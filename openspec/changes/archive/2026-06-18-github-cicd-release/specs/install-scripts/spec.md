## ADDED Requirements

### Requirement: install.sh 自動偵測平台並安裝最新版

`install.sh` SHALL 自動偵測作業系統與架構，從 GitHub Releases 下載最新 binary，放至 `/usr/local/bin/tt`，並設定執行權限。

#### Scenario: macOS Apple Silicon 安裝

- **WHEN** 使用者在 darwin/arm64 機器執行 `curl -fsSL <url>/install.sh | sh`
- **THEN** 下載 `tt-darwin-arm64`，chmod +x，移至 `/usr/local/bin/tt`，最後印出安裝的版本號

#### Scenario: macOS Intel 安裝

- **WHEN** 使用者在 darwin/amd64 機器執行 `curl -fsSL <url>/install.sh | sh`
- **THEN** 下載 `tt-darwin-amd64`，chmod +x，移至 `/usr/local/bin/tt`，最後印出安裝的版本號

#### Scenario: 不支援的平台

- **WHEN** 使用者在 Linux 或其他不支援的平台執行 install.sh
- **THEN** 印出錯誤訊息 `Unsupported platform: <os>/<arch>`，exit code 為 1

### Requirement: install.sh 移除 macOS quarantine

install.sh 在 macOS 上 SHALL 執行 `xattr -d com.apple.quarantine /usr/local/bin/tt`，移除 Gatekeeper quarantine 屬性，避免首次執行被阻擋。

#### Scenario: 移除 quarantine 屬性

- **WHEN** 安裝完成後，在 macOS 執行 xattr 移除指令
- **THEN** `xattr -l /usr/local/bin/tt` 不再顯示 `com.apple.quarantine`

### Requirement: install.ps1 自動偵測平台並安裝最新版

`install.ps1` SHALL 從 GitHub Releases 下載最新 `tt-windows-amd64.exe`，儲存至 `$env:USERPROFILE\bin\tt.exe`。

#### Scenario: Windows x64 安裝

- **WHEN** 使用者在 Windows x64 以 PowerShell 執行 `install.ps1`
- **THEN** 下載 `tt-windows-amd64.exe`，儲存至 `$env:USERPROFILE\bin\tt.exe`，並印出安裝的版本號

#### Scenario: PATH 提示

- **WHEN** `$env:USERPROFILE\bin` 不在 `$env:PATH`
- **THEN** install.ps1 印出提示，說明如何將該目錄加入 PATH，不自動修改系統設定

### Requirement: install script 驗證安裝結果

安裝完成後，install script SHALL 執行 `tt version` 並印出輸出，讓使用者確認安裝成功。

#### Scenario: 安裝驗證

- **WHEN** binary 成功放至目標路徑後
- **THEN** script 執行 `tt version`，輸出如 `v0.1.0`，exit code 為 0
