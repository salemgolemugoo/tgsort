# tgsort Design Spec

**Date:** 2026-05-07
**Status:** Approved

## Overview

`tgsort` is a Go CLI tool that sorts blocks and attributes in HCL Terragrunt files. It modifies files in-place (like `gofmt`), supports stdin/stdout mode, and is configurable via a TOML config file.

## CLI Interface

```
tgsort [file_or_directory|-] [flags]
```

**Flags:**
- `-d`, `--dry-run` — print a unified diff to stdout; exit 1 if changes would be made (useful for CI)
- `-r`, `--recursive` — recurse into subdirectories when a directory is given
- `-h`, `--help` — print help
- `-v`, `--version` — print version

**Invocation modes:**
- `tgsort path/to/file.hcl` — sort and overwrite in place
- `tgsort path/to/dir` — sort all `.hcl` files in that directory (non-recursive)
- `tgsort path/to/dir -r` — sort all `.hcl` files recursively
- `tgsort -` — read from stdin, write sorted output to stdout
- `tgsort - -r` — error: incompatible flags

**File support:** `.hcl` files only. `.hcl.json` files are not supported.

## Project Structure

```
tgsort/
├── main.go
├── cmd/
│   └── root.go          # cobra CLI definition, flag parsing
├── internal/
│   ├── config/
│   │   └── config.go    # .tgsort TOML loader
│   ├── sorter/
│   │   └── sorter.go    # hclwrite-based sort logic
│   └── walker/
│       └── walker.go    # file/dir/stdin input handling
├── go.mod
└── go.sum
```

**Dependencies:**
- `github.com/spf13/cobra` — CLI flag parsing
- `github.com/hashicorp/hcl/v2/hclwrite` — format-preserving HCL manipulation
- `github.com/BurntSushi/toml` — config file parsing

## Configuration

The `.tgsort` config file is looked up in the current working directory (repo root). If absent, defaults apply silently.

**Format:** TOML

**Default config (implicit when `.tgsort` is absent):**

```toml
block_order = [
  "terraform",
  "remote_state",
  "include",
  "locals",
  "generate",
  "dependency",
  "inputs",
]

sort_attributes_in = ["inputs"]
```

Both fields are optional — omitting `block_order` uses the default order; omitting `sort_attributes_in` uses `["inputs"]`.

## Sorting Logic

### Block ordering

Blocks are sorted according to `block_order`:

1. Blocks are grouped by type and ordered per the `block_order` list.
2. Multiple blocks of the same type (e.g., several `dependency` blocks) are sorted alphabetically by their label.
3. Block types not present in `block_order` are placed after all listed types, sorted alphabetically among themselves.

### Attribute sorting

Within any block type listed in `sort_attributes_in`, attributes are sorted alphabetically by key.

`inputs` is a special case: it is defined as an attribute assignment with a map value (`inputs = { ... }`), not an HCL block. When `inputs` appears in `sort_attributes_in`, the map keys within its value are sorted alphabetically.

For positioning purposes, `inputs` in `block_order` controls where the `inputs = { ... }` attribute appears relative to the surrounding blocks — it is treated as a named node in the ordering even though it is technically an attribute rather than a block.

### Comment attachment

Using `hclwrite`'s token model, comments on the line(s) immediately preceding a block or attribute are treated as attached to that node and travel with it during reordering. Comments do not stay in their original position.

**Example:**

Input:
```hcl
# Configures networking
dependency "vpc" {
  config_path = "../vpc"
}

# Sets up the k8s connection
dependency "eks" {
  config_path = "../eks"
}
```

Output (sorted alphabetically by label within `dependency` type):
```hcl
# Sets up the k8s connection
dependency "eks" {
  config_path = "../eks"
}

# Configures networking
dependency "vpc" {
  config_path = "../vpc"
}
```

### Blank lines

One blank line is preserved between top-level blocks. Blank lines within a block body are preserved as-is.

## Error Handling

All errors cause an immediate exit with a non-zero exit code and a message printed to stderr:

| Situation | Behaviour |
|-----------|-----------|
| Invalid HCL syntax | Print error with file path and line number, exit non-zero |
| File permission error | Print error with file path, exit non-zero |
| Config file parse error | Print error, exit non-zero |
| stdin with `-r` flag | Print error (incompatible flags), exit non-zero |

## Testing

- **Unit tests** (`sorter_test.go`): table-driven tests using input/expected HCL string pairs. Covers block reordering, attribute sorting within `inputs`, comment attachment, mixed block types, and unlisted block types placed at the end.
- **Integration tests**: run the `tgsort` binary against fixture files in `testdata/`, compare output to golden `.hcl` files.
- **Config tests**: valid config, partial config (only one field set), missing config (defaults apply).
