//go:build integration

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	binaryOnce sync.Once
	binaryPath string
)

func getBinary(t *testing.T) string {
	t.Helper()
	binaryOnce.Do(func() { binaryPath = buildBinary(t) })
	return binaryPath
}

func buildBinary(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "tgsort-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	bin := filepath.Join(dir, "tgsort")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build failed: %v", err)
	}
	return bin
}

func TestIntegration_FullFixture(t *testing.T) {
	bin := getBinary(t)

	input, err := os.ReadFile("testdata/full/input.hcl")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile("testdata/full/expected.hcl")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.hcl")
	if err := os.WriteFile(path, input, 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("tgsort failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(expected) {
		t.Errorf("full fixture mismatch:\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestIntegration_CommentsFixture(t *testing.T) {
	bin := getBinary(t)

	input, err := os.ReadFile("testdata/comments/input.hcl")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile("testdata/comments/expected.hcl")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.hcl")
	if err := os.WriteFile(path, input, 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("tgsort failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(expected) {
		t.Errorf("comments fixture mismatch:\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestIntegration_DryRun_ExitsNonZero_WhenChangesExist(t *testing.T) {
	bin := getBinary(t)

	input, err := os.ReadFile("testdata/full/input.hcl")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "test.hcl")
	if err := os.WriteFile(path, input, 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--dry-run", path)
	if err := cmd.Run(); err == nil {
		t.Error("expected non-zero exit for dry-run with changes, got 0")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(input) {
		t.Error("dry-run modified the file")
	}
}

func TestIntegration_Stdin(t *testing.T) {
	bin := getBinary(t)

	input := "inputs = { key = \"v\" }\n\nterraform { source = \"...\" }\n"
	want := "terraform { source = \"...\" }\n\ninputs = { key = \"v\" }\n"

	cmd := exec.Command(bin, "-")
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("tgsort - failed: %v", err)
	}
	if string(out) != want {
		t.Errorf("stdin output wrong:\ngot:\n%q\nwant:\n%q", out, want)
	}
}

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
