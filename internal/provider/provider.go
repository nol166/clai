package provider

import (
	"context"
	"fmt"
	"io"

	"github.com/johnmccambridge/clai/internal/config"
)

// Provider streams a response and can list available models.
type Provider interface {
	Stream(ctx context.Context, system, query string, w io.Writer) error
	ListModels(ctx context.Context) ([]string, error)
}

func New(cfg *config.Config) (Provider, error) {
	switch cfg.Provider {
	case "openai", "litellm", "ollama":
		return newOpenAI(cfg), nil
	case "anthropic":
		return newAnthropic(cfg), nil
	default:
		return nil, fmt.Errorf("unknown provider %q — valid: openai, anthropic, litellm, ollama", cfg.Provider)
	}
}
