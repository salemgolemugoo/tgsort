package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const FileName = ".tgsort"

var DefaultBlockOrder = []string{
	"terraform",
	"remote_state",
	"include",
	"locals",
	"generate",
	"dependency",
	"inputs",
}

var DefaultSortAttributesIn = []string{"inputs"}

type Config struct {
	BlockOrder       []string `toml:"block_order"`
	SortAttributesIn []string `toml:"sort_attributes_in"`
}

// Load reads .tgsort from dir. If absent, returns defaults silently.
// If present but unparseable, returns an error.
// Fields absent from the file keep their default values.
func Load(dir string) (*Config, error) {
	cfg := &Config{
		BlockOrder:       DefaultBlockOrder,
		SortAttributesIn: DefaultSortAttributesIn,
	}

	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	// Decode into a separate struct so nil (unset) slices don't overwrite defaults.
	var override struct {
		BlockOrder       []string `toml:"block_order"`
		SortAttributesIn []string `toml:"sort_attributes_in"`
	}
	if _, err := toml.Decode(string(data), &override); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if override.BlockOrder != nil {
		cfg.BlockOrder = override.BlockOrder
	}
	if override.SortAttributesIn != nil {
		cfg.SortAttributesIn = override.SortAttributesIn
	}
	return cfg, nil
}
