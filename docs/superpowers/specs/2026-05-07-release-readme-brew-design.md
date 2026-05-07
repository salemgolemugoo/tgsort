# Release Pipeline, README, and Homebrew Tap — Design Spec

**Date:** 2026-05-07
**Repo:** github.com/salemgolemugoo/tgsort
**License:** Apache-2.0

---

## Goals

1. Fix the broken release pipeline (semantic-release with stale copy-paste config).
2. Publish multi-platform Go binaries on every release via GoReleaser.
3. Provide a `brew install` path via a general Homebrew tap (`salemgolemugoo/homebrew-tap`).
4. Write a README with badges and complete installation/usage documentation.

---

## 1. Module Path Fix

`go.mod` declares the module as `tgsort`. This must be changed to `github.com/salemgolemugoo/tgsort` so that:
- `go install github.com/salemgolemugoo/tgsort@latest` resolves correctly.
- GoReleaser's build references the correct import path.

All internal import paths (`tgsort/cmd`, `tgsort/internal/...`) must be updated to `github.com/salemgolemugoo/tgsort/cmd`, etc.

---

## 2. Release Pipeline — GoReleaser

### Replace semantic-release

The existing `.releaserc.json` is a copy-paste from a different project. It references
`chart/argocd-bitbucket-proxy/Chart.yaml`, which does not exist here, and produces no Go
binary artifacts. Delete this file.

### `.goreleaser.yaml`

New file at repo root. Configuration:

**Builds:**
- Target OS/arch matrix: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`
- Binary name: `tgsort`
- `ldflags`: inject version from GoReleaser's `{{ .Version }}`

**Archives:**
- Format: `.tar.gz` for linux/darwin, `.zip` for windows
- Each archive contains the binary only (no README — it's a CLI tool)
- Name template: `{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}`

**Checksum:**
- SHA256, file named `checksums.txt`

**Changelog:**
- Use `git` changelog, grouped by conventional commit type (`feat`, `fix`, `chore`)
- Exclude `chore(release)` bump commits

**Homebrew tap:**
- Repository: `salemgolemugoo/homebrew-tap`
- Formula name: `tgsort`
- Description and homepage pulled from root fields
- Install block: install binary, install shell completions if added later
- Uses `HOMEBREW_TAP_GITHUB_TOKEN` for push access

### CI Workflow — `.github/workflows/ci.yaml`

**`test` job** — unchanged. Runs on push to `main` and on PRs.

**`release` job:**
- Trigger: `push` to tags matching `v*` (not on every push to `main`)
- Remove `actions/setup-node` and semantic-release steps
- Add `goreleaser/goreleaser-action@v6`
- Needs `GITHUB_TOKEN` (for GitHub release) and `HOMEBREW_TAP_GITHUB_TOKEN` (for tap formula push)
- `fetch-depth: 0` kept (GoReleaser needs full tag history for changelog)

Releasing a new version becomes: create and push a `vX.Y.Z` tag. CI handles the rest.

---

## 3. Cleanup

**`.pre-commit-config.yaml`:** Remove the `helm-docs` hook (references `chart/` directory that
does not exist in this repo — another copy-paste leftover).

---

## 4. README.md

New file at repo root.

### Badges (top of file)

| Badge | Source |
|-------|--------|
| CI | `github.com/salemgolemugoo/tgsort` Actions workflow status |
| Latest release | GitHub release shield |
| Go version | Static shield: `1.23` |
| License | Static shield: `Apache-2.0` |

### Sections

**Title + one-liner**
> `tgsort` — sort blocks and attributes in Terragrunt HCL files.

**What it does (2–3 sentences)**
Brief description of the problem it solves: consistent, deterministic ordering of Terragrunt
config blocks (like `gofmt` for HCL), idempotent in-place rewrites, configurable block order.

**Installation**

Three methods, Homebrew first (lowest friction for macOS users):

```sh
# Homebrew (macOS / Linux)
brew tap salemgolemugoo/tap
brew install tgsort

# Go install
go install github.com/salemgolemugoo/tgsort@latest

# Manual — download a pre-built binary from GitHub Releases
# https://github.com/salemgolemugoo/tgsort/releases/latest
```

**Usage**

```
tgsort [file_or_directory|-] [flags]

Flags:
  -d, --dry-run     Print diff without modifying files; exit 1 if changes exist
  -r, --recursive   Recurse into subdirectories
  -v, --version     Print version
```

Examples covering: single file, directory, recursive, stdin, dry-run.

**Configuration**

`.tgsort` file (TOML) in the working directory. Show defaults:

```toml
block_order = ["terraform", "remote_state", "include", "locals", "generate", "dependency", "inputs"]
sort_attributes_in = ["inputs"]
```

Note: absent file is silently ignored; malformed file is a fatal error.

**Contributing**
- Pre-commit hooks (`pre-commit install`)
- Conventional commits required
- `go test ./...` for unit tests, `go test -tags integration ./...` for integration tests

---

## 5. Manual Prerequisites (one-time)

Before the first release:

1. Create the `salemgolemugoo/homebrew-tap` GitHub repo (can be empty).
2. Generate a GitHub Personal Access Token (PAT) with `repo` scope.
3. Add it as a secret named `HOMEBREW_TAP_GITHUB_TOKEN` in `salemgolemugoo/tgsort` → Settings → Secrets.

GoReleaser will create and update the formula file in the tap repo on every release.

---

## Out of scope

- Shell completions (can be added later via Cobra's built-in completion command).
- Homebrew-core submission (requires significant install base first).
- Docker image / container distribution.
- Windows `.msi` installer.
