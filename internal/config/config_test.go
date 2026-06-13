package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupConfig points XDG_CONFIG_HOME at a temp dir, clears all CLAI/provider
// env vars, and optionally writes a config file.
func setupConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	for _, v := range []string{"CLAI_PROFILE", "CLAI_PROVIDER", "CLAI_API_KEY", "CLAI_MODEL", "CLAI_BASE_URL", "OPENAI_API_KEY", "ANTHROPIC_API_KEY"} {
		t.Setenv(v, "")
	}
	if contents != "" {
		path := filepath.Join(dir, "clai", "config.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLoadFreshInstallDefaults(t *testing.T) {
	setupConfig(t, "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProfileName != "default" {
		t.Errorf("profile = %q, want default", cfg.ProfileName)
	}
	if cfg.Provider != "openai" {
		t.Errorf("provider = %q, want openai", cfg.Provider)
	}
	if cfg.Model != "gpt-4o-mini" {
		t.Errorf("model = %q, want gpt-4o-mini", cfg.Model)
	}
}

func TestLegacyFlatConfigMigration(t *testing.T) {
	setupConfig(t, `provider: anthropic
api_key: sk-ant-test
model: claude-haiku-4-5-20251001
clipboard: true
`)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProfileName != "default" {
		t.Errorf("profile = %q, want default", cfg.ProfileName)
	}
	if cfg.Provider != "anthropic" || cfg.APIKey != "sk-ant-test" {
		t.Errorf("legacy values not migrated: %+v", cfg)
	}
	if !cfg.Clipboard {
		t.Error("clipboard not preserved through migration")
	}

	// saving must persist the new format without legacy top-level keys
	f, err := LoadFile()
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveFile(f); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "profiles:") || !strings.Contains(s, "default:") {
		t.Errorf("saved file missing profiles section:\n%s", s)
	}
	if strings.Contains(s, "\nprovider:") || strings.HasPrefix(s, "provider:") {
		t.Errorf("saved file still has top-level legacy provider key:\n%s", s)
	}
}

func TestMultipleProfilesAndActive(t *testing.T) {
	setupConfig(t, `active: ollama
profiles:
  ollama:
    provider: ollama
    model: llama3.2
    base_url: http://localhost:11434/v1
  work:
    provider: anthropic
    api_key: sk-ant-work
    model: claude-haiku-4-5-20251001
`)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProfileName != "ollama" || cfg.Provider != "ollama" || cfg.Model != "llama3.2" {
		t.Errorf("active profile not resolved: %+v", cfg)
	}

	// explicit profile selection
	cfg, err = LoadProfile("work")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "anthropic" || cfg.APIKey != "sk-ant-work" {
		t.Errorf("explicit profile not resolved: %+v", cfg)
	}

	// unknown profile errors
	if _, err := LoadProfile("nope"); err == nil {
		t.Error("expected error for unknown profile")
	}
}

func TestProfileEnvOverride(t *testing.T) {
	setupConfig(t, `active: a
profiles:
  a:
    provider: openai
    model: gpt-4o-mini
  b:
    provider: anthropic
    model: claude-haiku-4-5-20251001
    api_key: k
`)
	t.Setenv("CLAI_PROFILE", "b")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProfileName != "b" || cfg.Provider != "anthropic" {
		t.Errorf("CLAI_PROFILE not honored: %+v", cfg)
	}
}

func TestEnvValueOverrides(t *testing.T) {
	setupConfig(t, `active: a
profiles:
  a:
    provider: openai
    model: gpt-4o-mini
    api_key: from-file
`)
	t.Setenv("CLAI_MODEL", "gpt-4o")
	t.Setenv("CLAI_API_KEY", "from-env")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "gpt-4o" || cfg.APIKey != "from-env" {
		t.Errorf("env overrides not applied: %+v", cfg)
	}
}

func TestProviderKeyFallback(t *testing.T) {
	setupConfig(t, `active: a
profiles:
  a:
    provider: anthropic
`)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-env")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "sk-ant-env" {
		t.Errorf("provider key fallback not applied: %+v", cfg)
	}
	if cfg.Model != "claude-haiku-4-5-20251001" {
		t.Errorf("model default not applied: %+v", cfg)
	}
}

func TestSaveFilePermissions(t *testing.T) {
	setupConfig(t, "")
	f := &File{
		Active: "x",
		Profiles: map[string]Profile{
			"x": {Provider: "openai", APIKey: "secret"},
		},
	}
	if err := SaveFile(f); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(Path())
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config file perms = %o, want 0600", perm)
	}
}
