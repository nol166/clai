package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/johnmccambridge/clai/internal/config"
)

const (
	anthropicBase    = "https://api.anthropic.com/v1"
	anthropicVersion = "2023-06-01"
)

type anthropicProvider struct {
	apiKey string
	model  string
}

func newAnthropic(cfg *config.Config) *anthropicProvider {
	return &anthropicProvider{apiKey: cfg.APIKey, model: cfg.Model}
}

func (p *anthropicProvider) Stream(ctx context.Context, system, query string, w io.Writer) error {
	body, _ := json.Marshal(map[string]any{
		"model":      p.model,
		"max_tokens": 2048,
		"stream":     true,
		"system":     system,
		"messages": []map[string]string{
			{"role": "user", "content": query},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicBase+"/messages", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic %d: %s", resp.StatusCode, b)
	}

	type delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type event struct {
		Type  string `json:"type"`
		Delta *delta `json:"delta"`
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var e event
		if err := json.Unmarshal([]byte(line[6:]), &e); err != nil {
			continue
		}
		if e.Type == "content_block_delta" && e.Delta != nil && e.Delta.Type == "text_delta" {
			fmt.Fprint(w, e.Delta.Text)
		}
	}
	return scanner.Err()
}

func (p *anthropicProvider) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, anthropicBase+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic %d: %s", resp.StatusCode, b)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	sort.Strings(models)
	return models, nil
}
