# github-release-automation Specification

## Purpose
TBD - created by archiving change github-cicd-release. Update Purpose after archive.
## Requirements
### Requirement: push main 觸發 release-please 分析

每次 push 至 main branch，GitHub Actions SHALL 執行 release-please action，分析自上次 tag 以來的 Conventional Commits，依規則決定是否更新 Release PR。

#### Scenario: feat commit 推入 main

- **WHEN** push main 包含至少一個 `feat:` commit
- **THEN** release-please 開啟或更新一個標題為 `chore(main): release v<MINOR+1>.0` 的 PR，並更新 CHANGELOG.md

#### Scenario: 僅 chore commit 推入 main

- **WHEN** push main 僅包含 `chore:` 或 `docs:` commits
- **THEN** release-please 不開新 PR，不更改版本號

### Requirement: SemVer bump 規則符合 Conventional Commits

release-please SHALL 依以下規則 bump 版本號：

| Commit type | Bump |
|-------------|------|
| `fix:` | patch |
| `feat:` | minor |
| `feat!:` 或含 `BREAKING CHANGE:` footer | major |

#### Scenario: fix commit → patch bump

- **WHEN** Release PR 包含 `fix:` commit，無 `feat:` 或 breaking change
- **THEN** 版本號從 `vX.Y.Z` bump 至 `vX.Y.(Z+1)`

#### Scenario: feat commit → minor bump

- **WHEN** Release PR 包含 `feat:` commit，無 breaking change
- **THEN** 版本號從 `vX.Y.Z` bump 至 `vX.(Y+1).0`

### Requirement: merge Release PR 打 tag 並建立 GitHub Release

合併 release-please 開的 Release PR 後，release-please SHALL 自動打 git tag（格式 `v<MAJOR>.<MINOR>.<PATCH>`）並在 GitHub 建立 Release draft。

#### Scenario: merge Release PR

- **WHEN** 使用者 merge release-please 建立的 Release PR
- **THEN** repository 出現新 tag（如 `v0.1.0`），GitHub Releases 頁面出現對應 Release

