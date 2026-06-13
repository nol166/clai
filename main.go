package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/nol166/clai/internal/config"
	"github.com/nol166/clai/internal/provider"
)

var version = "dev"

func getVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		v := info.Main.Version
		// only use clean semver tags, not pseudo-versions from local builds
		if len(v) > 0 && v[0] == 'v' && !strings.Contains(v, "-0.") {
			return v
		}
	}
	return version
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printHelp()
		os.Exit(0)
	}

	// extract -c/--copy and -p/--profile flags
	copyFlag := false
	profileFlag := ""
	filtered := args[:0]
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-c" || a == "--copy":
			copyFlag = true
		case a == "-p" || a == "--profile":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: -p/--profile requires a name")
				os.Exit(1)
			}
			i++
			profileFlag = args[i]
		case strings.HasPrefix(a, "--profile="):
			profileFlag = strings.TrimPrefix(a, "--profile=")
		default:
			filtered = append(filtered, a)
		}
	}
	args = filtered
	if len(args) == 0 {
		printHelp()
		os.Exit(0)
	}

	switch args[0] {
	case "--version", "-v":
		fmt.Println("clai", getVersion())
		return
	case "--help", "-h":
		printHelp()
		return
	case "config":
		runConfig(args[1:], profileFlag)
		return
	case "models":
		runModels(args[1:], profileFlag)
		return
	}

	cfg, err := config.LoadProfile(profileFlag)
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
