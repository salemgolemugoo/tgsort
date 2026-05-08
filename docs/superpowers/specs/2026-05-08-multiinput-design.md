# Multi-file Input Support

**Date:** 2026-05-08  
**Branch:** multiinput  
**Problem:** When tgsort is used as a pre-commit hook, pre-commit passes all staged files as separate positional arguments. tgsort currently enforces `MaximumNArgs(1)` and fails with "accepts at most 1 arg(s), received N".

---

## CLI Behavior

`Use` string: `tgsort [file_or_directory...|-] [flags]`

Argument validation changes from `cobra.MaximumNArgs(1)` to `cobra.ArbitraryArgs` with manual validation in `run()`.

| Args | Behavior |
|------|----------|
| 0 | Sort current directory (unchanged) |
| 1, arg is `-` | Read from stdin, write to stdout (unchanged) |
| 1, arg is file | Sort that file (unchanged) |
| 1, arg is dir | Sort that dir, respecting `--recursive` (unchanged) |
| 2+, any is `-` | Error: stdin cannot be combined with other args |
| 2+ files/dirs | Process each in order, stop on first error |

- `--recursive` remains valid when 2+ args include directories.
- Dry-run exit-1 triggers if *any* file across all args would change.
- On error, stop immediately (fail-fast, no collect-all).

---

## Code Changes

**Only `cmd/root.go` changes** — `walker.go` already has `ProcessFile` and `ProcessDir` which handle individual paths correctly.

Changes in `run()`:
1. Replace the single `target` string with a loop over `args`.
2. Manual validation at top: if any arg is `-` and `len(args) > 1`, return an error.
3. The `target == ""` (no-args) branch becomes `len(args) == 0`.
4. The file/dir dispatch becomes a loop body, calling `w.ProcessFile` or `w.ProcessDir` for each arg, returning on first error.
5. `hasChanges` accumulates across all args; `os.Exit(1)` fires at the end if `dryRun && hasChanges`.

No changes to `walker.go`, `sorter.go`, `config.go`, or any test fixtures.

---

## Testing

Two new integration tests in `integration_test.go`:

1. **`TestIntegration_MultipleFiles`** — writes two unsorted HCL files, runs `tgsort file1 file2`, checks both are sorted correctly.
2. **`TestIntegration_StdinWithOtherArgs_Fails`** — runs `tgsort - somefile.hcl`, expects non-zero exit.

No new unit tests needed — the dispatch logic in `run()` is thin glue; sorting logic already has unit coverage.
