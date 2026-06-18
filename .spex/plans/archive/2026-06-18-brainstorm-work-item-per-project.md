# Brainstorm: Work Item Per-Project Scope

**Date**: 2026-06-18
**Status**: Converged

## Intent

`tt work` currently stores a single global work item at `~/.tt/work-item`.
Goal: make it per-project, so different repos can track different work items independently.

## Design Conclusion

**Decision**: Store per-project work items keyed by git root path.
**Rationale**: Git root is the natural project boundary — sub-directory `cd` doesn't fragment the key, and non-git dirs fall back to raw CWD.
**Approach selected**: `~/.tt/work-items/<sha256[:16]-of-resolved-project-path>`, one file per project.

## Key Decisions

### Storage path
`~/.tt/work-items/<sha256[:16]>` — hex prefix of SHA-256 of the resolved project path.
Content: just the label string with trailing newline (same as current).

### Project key resolution
Add `resolveProject(dir string) string` inside `workitem` package:
1. Run `git -C dir rev-parse --show-toplevel`
2. On success → use git root
3. On failure → use `dir` as-is (non-git projects still work)

### API changes
```go
func Get(project string) (string, error)
func Set(label, project string) error
func Clear(project string) error
```

### `tt work` CLI
Pass `os.Getwd()` as project to all three functions. Resolution happens inside the package.

### `recorder.RecordPrompt`
`input.Project` is already CWD from hook. Pass it to `workitem.Get(input.Project)` — resolution normalizes to git root.

### Migration / backward compat
**No migration.** Old `~/.tt/work-item` is left alone and ignored. The global file had no project association, so no safe way to guess which project it belongs to. Users re-set after upgrade.

## Scope

Files to change:
- `internal/workitem/workitem.go` — add `resolveProject`, update `Get`/`Set`/`Clear` signatures
- `internal/workitem/workitem_test.go` — update tests for new signatures
- `cmd/tt/work.go` — pass `os.Getwd()` to all calls
- `internal/recorder/recorder.go` — pass `input.Project` to `workitem.Get`

## Open Questions

None.
