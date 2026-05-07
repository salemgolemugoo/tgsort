# Release Pipeline, README, and Homebrew Tap — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the broken release pipeline, publish multi-platform Go binaries via GoReleaser, set up a Homebrew tap, and write a complete README.

**Architecture:** GoReleaser replaces semantic-release. On every `v*` tag push, CI builds linux/darwin/windows binaries, creates a GitHub release, and pushes an updated Homebrew formula to `salemgolemugoo/homebrew-tap`. The module path is corrected from `tgsort` to `github.com/salemgolemugoo/tgsort` so `go install` works.

**Tech Stack:** Go 1.23, GoReleaser v2, GitHub Actions, Homebrew tap (separate repo)

---

## File Map

| Action | File |
|--------|------|
| Modify | `go.mod` — module path |
| Modify | `main.go` — import path |
| Modify | `cmd/root.go` — import paths + `version` const→var |
| Modify | `internal/sorter/sorter.go` — import path |
| Modify | `internal/sorter/sorter_test.go` — import path |
| Modify | `internal/walker/walker.go` — import paths |
| Modify | `internal/walker/walker_test.go` — import paths |
| Modify | `internal/config/config_test.go` — import path |
| Delete | `.releaserc.json` |
| Modify | `.pre-commit-config.yaml` — remove helm-docs hook |
| Create | `.goreleaser.yaml` |
| Modify | `.github/workflows/ci.yaml` — replace release job, add tag trigger |
| Create | `README.md` |

---

## Task 1: Fix module path and update all internal imports

**Files:**
- Modify: `go.mod`
- Modify: `main.go`, `cmd/root.go`, `internal/sorter/sorter.go`, `internal/sorter/sorter_test.go`, `internal/walker/walker.go`, `internal/walker/walker_test.go`, `internal/config/config_test.go`

- [ ] **Step 1: Verify the files that need updating**

```bash
grep -rn '"tgsort/' . --include="*.go" -l
```

Expected output — exactly these 7 files:
```
./cmd/root.go
./internal/sorter/sorter_test.go
./main.go
./internal/sorter/sorter.go
./internal/walker/walker.go
./internal/walker/walker_test.go
./internal/config/config_test.go
```

- [ ] **Step 2: Update the module declaration in `go.mod`**

Change line 1 from:
```
module tgsort
```
to:
```
module github.com/salemgolemugoo/tgsort
```

- [ ] **Step 3: Rewrite all internal import paths in one pass**

```bash
find . -name "*.go" | xargs sed -i '' 's|"tgsort/|"github.com/salemgolemugoo/tgsort/|g'
```

- [ ] **Step 4: Verify the build compiles cleanly**

```bash
go build ./...
```

Expected: no output, exit 0.

- [ ] **Step 5: Verify all tests still pass**

```bash
go test ./... -race
```

Expected: all packages pass, no race conditions.

- [ ] **Step 6: Commit**

```bash
git add go.mod main.go cmd/root.go internal/sorter/sorter.go internal/sorter/sorter_test.go internal/walker/walker.go internal/walker/walker_test.go internal/config/config_test.go
git commit -m "fix: correct module path to github.com/salemgolemugoo/tgsort"
```

---

## Task 2: Make version injectable by GoReleaser ldflags

GoReleaser injects the version at build time via `-ldflags`. A `const` cannot be overwritten by the linker — it must be a `var`.

**Files:**
- Modify: `cmd/root.go:14`

- [ ] **Step 1: Change `version` from const to var in `cmd/root.go`**

Find line 14:
```go
const version = "0.1.0"
```

Replace with:
```go
var version = "dev"
```

- [ ] **Step 2: Verify the build and tests still pass**

```bash
go build ./... && go test ./... -race
```

Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add cmd/root.go
git commit -m "fix: make version injectable via ldflags for GoReleaser"
```

---

## Task 3: Remove stale copy-paste configuration

`.releaserc.json` references a Helm chart from a different project and produces no Go artifacts. The `helm-docs` pre-commit hook references a `chart/` directory that does not exist.

**Files:**
- Delete: `.releaserc.json`
- Modify: `.pre-commit-config.yaml`

- [ ] **Step 1: Delete `.releaserc.json`**

```bash
rm .releaserc.json
```

- [ ] **Step 2: Remove the `helm-docs` hook and clean up the stale `check-yaml` exclude from `.pre-commit-config.yaml`**

Replace the entire file content with:

```yaml
default_install_hook_types:
  - commit-msg
  - pre-commit

repos:
  - repo: https://github.com/compilerla/conventional-pre-commit
    rev: v4.1.0
    hooks:
      - id: conventional-pre-commit
        stages: [commit-msg]

  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v5.0.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-json
      - id: check-merge-conflict
      - id: detect-private-key

  - repo: https://github.com/dnephin/pre-commit-golang
    rev: v0.5.1
    hooks:
      - id: go-fmt
      - id: go-build
      - id: go-unit-tests
```

- [ ] **Step 3: Commit**

```bash
git add .releaserc.json .pre-commit-config.yaml
git commit -m "chore: remove stale semantic-release config and helm-docs hook"
```

---

## Task 4: Add GoReleaser configuration

**Files:**
- Create: `.goreleaser.yaml`

- [ ] **Step 1: Install GoReleaser locally for validation**

```bash
brew install goreleaser
```

Or, if using asdf — check the latest available version first:
```bash
asdf list-all goreleaser | tail -5
```
Then add to `.tool-versions` and install:
```bash
echo "goreleaser <version>" >> .tool-versions
asdf install goreleaser <version>
```

- [ ] **Step 2: Create `.goreleaser.yaml`**

```yaml
version: 2

project_name: tgsort

before:
  hooks:
    - go mod tidy

builds:
  - id: tgsort
    main: .
    binary: tgsort
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64
    ldflags:
      - -s -w -X github.com/salemgolemugoo/tgsort/cmd.version={{.Version}}

archives:
  - id: default
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    files:
      - LICENSE

checksum:
  name_template: checksums.txt
  algorithm: sha256

changelog:
  sort: asc
  use: git
  filters:
    exclude:
      - "^chore\\(release\\):"
  groups:
    - title: Features
      regexp: "^feat"
      order: 0
    - title: Bug Fixes
      regexp: "^fix"
      order: 1
    - title: Other
      order: 999

brews:
  - name: tgsort
    repository:
      owner: salemgolemugoo
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    homepage: "https://github.com/salemgolemugoo/tgsort"
    description: "Sort blocks and attributes in Terragrunt HCL files"
    license: "Apache-2.0"
    install: |
      bin.install "tgsort"
    test: |
      system "#{bin}/tgsort", "--version"
```

- [ ] **Step 3: Validate the GoReleaser config**

```bash
goreleaser check
```

Expected: `• config is valid` with no errors. Warnings about missing Git tags are fine (this is a local check, not a release run).

- [ ] **Step 4: Commit**

```bash
git add .goreleaser.yaml
git commit -m "feat: add GoReleaser config with multi-platform builds and Homebrew tap"
```

---

## Task 5: Update CI workflow

Replace the `release` job (currently broken semantic-release) with GoReleaser. Add a `v*` tag trigger so releases fire on tag push, not every commit to `main`.

**Files:**
- Modify: `.github/workflows/ci.yaml`

- [ ] **Step 1: Replace the entire `.github/workflows/ci.yaml` with:**

```yaml
name: CI

on:
  push:
    branches: [main]
    tags:
      - 'v*'
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - name: Run tests
        run: go test ./... -v -race -coverprofile=coverage.txt
      - name: Build
        run: go build ./...

  release:
    needs: [test]
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: ubuntu-latest
    permissions:
      contents: write
      issues: write
      pull-requests: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/ci.yaml
git commit -m "ci: replace semantic-release with GoReleaser, trigger on v* tags"
```

---

## Task 6: Write README.md

**Files:**
- Create: `README.md`

- [ ] **Step 1: Create `README.md` at the repo root with:**

```markdown
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
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README with badges, installation, usage, and configuration"
```

---

## Manual Prerequisites (one-time, before first release)

These steps require a human — they involve creating a GitHub repo and a secret.

1. Create the `salemgolemugoo/homebrew-tap` repository on GitHub (can be completely empty).
2. Generate a GitHub Personal Access Token (classic) with `repo` scope.
3. In `salemgolemugoo/tgsort` → Settings → Secrets and variables → Actions, add a secret named `HOMEBREW_TAP_GITHUB_TOKEN` with the PAT value.

## Releasing a new version

Once the above is done, releasing is:

```bash
git tag v0.1.0
git push origin v0.1.0
```

CI picks up the tag, runs tests, builds binaries for all platforms, creates a GitHub release with attached archives and `checksums.txt`, and pushes an updated `tgsort.rb` formula to `salemgolemugoo/homebrew-tap`.
