package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/salemgolemugoo/tgsort/internal/config"
)

func TestLoad_NoFile_ReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.BlockOrder) == 0 {
		t.Fatal("expected non-empty default block order")
	}
	if cfg.BlockOrder[0] != "terraform" {
		t.Errorf("expected first block to be 'terraform', got %q", cfg.BlockOrder[0])
	}
	want := []string{"terraform", "remote_state", "include", "locals", "generate", "dependency", "inputs"}
	if len(cfg.BlockOrder) != len(want) {
		t.Errorf("expected %d block order entries, got %d", len(want), len(cfg.BlockOrder))
	}
	if len(cfg.SortAttributesIn) != 1 || cfg.SortAttributesIn[0] != "inputs" {
		t.Errorf("expected default sort_attributes_in=[inputs], got %v", cfg.SortAttributesIn)
	}
}

func TestLoad_ValidFile_OverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".tgsort"), `
block_order = ["terraform", "locals", "inputs"]
sort_attributes_in = ["inputs", "locals"]
`)
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.BlockOrder) != 3 || cfg.BlockOrder[1] != "locals" {
		t.Errorf("unexpected block_order: %v", cfg.BlockOrder)
	}
	if len(cfg.SortAttributesIn) != 2 || cfg.SortAttributesIn[1] != "locals" {
		t.Errorf("unexpected sort_attributes_in: %v", cfg.SortAttributesIn)
	}
}

func TestLoad_PartialFile_UnsetFieldsKeepDefaults(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".tgsort"), `sort_attributes_in = ["inputs", "locals"]`)
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BlockOrder[0] != "terraform" {
		t.Errorf("expected default block_order when not set, got %v", cfg.BlockOrder)
	}
	if len(cfg.SortAttributesIn) != 2 {
		t.Errorf("unexpected sort_attributes_in: %v", cfg.SortAttributesIn)
	}
}

func TestLoad_InvalidTOML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".tgsort"), `block_order = [invalid toml`)
	_, err := config.Load(dir)
	if err == nil {
		t.Error("expected error for invalid TOML, got nil")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
