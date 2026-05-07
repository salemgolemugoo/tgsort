package sorter_test

import (
	"testing"

	"github.com/salemgolemugoo/tgsort/internal/config"
	"github.com/salemgolemugoo/tgsort/internal/sorter"
)

var defaultCfg = &config.Config{
	BlockOrder:       config.DefaultBlockOrder,
	SortAttributesIn: config.DefaultSortAttributesIn,
}

func check(t *testing.T, name, src, want string) {
	t.Helper()
	got, err := sorter.Sort([]byte(src), defaultCfg)
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", name, err)
	}
	if string(got) != want {
		t.Errorf("%s:\ngot:\n%s\nwant:\n%s", name, got, want)
	}
}

func TestSort_AlreadySorted_Unchanged(t *testing.T) {
	src := `terraform {
  source = "git::https://example.com/module.git"
}

inputs = {
  vpc_id = "vpc-123"
}
`
	check(t, "already sorted", src, src)
}

func TestSort_BlocksReordered(t *testing.T) {
	check(t, "reorder blocks",
		`inputs = {
  vpc_id = "vpc-123"
}

terraform {
  source = "git::https://example.com/module.git"
}
`,
		`terraform {
  source = "git::https://example.com/module.git"
}

inputs = {
  vpc_id = "vpc-123"
}
`)
}

func TestSort_SameTypeBlocksSortedByLabel(t *testing.T) {
	check(t, "same-type label sort",
		`dependency "vpc" {
  config_path = "../vpc"
}

dependency "eks" {
  config_path = "../eks"
}
`,
		`dependency "eks" {
  config_path = "../eks"
}

dependency "vpc" {
  config_path = "../vpc"
}
`)
}

func TestSort_CommentTravelsWithBlock(t *testing.T) {
	check(t, "comment attachment",
		`# Configures networking
dependency "vpc" {
  config_path = "../vpc"
}

# Sets up k8s
dependency "eks" {
  config_path = "../eks"
}
`,
		`# Sets up k8s
dependency "eks" {
  config_path = "../eks"
}

# Configures networking
dependency "vpc" {
  config_path = "../vpc"
}
`)
}

func TestSort_CommentSeparatedByBlankLine_StaysInPlace(t *testing.T) {
	// A blank line between a comment and the next block means the comment
	// is NOT attached to that block; it stays as a file header.
	src := `# File-level comment

dependency "vpc" {
  config_path = "../vpc"
}
`
	check(t, "detached comment stays as header", src, src)
}

func TestSort_UnlistedBlockTypesGoLast_SortedAlphabetically(t *testing.T) {
	check(t, "unlisted blocks last",
		`zebra_block "foo" {
  key = "val"
}

alpha_block "bar" {
  key = "val"
}

terraform {
  source = "..."
}
`,
		`terraform {
  source = "..."
}

alpha_block "bar" {
  key = "val"
}

zebra_block "foo" {
  key = "val"
}
`)
}

func TestSort_InvalidHCL_ReturnsError(t *testing.T) {
	_, err := sorter.Sort([]byte(`terraform {`), defaultCfg)
	if err == nil {
		t.Error("expected error for invalid HCL, got nil")
	}
}

func TestSort_InputsMapKeysSorted(t *testing.T) {
	check(t, "inputs map key sort",
		`inputs = {
  z_key = "last"
  a_key = "first"
  m_key = "middle"
}
`,
		`inputs = {
  a_key = "first"
  m_key = "middle"
  z_key = "last"
}
`)
}

func TestSort_InputsMapKeys_CommentTravelsWithKey(t *testing.T) {
	check(t, "inputs key comment attachment",
		`inputs = {
  # vpc output
  z_key = dependency.vpc.outputs.vpc_id

  a_key = "first"
}
`,
		`inputs = {
  a_key = "first"

  # vpc output
  z_key = dependency.vpc.outputs.vpc_id
}
`)
}

func TestSort_SingleLineInputsMapUnchanged(t *testing.T) {
	check(t, "single-line inputs map unchanged",
		`inputs = { z = "z", a = "a" }
`,
		`inputs = { z = "z", a = "a" }
`)
}

func TestSort_TrailingContentPreserved(t *testing.T) {
	check(t, "trailing comment preserved",
		`terraform {
  source = "..."
}

# trailing comment
`,
		`terraform {
  source = "..."
}

# trailing comment
`)
}

func TestSort_DoubleSlashCommentTravelsWithBlock(t *testing.T) {
	check(t, "// comment travels with block",
		`// Configures networking
dependency "vpc" {
  config_path = "../vpc"
}

// Sets up k8s
dependency "eks" {
  config_path = "../eks"
}
`,
		`// Sets up k8s
dependency "eks" {
  config_path = "../eks"
}

// Configures networking
dependency "vpc" {
  config_path = "../vpc"
}
`)
}

func TestSort_LocalsBlockAttributesSorted(t *testing.T) {
	cfg := &config.Config{
		BlockOrder:       config.DefaultBlockOrder,
		SortAttributesIn: []string{"inputs", "locals"},
	}
	src := `locals {
  z_var = "last"
  a_var = "first"
}
`
	want := `locals {
  a_var = "first"
  z_var = "last"
}
`
	got, err := sorter.Sort([]byte(src), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != want {
		t.Errorf("locals sort failed:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
