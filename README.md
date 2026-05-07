# tgsort

[![CI](https://github.com/salemgolemugoo/tgsort/actions/workflows/ci.yaml/badge.svg)](https://github.com/salemgolemugoo/tgsort/actions/workflows/ci.yaml)
[![Latest Release](https://img.shields.io/github/v/release/salemgolemugoo/tgsort)](https://github.com/salemgolemugoo/tgsort/releases/latest)
[![Go Version](https://img.shields.io/badge/go-1.23-blue)](https://go.dev/doc/go1.23)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

`tgsort` sorts blocks and attributes in Terragrunt HCL files — like `gofmt`, but for Terragrunt configuration. It rewrites files in-place so your `terragrunt.hcl` always has a consistent, reviewable layout regardless of who wrote it.

## Installation

### Homebrew (macOS and Linux)

```sh
brew tap salemgolemugoo/tap
brew install tgsort
```

### Go install

```sh
go install github.com/salemgolemugoo/tgsort@latest
```

### Binary

Download a pre-built binary for your platform from the [Releases](https://github.com/salemgolemugoo/tgsort/releases/latest) page.

## Usage

```
tgsort [file_or_directory|-] [flags]

Flags:
  -d, --dry-run     Print a unified diff without modifying files; exits 1 if any file would change
  -r, --recursive   Recurse into subdirectories when a directory is given
      --version     Print version and exit
```

### Examples

```sh
# Sort all .hcl files in the current directory
tgsort

# Sort a single file
tgsort terragrunt.hcl

# Sort an entire module tree recursively
tgsort --recursive ./modules

# Preview changes without writing them
tgsort --dry-run

# Read from stdin, write sorted output to stdout
cat terragrunt.hcl | tgsort -

# CI gate — fail if any files are unsorted
tgsort --dry-run --recursive .
```

## Configuration

Place a `.tgsort` file in the directory where you run `tgsort`. It is optional — all fields
have defaults and an absent file is silently ignored. A malformed file is a fatal error.

```toml
# .tgsort
block_order        = ["terraform", "remote_state", "include", "locals", "generate", "dependency", "inputs"]
sort_attributes_in = ["inputs"]
```

| Field | Default | Description |
|-------|---------|-------------|
| `block_order` | `["terraform", "remote_state", "include", "locals", "generate", "dependency", "inputs"]` | Block types sorted in this order; types not listed sort alphabetically after |
| `sort_attributes_in` | `["inputs"]` | Attribute keys inside these block types are also sorted |

## How it works

`tgsort` parses HCL with `hclsyntax` to identify top-level block and attribute positions, then
reorders them using a stable sort — preserving exact whitespace and inline formatting. Comments
immediately preceding a block travel with that block during reordering. Single-line object
expressions (e.g. `inputs = { key = "value" }`) are left unsorted.

## Contributing

```sh
# Install pre-commit hooks
pre-commit install

# Unit tests
go test ./...

# Integration tests (compiles the binary and runs golden-file fixtures)
go test -tags integration ./...

# Race detector
go test -race ./...
```

Commits must follow [Conventional Commits](https://www.conventionalcommits.org/) — enforced by
the `commit-msg` pre-commit hook.
