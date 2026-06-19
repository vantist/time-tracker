## 1. Test Setup (TDD)

- [x] 1.1 Update test data in `internal/report/report_test.go` to include mock projects and sessions with input/output tokens.
- [x] 1.2 Write a failing unit test in `internal/report/report_test.go` to verify that `Query()` returns correct `input_tokens` and `output_tokens` inside `ProjectSummary` elements.
- [x] 1.3 Write a failing unit test in `internal/report/report_test.go` to verify that `FormatText()` output contains Daily breakdown, By Project, By Work Item, and Sessions Log tables aligned properly.

## 2. Data Structure & Query Modifications

- [x] 2.1 Add `InputTokens` and `OutputTokens` fields to `ProjectSummary` in `internal/report/report.go`.
- [x] 2.2 Add `ByWorkItem` boolean field to `Result` in `internal/report/report.go`.
- [x] 2.3 Update `projState` struct in `Query()` to include `inputTokens` and `outputTokens`.
- [x] 2.4 Update `Query()` main row aggregation loop to sum input and output tokens for projects.
- [x] 2.5 Populate `InputTokens` and `OutputTokens` when constructing `res.ByProject` and set `res.ByWorkItem = opts.ByWorkItem` in `Query()`. Verify that the query token tests pass.

## 3. FormatText Reconstruction

- [x] 3.1 Rewrite `FormatText` in `internal/report/report.go` to output ASCII table headers and aligned columns for the Summary, Tokens, and Cost blocks.
- [x] 3.2 Add the Daily breakdown table section in `FormatText`.
- [x] 3.3 Add the By Project table section in `FormatText`, formatting project paths with `filepath.Base`.
- [x] 3.4 Add the By Work Item table section in `FormatText`, showing it if there is more than 1 group or if `Result.ByWorkItem` is true.
- [x] 3.5 Add the detailed Sessions log table section in `FormatText`, formatting date/time to local time string and project paths with `filepath.Base`. Verify that all `FormatText` unit tests pass.

## 4. CLI and Dashboard Web Updates

- [x] 4.1 Update `cmd/tt/report_cmd.go` to remove custom CLI printing logic for work item groupings, relying solely on `report.FormatText()`.
- [x] 4.2 Update the dashboard HTML template in `internal/report/html.go` to format the Tokens column in the By Project table using `input_tokens` and `output_tokens`.

## 5. Verification

- [x] 5.1 Run `go test ./...` to verify all tests pass successfully.
- [x] 5.2 Build the `tt` CLI and verify terminal output and local dashboard formatting manually.
