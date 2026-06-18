// Package config loads optional user configuration from a TOML file. Every
// field has a sensible default, so readme-radar runs with no config at all.
// API keys are never stored here — they are read from the environment.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is the on-disk configuration.
type Config struct {
	AI AIConfig `toml:"ai"`
}

// AIConfig controls the optional LLM narration.
type AIConfig struct {
	Provider string `toml:"provider"` // anthropic | openai | ollama (empty = auto-detect)
	Model    string `toml:"model"`    // empty = provider default
	Enabled  bool   `toml:"enabled"`  // false = always use the offline narrative
}

// Default returns the zero-config defaults.
func Default() Config {
	return Config{AI: AIConfig{Enabled: true}}
}

// Path returns the config file location, honoring XDG_CONFIG_HOME.
func Path() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "readme-radar", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "readme-radar", "config.toml"), nil
}

// Load reads the config file, returning defaults if it does not exist.
func Load() (Config, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

const starter = `# readme-radar configuration
# All values are optional; defaults are shown.

[ai]
# Provider for the plain-English verdict narration.
#   anthropic | openai | ollama   (empty = auto-detect from your environment)
provider = ""
# Model name (empty = the provider's default).
model = ""
# Set false to always use the deterministic offline narrative.
enabled = true

# API keys are read from the environment, never stored here:
#   ANTHROPIC_API_KEY, OPENAI_API_KEY
# A GitHub token (optional, raises the API rate limit) is read from:
#   GITHUB_TOKEN or GH_TOKEN
`

// Init writes a documented starter config if none exists and returns its path.
func Init() (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil // already exists; leave it untouched
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(starter), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
