package config

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	json5parser "github.com/jeff-french/clawback/internal/json5"
	"github.com/titanous/json5"
)

const DefaultConfigDir = "./config"
const DefaultOutputFile = "./openclaw.json"
const DefaultMasterTemplate = "./config/openclaw.json5"

// DefaultPassthrough returns the default passthrough paths.
// Returned as a function to prevent mutation of a shared slice.
func DefaultPassthrough() []string {
	return []string{"meta", "wizard", "plugins.installs"}
}

// Config represents the .clawback.json5 configuration file.
type Config struct {
	ConfigDir      string   `json:"configDir"`
	OutputFile     string   `json:"outputFile"`
	MasterTemplate string   `json:"masterTemplate"`
	Passthrough    []string `json:"passthrough"`
}

// Load reads .clawback.json5 from the given home directory.
// If the file doesn't exist, returns defaults.
func Load(homeDir string) (*Config, error) {
	cfg := &Config{
		ConfigDir:      DefaultConfigDir,
		OutputFile:     DefaultOutputFile,
		MasterTemplate: DefaultMasterTemplate,
		Passthrough:    DefaultPassthrough(),
	}

	path := filepath.Join(homeDir, ".clawback.json5")
	data, err := json5parser.SafeReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	if err := json5.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return cfg, nil
}

// ResolvePath resolves a config-relative path against the home directory.
func (c *Config) ResolvePath(homeDir, rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(homeDir, rel)
}

// OutputPath returns the absolute path to the output file.
func (c *Config) OutputPath(homeDir string) string {
	return c.ResolvePath(homeDir, c.OutputFile)
}

// MasterTemplatePath returns the absolute path to the master template.
func (c *Config) MasterTemplatePath(homeDir string) string {
	return c.ResolvePath(homeDir, c.MasterTemplate)
}

// Validate checks that all configured paths stay within homeDir.
// A malicious .clawback.json5 could otherwise point outputFile or
// masterTemplate at an arbitrary location on the filesystem.
func (c *Config) Validate(homeDir string) error {
	root := filepath.Clean(homeDir)
	checks := []struct {
		name string
		val  string
	}{
		{"configDir", c.ConfigDir},
		{"outputFile", c.OutputFile},
		{"masterTemplate", c.MasterTemplate},
	}
	for _, ch := range checks {
		abs := c.ResolvePath(homeDir, ch.val)
		rel, err := filepath.Rel(root, abs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("config %s path %q escapes home directory", ch.name, ch.val)
		}
	}
	return nil
}
