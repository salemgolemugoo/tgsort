# Multi-file Input Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow tgsort to accept multiple positional file/dir arguments so it works correctly as a pre-commit hook.

**Architecture:** Change `cobra.MaximumNArgs(1)` to `cobra.ArbitraryArgs` and refactor `run()` in `cmd/root.go` to loop over all args. The walker layer (`walker.go`) is unchanged — it already handles individual files and dirs. Tests are integration-level only since the dispatch logic is thin glue.

**Tech Stack:** Go, Cobra, `go test -tags integration`

---

### Task 1: Write failing integration tests

**Files:**
- Modify: `integration_test.go`

- [ ] **Step 1: Add the two new tests**

Open `integration_test.go` and append these two tests at the end of the file (before the final closing brace is not needed — just append to the file after the last test):

```go
func TestIntegration_MultipleFiles(t *testing.T) {
	bin := getBinary(t)

	unsorted := "inputs = {}\n\nterraform {\n  source = \".\"\n}\n"
	sorted := "terraform {\n  source = \".\"\n}\n\ninputs = {}\n"

	dir := t.TempDir()
	file1 := filepath.Join(dir, "a.hcl")
	file2 := filepath.Join(dir, "b.hcl")
	for _, p := range []string{file1, file2} {
		if err := os.WriteFile(p, []byte(unsorted), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command(bin, file1, file2)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("tgsort failed: %v", err)
	}

	for _, p := range []string{file1, file2} {
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != sorted {
			t.Errorf("%s: got %q, want %q", p, got, sorted)
		}
	}
}

func TestIntegration_StdinWithOtherArgs_Fails(t *testing.T) {
	bin := getBinary(t)

	dir := t.TempDir()
	dummy := filepath.Join(dir, "dummy.hcl")
	if err := os.WriteFile(dummy, []byte("inputs = {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "-", dummy)
	cmd.Stdin = strings.NewReader("inputs = {}\n")
	if err := cmd.Run(); err == nil {
		t.Error("expected non-zero exit when combining stdin (-) with other args")
	}
}
```

- [ ] **Step 2: Run the new tests to confirm they fail**

```bash
go test -tags integration -run 'TestIntegration_MultipleFiles|TestIntegration_StdinWithOtherArgs_Fails' ./...
```

Expected: both tests FAIL. `TestIntegration_MultipleFiles` fails because tgsort exits with "accepts at most 1 arg(s), received 2". `TestIntegration_StdinWithOtherArgs_Fails` fails because tgsort also exits non-zero for the wrong reason (arg count), but `err == nil` won't be triggered — actually this one may pass by accident. What matters is that `TestIntegration_MultipleFiles` fails.

---

### Task 2: Implement multi-arg support in cmd/root.go

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Update the `Use` string and `Args` validator**

Change `cmd/root.go` lines 22–27 from:

```go
var rootCmd = &cobra.Command{
	Use:     "tgsort [file_or_directory|-]",
	Short:   "Sort blocks and attributes in Terragrunt HCL files",
	Version: version,
	Args:    cobra.MaximumNArgs(1),
	RunE:    run,
}
```

to:

```go
var rootCmd = &cobra.Command{
	Use:     "tgsort [file_or_directory...|-]",
	Short:   "Sort blocks and attributes in Terragrunt HCL files",
	Version: version,
	Args:    cobra.ArbitraryArgs,
	RunE:    run,
}
```

- [ ] **Step 2: Replace the `run()` function body**

Replace the entire `run()` function (lines 41–105) with:

```go
func run(cmd *cobra.Command, args []string) error {
	for _, a := range args {
		if a == "-" && len(args) > 1 {
			return fmt.Errorf("cannot use stdin (-) with other arguments")
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	cfg, err := config.Load(wd)
	if err != nil {
		return err
	}

	w := walker.New(cfg, dryRun)

	if len(args) == 0 {
		hasChanges, err := w.ProcessDir(wd, recursive)
		if err != nil {
			return err
		}
		if dryRun && hasChanges {
			os.Exit(1)
		}
		return nil
	}

	if len(args) == 1 && args[0] == "-" {
		return w.ProcessStdin(os.Stdin, os.Stdout)
	}

	hasChanges := false
	for _, target := range args {
		if strings.HasSuffix(target, ".hcl.json") {
			return fmt.Errorf("%s: .hcl.json files are not supported", target)
		}
		info, err := os.Stat(target)
		if err != nil {
			return fmt.Errorf("%s: %w", target, err)
		}
		if info.IsDir() {
			changed, err := w.ProcessDir(target, recursive)
			if err != nil {
				return err
			}
			if changed {
				hasChanges = true
			}
		} else {
			changed, err := w.ProcessFile(target)
			if err != nil {
				return err
			}
			if changed {
				hasChanges = true
			}
		}
	}
	if dryRun && hasChanges {
		os.Exit(1)
	}
	return nil
}
```

Note: `.hcl.json` check is moved before `os.Stat` so it short-circuits before a syscall, but either order is correct.

- [ ] **Step 3: Verify it compiles**

```bash
go build ./...
```

Expected: no output, exit 0.

---

### Task 3: Run all tests

- [ ] **Step 1: Run unit tests**

```bash
go test ./...
```

Expected: all PASS, no failures.

- [ ] **Step 2: Run integration tests**

```bash
go test -tags integration ./...
```

Expected: all PASS including the two new tests.

- [ ] **Step 3: Run with race detector**

```bash
go test -race ./... && go test -race -tags integration ./...
```

Expected: all PASS.

---

### Task 4: Commit

- [ ] **Step 1: Commit the implementation**

```bash
git add cmd/root.go integration_test.go
git commit -m "feat: accept multiple file/dir arguments (pre-commit support)"
```
