package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/nol166/clai/internal/config"
	"github.com/nol166/clai/internal/provider"
)

// profileArg is the -p/--profile override extracted in main; empty
// means "use the active profile".
func runConfig(args []string, profileArg string) {
	if len(args) == 0 {
		runConfigInteractive(profileArg)
		return
	}
	switch args[0] {
	case "list":
		runConfigList(profileArg)
	case "set":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: clai config set <key> <value>")
			fmt.Fprintln(os.Stderr, "keys: provider, model, api-key, base-url, api-key-header")
			os.Exit(1)
		}
		runConfigSet(args[1], args[2], profileArg)
	case "clipboard":
		runConfigClipboardToggle()
	case "profile":
		runConfigProfile(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown config command %q\n", args[0])
		os.Exit(1)
	}
}

func runConfigProfile(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: clai config profile <add|update|use|list|delete> [name]")
		os.Exit(1)
	}
	switch args[0] {
	case "list":
		runProfileList()
	case "use":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: clai config profile use <name>")
			os.Exit(1)
		}
		runProfileUse(args[1])
	case "add":
		if len(args) < 2 || strings.HasPrefix(args[1], "-") {
			fmt.Fprintln(os.Stderr, "usage: clai config profile add <name> [--provider p] [--model m] [--api-key k] [--base-url u]")
			os.Exit(1)
		}
		runProfileAdd(args[1], args[2:])
	case "update":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: clai config profile update <name> [--provider p] [--model m] [--api-key k] [--base-url u]")
			os.Exit(1)
		}
		runProfileUpdate(args[1], args[2:])
	case "delete", "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: clai config profile delete <name>")
			os.Exit(1)
		}
		runProfileDelete(args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown profile command %q — valid: add, update, use, list, delete\n", args[0])
		os.Exit(1)
	}
}

func runProfileList() {
	f, err := config.LoadFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	names := f.ProfileNames()
	if len(names) == 0 {
		fmt.Println("no profiles — create one with: clai config profile add <name>")
		return
	}
	active := f.ActiveName()
	for _, n := range names {
		p := f.Profiles[n]
		marker := " "
		if n == active {
			marker = "*"
		}
		fmt.Printf("%s %-12s provider=%s model=%s\n", marker, n, p.Provider, p.Model)
	}
}

func runProfileUse(name string) {
	f, err := config.LoadFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if _, ok := f.Profiles[name]; !ok {
		fmt.Fprintf(os.Stderr, "profile %q not found (have: %v)\n", name, f.ProfileNames())
		os.Exit(1)
	}
	f.Active = name
	if err := config.SaveFile(f); err != nil {
		fmt.Fprintf(os.Stderr, "error saving: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("active profile: %s\n", name)
}

func runProfileAdd(name string, flags []string) {
	f, err := config.LoadFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	p := f.Profiles[name]

	if len(flags) == 0 {
		// no flags — run the interactive wizard for this profile
		runConfigInteractive(name)
		return
	}

	for i := 0; i < len(flags); i++ {
		flag := flags[i]
		var value string
		if eq := strings.Index(flag, "="); eq >= 0 {
			flag, value = flag[:eq], flag[eq+1:]
		} else {
			if i+1 >= len(flags) {
				fmt.Fprintf(os.Stderr, "flag %s requires a value\n", flag)
				os.Exit(1)
			}
			i++
			value = flags[i]
		}
		switch flag {
		case "--provider":
			if !isValidProvider(value) {
				fmt.Fprintf(os.Stderr, "unknown provider %q — valid: openai, anthropic, litellm, ollama, openrouter\n", value)
				os.Exit(1)
			}
			p.Provider = value
		case "--model":
			p.Model = value
		case "--api-key":
			p.APIKey = value
		case "--base-url":
			p.BaseURL = value
		default:
			fmt.Fprintf(os.Stderr, "unknown flag %q — valid: --provider, --model, --api-key, --base-url\n", flag)
			os.Exit(1)
		}
	}

	if p.Provider == "" {
		p.Provider = "openai"
	}
	if p.Provider == "ollama" {
		p.APIKey = ""
		if p.BaseURL == "" {
			p.BaseURL = defaultBaseURLFor("ollama")
		}
	}
	if p.Model == "" {
		p.Model = defaultModelFor(p.Provider)
	}

	f.Profiles[name] = p
	if f.Active == "" {
		f.Active = name
	}
	if err := config.SaveFile(f); err != nil {
		fmt.Fprintf(os.Stderr, "error saving: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("saved profile %q (provider=%s model=%s)\n", name, p.Provider, p.Model)
	if f.Active != name {
		fmt.Printf("switch to it with: clai config profile use %s\n", name)
	}
}

func runProfileUpdate(name string, flags []string) {
	f, err := config.LoadFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if _, ok := f.Profiles[name]; !ok {
		fmt.Fprintf(os.Stderr, "profile %q not found (have: %v)\n", name, f.ProfileNames())
		os.Exit(1)
	}
	if len(flags) == 0 {
		// no flags — run the interactive wizard for this profile
		runConfigInteractive(name)
		return
	}

	p := f.Profiles[name]
	for i := 0; i < len(flags); i++ {
		flag := flags[i]
		var value string
		if eq := strings.Index(flag, "="); eq >= 0 {
			flag, value = flag[:eq], flag[eq+1:]
		} else {
			if i+1 >= len(flags) {
				fmt.Fprintf(os.Stderr, "flag %s requires a value\n", flag)
				os.Exit(1)
			}
			i++
			value = flags[i]
		}
		shown := value
		switch flag {
		case "--provider":
			if !isValidProvider(value) {
				fmt.Fprintf(os.Stderr, "unknown provider %q — valid: openai, anthropic, litellm, ollama, openrouter\n", value)
				os.Exit(1)
			}
			p.Provider = value
		case "--model":
			p.Model = value
		case "--api-key":
			p.APIKey = value
			shown = redactKey(value)
		case "--base-url":
			p.BaseURL = value
		case "--api-key-header":
			p.APIKeyHeader = value
		default:
			fmt.Fprintf(os.Stderr, "unknown flag %q — valid: --provider, --model, --api-key, --base-url\n", flag)
			os.Exit(1)
		}
		if p.Provider == "ollama" {
			p.APIKey = ""
		}
		fmt.Printf("updated %s = %s\n", flag, shown)
	}

	f.Profiles[name] = p
	if err := config.SaveFile(f); err != nil {
		fmt.Fprintf(os.Stderr, "error saving: %v\n", err)
		os.Exit(1)
	}
}

func runProfileDelete(name string) {
	f, err := config.LoadFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if _, ok := f.Profiles[name]; !ok {
		fmt.Fprintf(os.Stderr, "profile %q not found (have: %v)\n", name, f.ProfileNames())
		os.Exit(1)
	}
	if f.ActiveName() == name {
		fmt.Fprintf(os.Stderr, "profile %q is active — switch first: clai config profile use <other>\n", name)
		os.Exit(1)
	}
	delete(f.Profiles, name)
	if err := config.SaveFile(f); err != nil {
		fmt.Fprintf(os.Stderr, "error saving: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("deleted profile %q\n", name)
}

func runConfigClipboardToggle() {
	f, err := config.LoadFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	f.Clipboard = !f.Clipboard
	if err := config.SaveFile(f); err != nil {
		fmt.Fprintf(os.Stderr, "error saving: %v\n", err)
		os.Exit(1)
	}
	if f.Clipboard {
		fmt.Println("clipboard: on — responses will always be copied")
	} else {
		fmt.Println("clipboard: off — use -c to copy ad hoc")
	}
}

func runConfigList(profileArg string) {
	cfg, err := config.LoadProfile(profileArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	f, err := config.LoadFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("profile:   %s\n", cfg.ProfileName)
	fmt.Printf("provider:  %s\n", cfg.Provider)
	fmt.Printf("model:     %s\n", cfg.Model)
	fmt.Printf("api_key:   %s\n", redactKey(cfg.APIKey))
	fmt.Printf("base_url:  %s\n", cfg.BaseURL)
	fmt.Printf("clipboard: %v\n", cfg.Clipboard)
	fmt.Printf("config:    %s\n", config.Path())
	if names := f.ProfileNames(); len(names) > 1 {
		fmt.Printf("profiles:  %s\n", strings.Join(names, ", "))
	}
}

func redactKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) > 8 {
		return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
	}
	return "****"
}

func runConfigSet(key, value, profileArg string) {
	f, err := config.LoadFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}
	name := profileArg
	if name == "" {
		name = f.ActiveName()
	}
	p, ok := f.Profiles[name]
	if !ok && name != config.DefaultProfile {
		fmt.Fprintf(os.Stderr, "profile %q not found (have: %v)\n", name, f.ProfileNames())
		os.Exit(1)
	}

	shown := value
	switch strings.ToLower(strings.ReplaceAll(key, "-", "_")) {
	case "provider":
		if !isValidProvider(value) {
			fmt.Fprintf(os.Stderr, "unknown provider %q — valid: openai, anthropic, litellm, ollama, openrouter\n", value)
			os.Exit(1)
		}
		p.Provider = value
		if value == "ollama" {
			p.APIKey = ""
		}
		if p.Model == "" || isDefaultModel(p.Model) {
			p.Model = defaultModelFor(value)
		}
	case "model":
		p.Model = value
	case "api_key":
		p.APIKey = value
		shown = redactKey(value)
	case "base_url":
		p.BaseURL = value
	case "api_key_header":
		p.APIKeyHeader = value
	default:
		fmt.Fprintf(os.Stderr, "unknown key %q — valid: provider, model, api-key, base-url, api-key-header\n", key)
		os.Exit(1)
	}
	f.Profiles[name] = p
	if f.Active == "" {
		f.Active = name
	}
	if err := config.SaveFile(f); err != nil {
		fmt.Fprintf(os.Stderr, "error saving config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[%s] set %s = %s\n", name, key, shown)
}

func runConfigInteractive(profileArg string) {
	f, err := config.LoadFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	name := profileArg
	if name == "" {
		name = f.ActiveName()
	}
	cfg := f.Profiles[name]
	if cfg.Provider == "" {
		cfg.Provider = "openai"
	}
	reader := bufio.NewReader(os.Stdin)

	printLogo()
	fmt.Printf("Configure clai (profile: %s) — press Enter to keep current value\n", name)
	fmt.Println()

	// provider
	fmt.Printf("Provider (openai/anthropic/litellm/ollama/openrouter) [%s]: ", cfg.Provider)
	if p := readLine(reader); p != "" {
		if !isValidProvider(p) {
			fmt.Fprintf(os.Stderr, "unknown provider %q\n", p)
			os.Exit(1)
		}
		cfg.Provider = p
	}

	// api key (hidden input, skip for ollama)
	if cfg.Provider != "ollama" {
		keyHint := ""
		if cfg.APIKey != "" {
			keyHint = " [current key kept if empty]"
		}
		fmt.Printf("API key%s: ", keyHint)
		keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			// fallback to plain input
			fmt.Print("API key: ")
			if k := readLine(reader); k != "" {
				cfg.APIKey = k
			}
		} else if len(keyBytes) > 0 {
			cfg.APIKey = string(keyBytes)
		}
	} else {
		cfg.APIKey = ""
	}

	// base url (only relevant for litellm/ollama/openrouter) — before model so we can query the provider
	if cfg.Provider == "litellm" || cfg.Provider == "ollama" || cfg.Provider == "openrouter" {
		defaultURL := defaultBaseURLFor(cfg.Provider)
		hint := cfg.BaseURL
		if hint == "" {
			hint = defaultURL
		}
		fmt.Printf("Base URL [%s]: ", hint)
		if u := readLine(reader); u != "" {
			cfg.BaseURL = u
		} else if cfg.BaseURL == "" {
			cfg.BaseURL = defaultURL
		}
	}

	// model — try to fetch live list from provider; fall back to free-form
	defaultModel := defaultModelFor(cfg.Provider)
	current := cfg.Model
	if current == "" {
		current = defaultModel
	}
	var liveModels []string
	probe := &config.Config{
		Provider: cfg.Provider,
		APIKey:   cfg.APIKey,
		Model:    cfg.Model,
		BaseURL:  cfg.BaseURL,
	}
	if p, err := provider.New(probe); err == nil {
		liveModels, _ = p.ListModels(context.Background())
	}
	if len(liveModels) > 0 {
		fmt.Println("Available models:")
		for i, m := range liveModels {
			marker := ""
			if m == current {
				marker = "  *"
			}
			fmt.Printf("  %d) %s%s\n", i+1, m, marker)
		}
		fmt.Printf("Model (number or name) [%s]: ", current)
		if m := readLine(reader); m != "" {
			if idx, err := strconv.Atoi(m); err == nil && idx >= 1 && idx <= len(liveModels) {
				cfg.Model = liveModels[idx-1]
			} else {
				cfg.Model = m
			}
		} else if cfg.Model == "" {
			cfg.Model = defaultModel
		}
	} else {
		fmt.Printf("Model [%s]: ", current)
		if m := readLine(reader); m != "" {
			cfg.Model = m
		} else if cfg.Model == "" {
			cfg.Model = defaultModel
		}
	}

	f.Profiles[name] = cfg
	if f.Active == "" {
		f.Active = name
	}
	if err := config.SaveFile(f); err != nil {
		fmt.Fprintf(os.Stderr, "error saving: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nSaved profile %q to %s\n", name, config.Path())
}

func readLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}

func isValidProvider(p string) bool {
	switch p {
	case "openai", "anthropic", "litellm", "ollama", "openrouter":
		return true
	}
	return false
}

func isDefaultModel(m string) bool {
	for _, p := range []string{"openai", "anthropic", "litellm", "ollama", "openrouter"} {
		if defaultModelFor(p) == m {
			return true
		}
	}
	return false
}

func defaultModelFor(provider string) string {
	switch provider {
	case "anthropic":
		return "claude-haiku-4-5-20251001"
	case "ollama":
		return ""
	case "openrouter":
		return "openai/gpt-4o-mini"
	default:
		return "gpt-4o-mini"
	}
}

func defaultBaseURLFor(provider string) string {
	switch provider {
	case "ollama":
		return "http://localhost:11434/v1"
	default:
		return ""
	}
}
