package provider

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnthropicStream(t *testing.T) {
	var gotKey, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: message_start\n")
		fmt.Fprint(w, "data: {\"type\":\"message_start\"}\n\n")
		fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"hel\"}}\n\n")
		fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"lo\"}}\n\n")
		fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	p := &anthropicProvider{apiKey: "sk-ant-test", apiKeyHeader: "x-api-key", model: "m", baseURL: srv.URL}
	var buf bytes.Buffer
	if err := p.Stream(context.Background(), "sys", "query", &buf); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "hello" {
		t.Errorf("streamed %q, want hello", buf.String())
	}
	if gotKey != "sk-ant-test" {
		t.Errorf("x-api-key = %q", gotKey)
	}
	if gotVersion != anthropicVersion {
		t.Errorf("anthropic-version = %q", gotVersion)
	}
}

func TestAnthropicStreamHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"invalid api key"}}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := &anthropicProvider{apiKey: "bad", apiKeyHeader: "x-api-key", model: "m", baseURL: srv.URL}
	err := p.Stream(context.Background(), "sys", "query", &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("err = %v, want 401", err)
	}
}

func TestAnthropicListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"data":[{"id":"claude-b"},{"id":"claude-a"}]}`)
	}))
	defer srv.Close()

	p := &anthropicProvider{apiKey: "k", apiKeyHeader: "x-api-key", model: "m", baseURL: srv.URL}
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0] != "claude-a" || models[1] != "claude-b" {
		t.Errorf("models = %v, want sorted [claude-a claude-b]", models)
	}
}
