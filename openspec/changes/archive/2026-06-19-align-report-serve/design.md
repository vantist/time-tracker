## Context

The `tt` command-line tool provides a terminal-based text report (`tt report`) and a local web dashboard (`tt serve`). The terminal text report has fewer details than the web dashboard, and the web dashboard's project table displays a static `0` in the "Tokens" column because the underlying Go struct (`ProjectSummary`) does not aggregate nor serialize project-specific input and output token counts.

We need to align both representations and fix the dashboard rendering bugs.

## Goals / Non-Goals

**Goals:**
- Add `input_tokens` and `output_tokens` to the `ProjectSummary` Go struct and populate them during aggregation in `Query()`.
- Rewrite `FormatText` in `internal/report/report.go` to output structured ASCII/UTF-8 tables for the Daily breakdown, By Project, By Work Item, and detailed Sessions log.
- Fix the `By Project` table in the dashboard HTML to render the correct input/output tokens in a `Input / Output` format.
- Clean up `cmd/tt/report_cmd.go` by removing CLI-specific work item printing, letting `FormatText` handle all layout formatting.
- Preserve backward compatibility with existing tests by adding a `ByWorkItem` field to `Result` instead of changing function signatures.

**Non-Goals:**
- Do not introduce interactive features, sorting options, or pagination to the terminal report.
- Do not add support for CSV/HTML exports to the report command (keep formatting output to plain text and JSON).

## Decisions

### 1. Unified Formatting in `FormatText`
We will rewrite `FormatText` to construct the entire report layout (Summary, Tokens, Cost, Daily breakdown, By Project, By Work Item, and Sessions).
- **Alternatives Considered**: Keeping layout sections separate and letting the CLI print them sequentially.
- **Rationale**: Keeping all formatting in a single library function (`FormatText`) makes it highly cohesive, easy to unit test, and avoids duplication between CLI command packages and internal libraries.

### 2. State Propagation via `Result.ByWorkItem`
We will add `ByWorkItem` to the `Result` struct to signal if `--by-work-item` was requested.
- **Alternatives Considered**: Changing the signature of `FormatText` to accept options/flags.
- **Rationale**: Changing `FormatText` signature would break existing unit tests and mock implementations. Carrying this state in `Result` (which is returned by `Query()`) avoids breaking the API.

### 3. Base Folder Formatting in CLI
For Project names and session paths, `FormatText` will format the output using `filepath.Base` to keep the tabular width predictable in the terminal.
- **Alternatives Considered**: Printing full absolute paths.
- **Rationale**: Absolute paths on developer machines are long (e.g. `/Users/username/workspace/project-name`) and would disrupt terminal table alignment, making reports unreadable.

## Risks / Trade-offs

- **[Risk] Terminal Output Overflow** → If there are many projects or sessions, printing them all in the terminal will scroll the screen.
  - **Mitigation** → This aligns with typical developer CLI behavior. If users want to look at large datasets, they can run `tt serve` for a scrollable web UI, or specify `--since` to restrict the range.
