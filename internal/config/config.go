package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Profile holds the provider settings for one named profile.
type Profile struct {
	Provider     string `yaml:"provider,omitempty"`
	APIKey       string `yaml:"api_key,omitempty"`
	APIKeyHeader string `yaml:"api_key_header,omitempty"`
	Model        string `yaml:"model,omitempty"`
	BaseURL      string `yaml:"base_url,omitempty"`
}

// File is the on-disk config: named profiles plus global settings.
type File struct {
	Active    string             `yaml:"active,omitempty"`
	Clipboard bool               `yaml:"clipboard,omitempty"`
	Profiles  map[string]Profile `yaml:"profiles,omitempty"`

	// legacy flat fields — read once for migration, never written back
	LegacyProvider string `yaml:"provider,omitempty"`
	LegacyAPIKey   string `yaml:"api_key,omitempty"`
	LegacyModel    string `yaml:"model,omitempty"`
	LegacyBaseURL  string `yaml:"base_url,omitempty"`
}

// Config is the resolved view of the active (or requested) profile,
// after env overrides and defaults. This is what commands consume.
type Config struct {
	ProfileName string
	Provider    string
	APIKey      string
	APIKeyHeader string
	Model       string
	BaseURL     string
	Clipboard   bool
}

const DefaultProfile = "default"

// LoadFile reads the raw config file, migrating a legacy flat config
// into a "default" profile in memory. The migrated form is persisted
// the next time SaveFile is called.
func LoadFile() (*File, error) {
	f := &File{}
	path := Path()
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, f); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	}
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	// migrate legacy flat config into the default profile
	if f.LegacyProvider != "" || f.LegacyAPIKey != "" || f.LegacyModel != "" || f.LegacyBaseURL != "" {
		if _, exists := f.Profiles[DefaultProfile]; !exists {
			f.Profiles[DefaultProfile] = Profile{
				Provider: f.LegacyProvider,
				APIKey:   f.LegacyAPIKey,
				Model:    f.LegacyModel,
				BaseURL:  f.LegacyBaseURL,
			}
		}
		f.LegacyProvider, f.LegacyAPIKey, f.LegacyModel, f.LegacyBaseURL = "", "", "", ""
		if f.Active == "" {
			f.Active = DefaultProfile
		}
	}
	return f, nil
}

// SaveFile writes the config file with owner-only permissions.
func SaveFile(f *File) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ProfileNames returns all profile names, sorted.
func (f *File) ProfileNames() []string {
	names := make([]string, 0, len(f.Profiles))
	for n := range f.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// ActiveName resolves which profile is active: CLAI_PROFILE env wins,
// then the file's active field, then "default".
func (f *File) ActiveName() string {
	if v := os.Getenv("CLAI_PROFILE"); v != "" {
		return v
	}
	if f.Active != "" {
		return f.Active
	}
	return DefaultProfile
}

// Load resolves the active profile.
func Load() (*Config, error) {
	return LoadProfile("")
}

// LoadProfile resolves the named profile (or the active one if name is
// empty), applying env overrides and per-provider defaults.
func LoadProfile(name string) (*Config, error) {
	f, err := LoadFile()
	if err != nil {
		return nil, err
	}

	explicit := name != ""
	if name == "" {
		name = f.ActiveName()
	}

	p, ok := f.Profiles[name]
	if !ok {
		// a profile the user asked for by name (flag, env, or saved
		// active) must exist; only the implicit default may be empty
		if explicit || name != DefaultProfile {
			return nil, fmt.Errorf("profile %q not found (have: %v) — create it with: clai config profile add %s", name, f.ProfileNames(), name)
		}
	}

	cfg := &Config{
		ProfileName:  name,
		Provider:     p.Provider,
		APIKey:       p.APIKey,
		APIKeyHeader: p.APIKeyHeader,
		Model:        p.Model,
		BaseURL:      p.BaseURL,
		Clipboard:    f.Clipboard,
	}
	if cfg.Provider == "" {
		cfg.Provider = "openai"
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

func Path() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "clai", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "clai", "config.yaml")
}
