package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Provider  string `yaml:"provider"`
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	BaseURL   string `yaml:"base_url"`
	Clipboard bool   `yaml:"clipboard"`
}

func Load() (*Config, error) {
	cfg := &Config{
		Provider: "openai",
	}

	path := Path()
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	}

	if v := os.Getenv("CLAI_PROVIDER"); v != "" {
		cfg.Provider = v
	}
	if v := os.Getenv("CLAI_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("CLAI_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("CLAI_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}

	if cfg.APIKey == "" {
		switch cfg.Provider {
		case "openai", "litellm":
			cfg.APIKey = os.Getenv("OPENAI_API_KEY")
		case "anthropic":
			cfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}

	if cfg.Model == "" {
		switch cfg.Provider {
		case "openai", "litellm":
			cfg.Model = "gpt-4o-mini"
		case "anthropic":
			cfg.Model = "claude-haiku-4-5-20251001"
		}
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func Path() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "clai", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "clai", "config.yaml")
}
