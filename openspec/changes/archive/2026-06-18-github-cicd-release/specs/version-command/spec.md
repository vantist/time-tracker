## ADDED Requirements

### Requirement: tt version 命令存在且可執行

`tt` CLI SHALL 提供 `version` 子命令，執行後印出版本字串至 stdout 並以 exit code 0 結束。

#### Scenario: 開發環境執行（無 ldflags）

- **WHEN** 使用者執行 `tt version`，且 binary 未以 ldflags 注入版本號
- **THEN** stdout 印出 `dev`，exit code 為 0

#### Scenario: Release binary 執行（有 ldflags）

- **WHEN** 使用者執行 `tt version`，且 binary 以 `-ldflags "-X main.version=v1.2.3"` 編譯
- **THEN** stdout 印出 `v1.2.3`，exit code 為 0

### Requirement: version 變數在 package main 宣告

版本字串 SHALL 以 package-level var `version` 宣告於 `cmd/tt/version.go`，初始值為 `"dev"`，供 `-ldflags "-X main.version=<tag>"` 覆寫。

#### Scenario: ldflags 覆寫正確路徑

- **WHEN** CI 執行 `go build -ldflags "-X main.version=v0.1.0" ./cmd/tt`
- **THEN** 編譯後的 binary 執行 `tt version` 輸出 `v0.1.0`
