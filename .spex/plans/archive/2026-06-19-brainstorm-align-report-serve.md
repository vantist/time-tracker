# Align Report and Serve Formats

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

The CLI time-tracking tool `tt` currently has two ways of presenting usage reports:
1. `tt report`: Terminal text output formatted via `report.FormatText()`.
2. `tt serve`: An interactive web dashboard calling `report.HandleAPIReport()`.

There are several format mismatches between these two views:
- The terminal text report lacks `User Active` time and `Tokens` details in the project breakdown.
- The terminal report does not display the detailed `Sessions` log or the `Daily` stats breakdown.
- The web dashboard's `By Project` table contains a `Tokens` column which is currently broken (always renders `0`) because the Go struct `ProjectSummary` does not aggregate and expose input/output tokens.

## Decision

We will align `tt report` with the web dashboard by implementing a unified terminal layout containing all data sections (Summary, Tokens, Cost, Daily breakdown, By Project, By Work Item, and detailed Sessions log), and we will fix the web dashboard's token column by exposing input/output token counts in the `ProjectSummary` Go struct.

## Rationale

Having identical information in both terminal and web dashboard provides a consistent, high-fidelity developer experience. Developers will not need to launch the web interface just to see daily trends or verify specific session times/branches/models.

## Approach

1. **Backend Struct & Aggregation Update**: Add `InputTokens` and `OutputTokens` to `ProjectSummary` in `internal/report/report.go`. Update `Query()` to accumulate these values under the project-level map.
2. **Flag State Preservation**: Add `ByWorkItem` to `Result` struct, set via `Options.ByWorkItem` in `Query()`, to allow `FormatText` to know if the CLI `--by-work-item` was requested.
3. **FormatText Redesign**: Re-implement `report.FormatText` to render beautifully formatted ASCII/UTF-8 tables for the Daily breakdown, By Project, By Work Item, and detailed Sessions log, using aligned string formatting and `filepath.Base` for paths.
4. **Remove CLI-level Print**: Clean up `cmd/tt/report_cmd.go` to let `report.FormatText` handle all layout rendering.
5. **Dashboard UI Fix**: Update `internal/report/html.go` to format the `Tokens` column as `Input / Output` using the new struct fields.

## Design Notes

### ProjectSummary (in `internal/report/report.go`)
```go
type ProjectSummary struct {
	Project            string   `json:"project"`
	SessionsCount      int      `json:"sessions"`
	AgentTimeSec       int64    `json:"agent_time_seconds"`
	UserActiveTimeSec  int64    `json:"user_active_time_sec"`
	InputTokens        int64    `json:"input_tokens"`
	OutputTokens       int64    `json:"output_tokens"`
	CostUSD            *float64 `json:"cost_usd"`
}
```

### Result (in `internal/report/report.go`)
```go
type Result struct {
	// ... existing fields ...
	ByWorkItem bool // set in Query from opts.ByWorkItem
}
```

### Dashboard Project Table (in `internal/report/html.go`)
```javascript
tr.innerHTML = '<td>'+esc(p.project)+'</td><td>'+p.sessions+'</td><td>'+fmtTime(p.agent_time_seconds||0)+'</td><td>'+fmtTime(p.user_active_time_sec||0)+'</td><td>'+fmt(p.input_tokens)+' / '+fmt(p.output_tokens)+'</td><td>'+fmtCost(p.cost_usd)+'</td>';
```

## Insights to Capture

- `design.md`: Align terminal report format to match dashboard data columns.
- `proposal.md`: Align report CLI output format with web dashboard.
- `tasks.md`: Implement struct fields, rewrite FormatText, fix dashboard HTML, update cmd/tt print logic.

## Open Questions

None.
