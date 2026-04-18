package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/johnmccambridge/clai/internal/config"
	"github.com/johnmccambridge/clai/internal/provider"
)

var version = "dev"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: clai <query>")
		fmt.Fprintln(os.Stderr, "       clai config [list | set <key> <value>]")
		fmt.Fprintln(os.Stderr, "       clai models list")
		os.Exit(1)
	}

	// extract -c/--copy flag
	copyFlag := false
	filtered := args[:0]
	for _, a := range args {
		if a == "-c" || a == "--copy" {
			copyFlag = true
		} else {
			filtered = append(filtered, a)
		}
	}
	args = filtered

	switch args[0] {
	case "--version", "-v":
		fmt.Println("clai", version)
		return
	case "--help", "-h":
		printHelp()
		return
	case "config":
		runConfig(args[1:])
		return
	case "models":
		runModels(args[1:])
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	p, err := provider.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cwd, _ := os.Getwd()
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "unknown"
	}

	query := strings.Join(args, " ")
	systemPrompt := buildSystemPrompt(cwd, shell)

	shouldCopy := copyFlag || cfg.Clipboard

	var buf bytes.Buffer
	w := io.Writer(os.Stdout)
	if shouldCopy {
		w = io.MultiWriter(os.Stdout, &buf)
	}

	ctx := context.Background()
	if err := p.Stream(ctx, systemPrompt, query, w); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()

	if shouldCopy {
		if err := copyToClipboard(buf.String()); err != nil {
			fmt.Fprintf(os.Stderr, "clipboard: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "copied to clipboard")
		}
	}
}

func buildSystemPrompt(cwd, shell string) string {
	return fmt.Sprintf(`You are a terminal assistant. Answer questions about shell commands, CLI tools, code, and developer tasks.

Rules:
- Return ONLY the answer — no preamble, no explanation unless the user asks "why" or "explain"
- If the answer is a command, return just the command
- For aliases or shell functions, return just the definition
- No markdown formatting: no **, no ##, no bullet dashes
- If multiple steps are required, use brief numbered lines
- Assume the user is an experienced developer

Context:
- OS: %s
- Shell: %s
- Working directory: %s`, runtime.GOOS, shell, cwd)
}
