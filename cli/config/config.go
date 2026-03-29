package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds user preferences
type Config struct {
	Columns     []string `json:"columns"`
	RefreshRate int      `json:"refresh_rate_ms"`
	Theme       string   `json:"theme"`
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Columns:     []string{"pid", "name", "command", "threads", "user", "memory", "cpu"},
		RefreshRate: 1000,
		Theme:       "charm",
	}
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "neohtop-cli", "config.json")
}

// Load reads config from disk, returning defaults if not found
func Load() *Config {
	path := configPath()
	if path == "" {
		return DefaultConfig()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig()
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return DefaultConfig()
	}

	return cfg
}

// Save writes config to disk
func Save(cfg *Config) error {
	path := configPath()
	if path == "" {
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
