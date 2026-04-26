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

func runConfig(args []string) {
	if len(args) == 0 {
		runConfigInteractive()
		return
	}
	switch args[0] {
	case "list":
		runConfigList()
	case "set":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: clai config set <key> <value>")
			fmt.Fprintln(os.Stderr, "keys: provider, model, api-key, base-url")
			os.Exit(1)
		}
		runConfigSet(args[1], args[2])
	case "clipboard":
		runConfigClipboardToggle()
	default:
		fmt.Fprintf(os.Stderr, "unknown config command %q\n", args[0])
		os.Exit(1)
	}
}

func runConfigClipboardToggle() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	cfg.Clipboard = !cfg.Clipboard
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error saving: %v\n", err)
		os.Exit(1)
	}
	if cfg.Clipboard {
		fmt.Println("clipboard: on — responses will always be copied")
	} else {
		fmt.Println("clipboard: off — use -c to copy ad hoc")
	}
}

func runConfigList() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	path := config.Path()
	redacted := ""
	if cfg.APIKey != "" {
		if len(cfg.APIKey) > 8 {
			redacted = cfg.APIKey[:4] + strings.Repeat("*", len(cfg.APIKey)-8) + cfg.APIKey[len(cfg.APIKey)-4:]
		} else {
			redacted = "****"
		}
	}
	fmt.Printf("provider:  %s\n", cfg.Provider)
	fmt.Printf("model:     %s\n", cfg.Model)
	fmt.Printf("api_key:   %s\n", redacted)
	fmt.Printf("base_url:  %s\n", cfg.BaseURL)
	fmt.Printf("clipboard: %v\n", cfg.Clipboard)
	fmt.Printf("config:    %s\n", path)
}

func runConfigSet(key, value string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}
	switch strings.ToLower(strings.ReplaceAll(key, "-", "_")) {
	case "provider":
		if !isValidProvider(value) {
			fmt.Fprintf(os.Stderr, "unknown provider %q — valid: openai, anthropic, litellm, ollama\n", value)
			os.Exit(1)
		}
		cfg.Provider = value
		if value == "ollama" {
			cfg.APIKey = ""
		}
		if cfg.Model == "" || isDefaultModel(cfg.Model) {
			cfg.Model = defaultModelFor(value)
		}
	case "model":
		cfg.Model = value
	case "api_key":
		cfg.APIKey = value
	case "base_url":
		cfg.BaseURL = value
	default:
		fmt.Fprintf(os.Stderr, "unknown key %q — valid: provider, model, api-key, base-url\n", key)
		os.Exit(1)
	}
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error saving config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("set %s = %s\n", key, value)
}

func runConfigInteractive() {
	cfg, _ := config.Load()
	reader := bufio.NewReader(os.Stdin)

	printLogo()
	fmt.Println("Configure clai — press Enter to keep current value")
	fmt.Println()

	// provider
	fmt.Printf("Provider (openai/anthropic/litellm/ollama) [%s]: ", cfg.Provider)
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

	// base url (only relevant for litellm/ollama) — before model so we can query the provider
	if cfg.Provider == "litellm" || cfg.Provider == "ollama" {
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
	if p, err := provider.New(cfg); err == nil {
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

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error saving: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nSaved to %s\n", config.Path())
}

func readLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}

func isValidProvider(p string) bool {
	switch p {
	case "openai", "anthropic", "litellm", "ollama":
		return true
	}
	return false
}

func isDefaultModel(m string) bool {
	for _, p := range []string{"openai", "anthropic", "litellm", "ollama"} {
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
