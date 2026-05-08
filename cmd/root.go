package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salemgolemugoo/tgsort/internal/config"
	"github.com/salemgolemugoo/tgsort/internal/walker"
)

var version = "dev"

var (
	dryRun    bool
	recursive bool
)

var rootCmd = &cobra.Command{
	Use:     "tgsort [file_or_directory...|-]",
	Short:   "Sort blocks and attributes in Terragrunt HCL files",
	Version: version,
	Args:    cobra.ArbitraryArgs,
	RunE:    run,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Print diff without modifying files; exit 1 if changes exist")
	rootCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recurse into subdirectories")
}

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
