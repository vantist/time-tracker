## Why

The time-tracking tool `tt` has formatting and data discrepancies between `tt report` (terminal text) and `tt serve` (web dashboard). The terminal output lacks detailed tables (Daily, By Project, By Work Item, Sessions log) and the web dashboard shows 0 tokens in the Project table due to a lack of backend data mapping. Aligning both formats improves developer experience.

## What Changes

- Update `tt report` terminal text output to include Daily, By Project, By Work Item, and detailed Sessions log tables.
- Add `input_tokens` and `output_tokens` fields to the `ProjectSummary` struct and populate them during aggregation in `Query()`.
- Fix the `By Project` table in the web dashboard to format and display `input_tokens / output_tokens` instead of a static `0`.
- Remove CLI-level work item custom printing from `cmd/tt/report_cmd.go` and delegate all formatting to `report.FormatText()`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `report-text-tokens`: Align the terminal report text format to include all data tables.
- `web-dashboard`: Update the Project table to format and display the new token fields.

## Impact

- Affected specs:
  - `openspec/specs/report-text-tokens/spec.md`
  - `openspec/specs/web-dashboard/spec.md`
- Affected code:
  - Modified:
    - `internal/report/report.go`
    - `internal/report/html.go`
    - `cmd/tt/report_cmd.go`
    - `internal/report/report_test.go`

## Source

Derived from brainstorm plan: `.spex/plans/2026-06-19-brainstorm-align-report-serve.md`

## Implementation Approach

Testing strategy: TDD — write failing tests before each implementation unit.
