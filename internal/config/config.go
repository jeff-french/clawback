package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/titanous/json5"
)

const DefaultConfigDir = "./config"
const DefaultOutputFile = "./openclaw.json"
const DefaultMasterTemplate = "./config/openclaw.json5"

var DefaultPassthrough = []string{"meta", "wizard", "plugins.installs"}

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
		Passthrough:    DefaultPassthrough,
	}

	path := filepath.Join(homeDir, ".clawback.json5")
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to read symlink: %s", path)
	}
	const maxConfigSize = 1 << 20 // 1 MB
	if info.Size() > maxConfigSize {
		return nil, fmt.Errorf("config file too large (%d bytes, limit %d): %s", info.Size(), maxConfigSize, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := json5.Unmarshal(data, cfg); err != nil {
		return nil, err
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
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("config %s path %q escapes home directory", ch.name, ch.val)
		}
	}
	return nil
}
