package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "clai-bin")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	binPath = filepath.Join(dir, "clai")
	build := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n%s", err, out)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// runCLI executes the built binary with an isolated config dir and a
// scrubbed environment so host CLAI_*/API key vars can't leak in.
func runCLI(t *testing.T, configDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + configDir,
		"XDG_CONFIG_HOME=" + configDir,
	}
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exitCode = 0
	if ee, ok := err.(*exec.ExitError); ok {
		exitCode = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run %v: %v", args, err)
	}
	return out.String(), errBuf.String(), exitCode
}

func TestProfileLifecycle(t *testing.T) {
	dir := t.TempDir()

	// add two profiles
	out, _, code := runCLI(t, dir, "config", "profile", "add", "local", "--provider", "ollama", "--model", "llama3.2")
	if code != 0 || !strings.Contains(out, `saved profile "local"`) {
		t.Fatalf("add local failed (code %d): %s", code, out)
	}
	out, _, code = runCLI(t, dir, "config", "profile", "add", "work", "--provider", "anthropic", "--model", "claude-haiku-4-5-20251001", "--api-key", "sk-ant-test123456")
	if code != 0 {
		t.Fatalf("add work failed: %s", out)
	}

	// first added profile became active
	out, _, _ = runCLI(t, dir, "config", "profile", "list")
	if !strings.Contains(out, "* local") || !strings.Contains(out, "  work") {
		t.Errorf("list markers wrong:\n%s", out)
	}

	// switch
	out, _, code = runCLI(t, dir, "config", "profile", "use", "work")
	if code != 0 || !strings.Contains(out, "active profile: work") {
		t.Fatalf("use work failed: %s", out)
	}
	out, _, _ = runCLI(t, dir, "config", "list")
	if !strings.Contains(out, "profile:   work") || !strings.Contains(out, "provider:  anthropic") {
		t.Errorf("config list after switch:\n%s", out)
	}

	// deleting the active profile is refused
	_, errOut, code := runCLI(t, dir, "config", "profile", "delete", "work")
	if code == 0 || !strings.Contains(errOut, "is active") {
		t.Errorf("delete active: code=%d stderr=%s", code, errOut)
	}

	// deleting an inactive one works
	out, _, code = runCLI(t, dir, "config", "profile", "delete", "local")
	if code != 0 || !strings.Contains(out, `deleted profile "local"`) {
		t.Errorf("delete local: code=%d out=%s", code, out)
	}

	// unknown profile use fails with the available list
	_, errOut, code = runCLI(t, dir, "config", "profile", "use", "ghost")
	if code == 0 || !strings.Contains(errOut, "not found") {
		t.Errorf("use ghost: code=%d stderr=%s", code, errOut)
	}
}

func TestConfigSetAPIKeyRedacted(t *testing.T) {
	dir := t.TempDir()
	secret := "sk-supersecretkey123456"
	out, _, code := runCLI(t, dir, "config", "set", "api-key", secret)
	if code != 0 {
		t.Fatalf("set failed: %s", out)
	}
	if strings.Contains(out, secret) {
		t.Errorf("api key echoed in plaintext: %s", out)
	}
	if !strings.Contains(out, "sk-s") || !strings.Contains(out, "****") {
		t.Errorf("expected redacted key in output: %s", out)
	}
	// but the key is stored intact
	data, err := os.ReadFile(filepath.Join(dir, "clai", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), secret) {
		t.Error("key not persisted to config file")
	}
}

func TestProfileFlagOverride(t *testing.T) {
	dir := t.TempDir()
	runCLI(t, dir, "config", "profile", "add", "a", "--provider", "openai", "--model", "gpt-4o-mini")
	runCLI(t, dir, "config", "profile", "add", "b", "--provider", "anthropic", "--model", "claude-haiku-4-5-20251001")

	// active is "a"; -p picks "b" without switching
	out, _, _ := runCLI(t, dir, "-p", "b", "config", "list")
	if !strings.Contains(out, "profile:   b") || !strings.Contains(out, "provider:  anthropic") {
		t.Errorf("-p override not applied:\n%s", out)
	}
	out, _, _ = runCLI(t, dir, "--profile=b", "config", "list")
	if !strings.Contains(out, "profile:   b") {
		t.Errorf("--profile= form not applied:\n%s", out)
	}
	// active unchanged
	out, _, _ = runCLI(t, dir, "config", "list")
	if !strings.Contains(out, "profile:   a") {
		t.Errorf("active profile changed by -p:\n%s", out)
	}
	// unknown -p errors
	_, errOut, code := runCLI(t, dir, "-p", "nope", "config", "list")
	if code == 0 || !strings.Contains(errOut, "not found") {
		t.Errorf("unknown -p: code=%d stderr=%s", code, errOut)
	}
}

func TestQueryStreamsThroughProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ls -la\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	dir := t.TempDir()
	runCLI(t, dir, "config", "profile", "add", "fake", "--provider", "openai", "--model", "m", "--api-key", "k", "--base-url", srv.URL)

	out, errOut, code := runCLI(t, dir, "-p", "fake", "list files")
	if code != 0 {
		t.Fatalf("query failed: %s", errOut)
	}
	if strings.TrimSpace(out) != "ls -la" {
		t.Errorf("query output = %q, want ls -la", out)
	}
}

func TestModelsListUsesProfile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"data":[{"id":"llama3.2"},{"id":"mistral"}]}`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	runCLI(t, dir, "config", "profile", "add", "local", "--provider", "ollama", "--model", "llama3.2", "--base-url", srv.URL)

	out, errOut, code := runCLI(t, dir, "models", "list")
	if code != 0 {
		t.Fatalf("models list failed: %s", errOut)
	}
	if !strings.Contains(out, "(profile: local)") {
		t.Errorf("missing profile name in header:\n%s", out)
	}
	if !strings.Contains(out, "llama3.2  (current)") || !strings.Contains(out, "mistral") {
		t.Errorf("model listing wrong:\n%s", out)
	}
}

func TestLegacyConfigMigratesOnFirstWrite(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "clai", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0700); err != nil {
		t.Fatal(err)
	}
	legacy := "provider: anthropic\napi_key: sk-ant-legacy12345\nmodel: claude-haiku-4-5-20251001\n"
	if err := os.WriteFile(cfgPath, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}

	// read-only command sees migrated view without rewriting the file
	out, _, _ := runCLI(t, dir, "config", "list")
	if !strings.Contains(out, "profile:   default") || !strings.Contains(out, "provider:  anthropic") {
		t.Errorf("legacy view wrong:\n%s", out)
	}

	// a write migrates the file format
	runCLI(t, dir, "config", "set", "model", "claude-opus-4-8")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "profiles:") || !strings.Contains(s, "default:") {
		t.Errorf("file not migrated:\n%s", s)
	}
	if strings.HasPrefix(s, "provider:") || strings.Contains(s, "\nprovider: anthropic\n") {
		// top-level legacy key must be gone (the nested one is indented)
		t.Errorf("legacy top-level keys remain:\n%s", s)
	}
	if !strings.Contains(s, "sk-ant-legacy12345") {
		t.Errorf("api key lost in migration:\n%s", s)
	}
}

func TestHelpMentionsProfiles(t *testing.T) {
	dir := t.TempDir()
	out, _, code := runCLI(t, dir, "--help")
	if code != 0 {
		t.Fatal("help failed")
	}
	for _, want := range []string{"profile add", "profile use", "-p, --profile"} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q", want)
		}
	}
}
