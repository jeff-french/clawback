package config

import (
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
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
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

// IsPassthrough returns true if the given JSON path is a passthrough section
// or a child of one (e.g. "plugins.installs.foo" matches "plugins.installs").
func (c *Config) IsPassthrough(path string) bool {
	for _, p := range c.Passthrough {
		if p == path || strings.HasPrefix(path, p+".") {
			return true
		}
	}
	return false
}
