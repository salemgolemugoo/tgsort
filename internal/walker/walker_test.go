package walker_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/salemgolemugoo/tgsort/internal/config"
	"github.com/salemgolemugoo/tgsort/internal/walker"
)

var cfg = &config.Config{
	BlockOrder:       config.DefaultBlockOrder,
	SortAttributesIn: config.DefaultSortAttributesIn,
}

func TestProcessFile_SortsInPlace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.hcl")
	input := `inputs = {
  z_key = "last"
}

terraform {
  source = "..."
}
`
	want := `terraform {
  source = "..."
}

inputs = {
  z_key = "last"
}
`
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}
	w := walker.New(cfg, false)
	changed, err := w.ProcessFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	got, _ := os.ReadFile(path)
	if string(got) != want {
		t.Errorf("file content wrong:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestProcessFile_AlreadySorted_NotChanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.hcl")
	input := `terraform {
  source = "..."
}
`
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}
	w := walker.New(cfg, false)
	changed, err := w.ProcessFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Error("expected changed=false for already-sorted file")
	}
}

func TestProcessFile_DryRun_DoesNotModify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.hcl")
	input := `inputs = {
  z = "z"
}

terraform {
  source = "..."
}
`
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}
	w := walker.New(cfg, true)
	changed, err := w.ProcessFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true in dry-run")
	}
	// File should not be modified.
	got, _ := os.ReadFile(path)
	if string(got) != input {
		t.Errorf("dry-run must not modify file")
	}
}

func TestProcessDir_ProcessesHCLFiles(t *testing.T) {
	dir := t.TempDir()
	input := `inputs = {
  key = "v"
}

terraform {
  source = "..."
}
`
	want := `terraform {
  source = "..."
}

inputs = {
  key = "v"
}
`
	write := func(name string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(input), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("a.hcl")
	write("b.hcl")
	if err := os.WriteFile(filepath.Join(dir, "skip.tf"), []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	w := walker.New(cfg, false)
	if _, err := w.ProcessDir(dir, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, name := range []string{"a.hcl", "b.hcl"} {
		got, _ := os.ReadFile(filepath.Join(dir, name))
		if string(got) != want {
			t.Errorf("%s: wrong content after sort", name)
		}
	}
	// .tf file must not be touched.
	got, _ := os.ReadFile(filepath.Join(dir, "skip.tf"))
	if string(got) != input {
		t.Error("non-.hcl file was modified")
	}
}
