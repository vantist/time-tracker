## 1. Struct Expansion & SQL Aggregation (TDD)

- [x] 1.1 Add CacheReadTokens and CacheCreationTokens to ProjectSummary, AgentSummary, GroupResult, SessionRow, and DailyStat structs in `internal/report/report.go`.
- [x] 1.2 Write a failing unit test in `internal/report/report_test.go` to verify that `Query` correctly populates the extended token fields (`CacheReadTokens` and `CacheCreationTokens`) on all query dimensions.
- [x] 1.3 Update the aggregation logic in `Query` inside `internal/report/report.go` to group, aggregate, and populate the new cache token fields. Verify the test passes.

## 2. CLI FormatText Updates & Output File Flag (TDD)

- [x] 2.1 Write failing unit tests in `internal/report/report_test.go` to verify that `FormatText` prints all four categories (Input, Output, Cache read, Cache create) for all tables by default.
- [x] 2.2 Update `FormatText` in `internal/report/report.go` to include cache token columns and display the 4-category breakdown for all tables. Verify the tests pass.
- [ ] 2.3 Write a failing unit test in `cmd/tt/setup_cmd_test.go` (or a new test file) to verify the new `--output` / `-o` flag in the report command writes report contents directly to a file with `0600` permissions.
- [ ] 2.4 Add standard flag `-o` / `--output` to `reportCmd` in `cmd/tt/report_cmd.go` and implement the file output write logic. Verify the tests pass.

## 3. Web Dashboard Tooltips & Tables (TDD)

- [ ] 3.1 Write a failing unit test in `internal/report/html_test.go` verifying that `/api/report` response contains the new token fields.
- [ ] 3.2 Add `.tooltip` CSS styles and update HTML table structures in `internal/report/html.go` to display a single "Tokens" column with hover tooltip containing the 4-category breakdown.
- [ ] 3.3 Update client-side javascript `render` function in `internal/report/html.go` to build interactive tooltips for By Project, By Agent, By Model & Role, By Work Item, and Sessions tables. Verify the test passes.
