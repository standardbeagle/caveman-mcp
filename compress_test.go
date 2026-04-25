package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

const sampleText = `The purpose of this document is to provide a comprehensive overview of the
key features and functionality that are essentially available in the system.
Actually, the main thing you need to understand is basically that the system
works by processing requests from users and then sending them to the appropriate
service handler. The handler will typically process the request and return a
result. This is generally how most modern web services work.

Here is an example of some code:
` + "```go\nfunc handleRequest(w http.ResponseWriter, r *http.Request) {\n\tres := processRequest(r)\n\tw.Write(res)\n}\n```" + `

The URL https://example.com/api/v1/endpoint is the primary endpoint for this service.
Make sure to set the SOME_API_KEY environment variable before starting.`

func TestMechanical(t *testing.T) {
	comp := NewCompressor(Config{})
	r, err := comp.Condense(context.Background(), sampleText, false)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Method: %s", r.Method)
	t.Logf("Original: %d chars", r.OriginalChars)
	t.Logf("Compressed: %d chars", r.CompressedChars)
	t.Logf("Ratio: %.1f%%", r.Ratio*100)
	t.Logf("Result:\n%s", r.Compressed)

	// Code block must survive intact
	if !strings.Contains(r.Compressed, "func handleRequest") {
		t.Error("code block not preserved")
	}
	// URL must survive
	if !strings.Contains(r.Compressed, "https://example.com/api/v1/endpoint") {
		t.Error("URL not preserved")
	}
	// Drop words removed
	if strings.Contains(r.Compressed, " the ") || strings.Contains(r.Compressed, " basically ") {
		t.Error("drop words not removed")
	}
}

func TestLLMWenyan(t *testing.T) {
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		t.Skip("no LLM_API_KEY set")
	}

	cfg := Config{
		BaseURL: envOrDefault("LLM_BASE_URL", "https://openrouter.ai/api/v1"),
		APIKey:  apiKey,
		Model:   envOrDefault("LLM_MODEL", "anthropic/claude-haiku-4-5"),
	}
	comp := NewCompressor(cfg)

	r, err := comp.Condense(context.Background(), sampleText, true)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Method: %s", r.Method)
	t.Logf("Original: %d chars", r.OriginalChars)
	t.Logf("Compressed: %d chars", r.CompressedChars)
	t.Logf("Ratio: %.1f%%", r.Ratio*100)
	t.Logf("Result:\n%s", r.Compressed)

	if r.Compressed == "" {
		t.Error("empty compressed output")
	}
	if r.CompressedChars >= r.OriginalChars {
		t.Errorf("no compression: %d >= %d", r.CompressedChars, r.OriginalChars)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
