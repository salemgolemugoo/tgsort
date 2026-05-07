package walker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/salemgolemugoo/tgsort/internal/config"
	"github.com/salemgolemugoo/tgsort/internal/sorter"
)

// Walker processes HCL files using the provided config.
type Walker struct {
	cfg    *config.Config
	dryRun bool
}

// New creates a Walker.
func New(cfg *config.Config, dryRun bool) *Walker {
	return &Walker{cfg: cfg, dryRun: dryRun}
}

// ProcessFile sorts a single .hcl file. Returns true if the file would/did change.
// In dry-run mode the file is not written; a unified diff is printed to stdout instead.
func (w *Walker) ProcessFile(path string) (bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}

	result, err := sorter.Sort(src, w.cfg)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}

	if bytes.Equal(src, result) {
		return false, nil
	}

	if w.dryRun {
		diff, err := unifiedDiff(path, string(src), string(result))
		if err != nil {
			return true, err
		}
		fmt.Print(diff)
		return true, nil
	}

	if err := os.WriteFile(path, result, 0644); err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	return true, nil
}

// ProcessDir processes all .hcl files in dir. Returns true if any file was/would be changed.
func (w *Walker) ProcessDir(dir string, recursive bool) (bool, error) {
	anyChanged := false
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if info.IsDir() {
			if path != dir && !recursive {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".hcl") {
			return nil
		}
		changed, err := w.ProcessFile(path)
		if changed {
			anyChanged = true
		}
		return err
	})
	return anyChanged, err
}

// ProcessStdin reads HCL from r, sorts it, writes the result to out.
// dryRun is ignored for stdin (sorted output is always written).
func (w *Walker) ProcessStdin(r io.Reader, out io.Writer) error {
	src, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	result, err := sorter.Sort(src, w.cfg)
	if err != nil {
		return fmt.Errorf("stdin: %w", err)
	}
	_, err = out.Write(result)
	return err
}

func unifiedDiff(filename, original, modified string) (string, error) {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(original),
		B:        difflib.SplitLines(modified),
		FromFile: filename,
		ToFile:   filename,
		Context:  3,
	}
	return difflib.GetUnifiedDiffString(diff)
}
