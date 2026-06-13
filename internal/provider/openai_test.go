package provider

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nol166/clai/internal/config"
)

func TestOpenAIStream(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hel\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n")
		fmt.Fprint(w, ": comment line ignored\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p := newOpenAI(&config.Config{Provider: "openai", APIKey: "sk-test", Model: "m", BaseURL: srv.URL})
	var buf bytes.Buffer
	if err := p.Stream(context.Background(), "sys", "query", &buf); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "hello" {
		t.Errorf("streamed %q, want hello", buf.String())
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotPath != "/chat/completions" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestOpenAIStreamNoAuthHeaderWithoutKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p := newOpenAI(&config.Config{Provider: "ollama", Model: "m", BaseURL: srv.URL})
	var buf bytes.Buffer
	if err := p.Stream(context.Background(), "sys", "query", &buf); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "" {
		t.Errorf("ollama sent auth header %q, want none", gotAuth)
	}
}

func TestOpenAIStreamHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"bad key"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newOpenAI(&config.Config{Provider: "openai", Model: "m", BaseURL: srv.URL})
	err := p.Stream(context.Background(), "sys", "query", &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("err = %v, want 401", err)
	}
}

func TestOpenAIStreamLongChunk(t *testing.T) {
	// single SSE line larger than bufio.Scanner's 64K default cap
	long := strings.Repeat("x", 100*1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"%s\"}}]}\n\n", long)
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p := newOpenAI(&config.Config{Provider: "openai", Model: "m", BaseURL: srv.URL})
	var buf bytes.Buffer
	if err := p.Stream(context.Background(), "sys", "query", &buf); err != nil {
		t.Fatalf("long chunk failed: %v", err)
	}
	if buf.Len() != len(long) {
		t.Errorf("streamed %d bytes, want %d", buf.Len(), len(long))
	}
}

func TestOpenAIListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"data":[{"id":"zeta"},{"id":"alpha"}]}`)
	}))
	defer srv.Close()

	p := newOpenAI(&config.Config{Provider: "openai", Model: "m", BaseURL: srv.URL})
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0] != "alpha" || models[1] != "zeta" {
		t.Errorf("models = %v, want sorted [alpha zeta]", models)
	}
}

func TestNewProviderDispatch(t *testing.T) {
	for _, name := range []string{"openai", "litellm", "ollama", "anthropic"} {
		if _, err := New(&config.Config{Provider: name}); err != nil {
			t.Errorf("New(%s) failed: %v", name, err)
		}
	}
	if _, err := New(&config.Config{Provider: "bogus"}); err == nil {
		t.Error("New(bogus) should fail")
	}
}

func TestOllamaDefaultBaseURL(t *testing.T) {
	p := newOpenAI(&config.Config{Provider: "ollama"})
	if p.baseURL != defaultOllamaBase {
		t.Errorf("baseURL = %q, want %q", p.baseURL, defaultOllamaBase)
	}
	p = newOpenAI(&config.Config{Provider: "openai"})
	if p.baseURL != defaultOpenAIBase {
		t.Errorf("baseURL = %q, want %q", p.baseURL, defaultOpenAIBase)
	}
}
