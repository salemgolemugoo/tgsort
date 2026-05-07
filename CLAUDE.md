# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`tgsort` is a Go CLI tool that sorts blocks and attributes in Terragrunt HCL files, modifying them in-place (like `gofmt`). It supports stdin/stdout mode and is configurable via a `.tgsort` TOML file in the working directory.

## Commands

```bash
# Build
go build ./...

# Run unit tests (all packages)
go test ./...

# Run a single package's tests
go test ./internal/sorter/...

# Run integration tests (build binary + golden file fixtures)
go test -tags integration ./...

# Run with race detector
go test -race ./...

# Install locally
go install .
```

## Architecture

```
main.go              # Entry point — delegates to cmd.Execute()
cmd/root.go          # Cobra CLI: flag parsing, mode dispatch (file/dir/stdin)
internal/
  config/config.go   # Loads .tgsort TOML; returns defaults if file is absent
  sorter/sorter.go   # Core sorting logic using hclsyntax
  walker/walker.go   # File/dir/stdin I/O; calls sorter.Sort per file
```

**Data flow:** `cmd/root.go` → `walker.Walker` → `sorter.Sort` → write back to disk (or diff to stdout in dry-run).

**Sorter internals (`sorter.go`):**
- Parses HCL with `hclsyntax.ParseConfig` to get node positions.
- Uses line-based text manipulation (not `hclwrite`) to preserve formatting exactly.
- `extractItems` splits the file into a header, footer, and list of `hclItem` values — each item carries its raw text plus any immediately-preceding comment lines (`commentStart`).
- `sortItems` orders items by `block_order` priority, then by type name, then by first label.
- For types listed in `sort_attributes_in` (default: `["inputs"]`), `sortItemAttributes` additionally sorts keys inside the block/object literal.
- `reconstruct` joins items with one blank line between them and re-attaches the header/footer.

## Configuration

`.tgsort` in the working directory (TOML, optional):

```toml
block_order = ["terraform", "remote_state", "include", "locals", "generate", "dependency", "inputs"]
sort_attributes_in = ["inputs"]
```

Absent fields keep their defaults. An absent file is silently ignored; a malformed file is a fatal error.

## Testing

- **Unit tests**: table-driven, in `internal/*/` packages alongside the source.
- **Integration tests**: `integration_test.go` at the repo root, build-tagged `integration`. They compile the binary, run it against `testdata/{full,comments}/input.hcl`, and compare output to `expected.hcl` golden files. Add new fixtures by creating a `testdata/<name>/` directory with `input.hcl` and `expected.hcl`.

## Key behaviors / invariants

- `.hcl.json` files are rejected outright.
- `--recursive` is incompatible with stdin mode (`-`).
- `--dry-run` exits 1 if any file would change; prints a unified diff to stdout without modifying files.
- Comments on lines immediately preceding a block/attribute travel with that node during reordering.
- Single-line object expressions (`inputs = { key = "v" }`) are left unsorted.
- Trailing content after the last block is preserved as a footer.
