package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nol166/clai/internal/config"
	"github.com/nol166/clai/internal/provider"
)

func runModels(args []string) {
	if len(args) == 0 || args[0] != "list" {
		fmt.Fprintln(os.Stderr, "usage: clai models list")
		os.Exit(1)
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

	models, err := p.ListModels(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing models: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Models for %s:\n\n", cfg.Provider)
	for _, m := range models {
		if m == cfg.Model {
			fmt.Printf("  %s  (current)\n", m)
		} else {
			fmt.Printf("  %s\n", m)
		}
	}
}
