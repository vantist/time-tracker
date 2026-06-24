# VS Code Extension Build & Distribution Pipeline

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

`vscode-extension/` 已有 `tt-copilot-bridge` VS Code extension 程式碼（TypeScript），但目前：
1. CI（`build.yml`）只 build Go binaries，沒有打包 `.vsix`
2. `SetupVSCodeCopilot()` 只印手動安裝指示，沒有自動安裝
3. Extension 版本（`0.1.0`）與 `tt` 版本（`1.10.0`）不同步

使用者希望：在 CI 自動打包 `.vsix` 並上傳到 GitHub Release，`tt setup --vscode-copilot` 自動下載安裝。

## Decision

在 `build.yml` 新增 `build-extension` job 打包 `.vsix` 上傳 GitHub Release；`SetupVSCodeCopilot()` 從 Release 下載 `.vsix` 並執行 `code --install-extension`。

## Rationale

GitHub Releases 路線最簡單：不侵入 Go build 流程、不需要額外帳號（如 Marketplace publisher）、.vsix 幾百 KB 下載無感。版本同步透過 CI 將 `package.json` version 設為 tag version 實現。

## Approach

### 1. CI Pipeline（build.yml）

新增 `build-extension` job，與 Go build 並行：

```yaml
build-extension:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4

    - uses: actions/setup-node@v4
      with:
        node-version: '20'

    - name: Sync extension version to tag
      run: |
        VERSION="${GITHUB_REF_NAME#v}"
        cd vscode-extension
        jq --arg v "$VERSION" '.version = $v' package.json > tmp.json && mv tmp.json package.json

    - name: Build Extension
      working-directory: vscode-extension
      run: |
        npm ci
        npm run compile
        npx @vscode/vsce package --no-dependencies

    - name: Upload VSIX to Release
      uses: softprops/action-gh-release@v2
      with:
        files: vscode-extension/tt-copilot-bridge-*.vsix
```

要點：
- Extension 版本與 `tt` 同步（tag `v1.10.0` → extension `1.10.0`）
- `vsce package --no-dependencies` 避免 vsce 嘗試安裝 npm dependencies（已在 `npm ci` 處理）
- `.vsix` 檔名格式: `tt-copilot-bridge-{version}.vsix`
- 需在 `vscode-extension/` 加 `node_modules/` 到 `.gitignore`（如尚未）

### 2. Go Setup — SetupVSCodeCopilot()

改寫 `internal/setup/vscode_copilot.go` 的 `SetupVSCodeCopilot()`：

```
SetupVSCodeCopilot()
  │
  ├── 1. 確認 code CLI 可用（findVSCodePath）
  │      └── 不可用 → return error
  │
  ├── 2. 取得 tt 當前版本（main.version, ldflags 注入）
  │
  ├── 3. 下載 .vsix
  │      ├── 嘗試精確版本:
  │      │   GET https://github.com/vantist/time-tracker/releases/download/v{version}/tt-copilot-bridge-{version}.vsix
  │      ├── Fallback 到 latest:
  │      │   GET https://api.github.com/repos/vantist/time-tracker/releases/latest
  │      │   → 解析 assets 找 tt-copilot-bridge-*.vsix → 下載
  │      └── 都失敗 → return error（fallback 到手動指示）
  │
  ├── 4. 執行 code --install-extension <path-to-vsix>
  │
  └── 5. 清理暫存 .vsix（os.Remove）
```

關鍵設計：
- **版本**: 從 `main.version`（build 時 `-ldflags "-X main.version=..."` 注入）讀取
- **下載**: 使用 `net/http` 下載到 `os.TempDir()`
- **Fallback**: 精確版本 → latest release → 手動安裝指示
- **安全性**: 下載後不驗證 checksum（.vsix 來自自己的 GitHub Release，信任鏈明確）

### 3. 版本同步機制

- `go build` 時: `-ldflags "-X main.version=$VERSION"` 注入版本
- CI build-extension job: `jq` 將 `package.json` version 設為同版本
- 確保 `tt setup --vscode-copilot` 下載的 `.vsix` 與 `tt` 二進位版本一致

## Design Notes

### 檔案異動

| 檔案 | 變更 |
|------|------|
| `.github/workflows/build.yml` | 新增 `build-extension` job |
| `internal/setup/vscode_copilot.go` | 重寫 `SetupVSCodeCopilot()` 為自動下載安裝 |
| `internal/setup/vscode_copilot_test.go` | 新增測試（mock HTTP server 模擬 GitHub Release） |
| `vscode-extension/package.json` | 版本號由 CI 自動同步，不需手動改 |

### 不需要改的

- `cmd/tt/setup_cmd.go` — 已有 `vscode-copilot` flag 和 toolInfo entry
- `vscode-extension/src/extension.ts` — Extension 邏輯不變
- `vscode-extension/tsconfig.json` — build config 不變

### 錯誤處理

- `code` CLI 不可用 → 明確錯誤訊息
- 網路下載失敗 → fallback 到 latest → 最終 fallback 到手動指示
- `.vsix` 安裝失敗 → 傳播 error

## Insights to Capture

- `proposal.md`: CI 打包 VS Code Extension 到 GitHub Release + setup 自動安裝
- `tasks.md`:
  - 修改 `build.yml` 新增 `build-extension` job（Node.js setup + vsce package + upload）
  - 重寫 `SetupVSCodeCopilot()` 為自動下載安裝（含精確版本 + latest fallback）
  - 新增 `SetupVSCodeCopilot` 的 unit test（mock HTTP）
  - 確認 `vscode-extension/.gitignore` 包含 `node_modules/` 和 `out/`
  - 端到端測試：手動觸發 workflow 確認 .vsix 上傳到 Release

## Open Questions

（無 — 完全收斂）
