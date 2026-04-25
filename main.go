package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ── tool arg structs ──────────────────────────────────────────────────────────

type CondenseURLArgs struct {
	URL     string `json:"url"               jsonschema:"Webpage URL to fetch and condense"`
	SkipLLM bool   `json:"skip_llm,omitempty" jsonschema:"Skip LLM Wenyan pass; use mechanical compression only"`
}

type CondenseFileArgs struct {
	Path    string `json:"path"              jsonschema:"Absolute path to file (md/pdf/docx/html/txt)"`
	SkipLLM bool   `json:"skip_llm,omitempty" jsonschema:"Skip LLM Wenyan pass; use mechanical compression only"`
}

type CondenseTextArgs struct {
	Text    string `json:"text"              jsonschema:"Raw text to condense"`
	SkipLLM bool   `json:"skip_llm,omitempty" jsonschema:"Skip LLM Wenyan pass; use mechanical compression only"`
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	cfg := Config{
		BaseURL: envOr("LLM_BASE_URL", "https://openrouter.ai/api/v1"),
		APIKey:  envOr("LLM_API_KEY", envOr("OPENROUTER_API_KEY", os.Getenv("MINIMAX_API_KEY"))),
		Model:   envOr("LLM_MODEL", "anthropic/claude-haiku-4-5"),
	}
	if cfg.APIKey == "" {
		log.Println("warning: no LLM_API_KEY set; LLM Wenyan pass disabled")
	}

	comp := NewCompressor(cfg)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "caveman-mcp",
		Version: "0.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "condense_url",
		Description: "Fetch a webpage, extract main content, and condense it to ultra-compressed Wenyan classical Chinese (mechanical pre-pass + LLM). Returns compressed text with stats.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args CondenseURLArgs) (*mcp.CallToolResult, any, error) {
		text, err := ExtractURL(ctx, args.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("extract URL %s: %w", args.URL, err)
		}
		r, err := comp.Condense(ctx, text, !args.SkipLLM)
		if err != nil {
			return nil, nil, err
		}
		return resultContent(r), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "condense_file",
		Description: "Read a local file (md/pdf/docx/html/txt), extract text, and condense to Wenyan. Returns compressed text with stats.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args CondenseFileArgs) (*mcp.CallToolResult, any, error) {
		text, err := ExtractFile(args.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("extract file %s: %w", args.Path, err)
		}
		r, err := comp.Condense(ctx, text, !args.SkipLLM)
		if err != nil {
			return nil, nil, err
		}
		return resultContent(r), nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "condense_text",
		Description: "Condense raw text to Wenyan ultra-compressed form. Returns compressed text with stats.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args CondenseTextArgs) (*mcp.CallToolResult, any, error) {
		r, err := comp.Condense(ctx, args.Text, !args.SkipLLM)
		if err != nil {
			return nil, nil, err
		}
		return resultContent(r), nil, nil
	})

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func resultContent(r *Result) *mcp.CallToolResult {
	type out struct {
		Compressed      string  `json:"compressed"`
		Method          string  `json:"method"`
		OriginalChars   int     `json:"original_chars"`
		CompressedChars int     `json:"compressed_chars"`
		Ratio           string  `json:"ratio"`
	}
	o := out{
		Compressed:      r.Compressed,
		Method:          r.Method,
		OriginalChars:   r.OriginalChars,
		CompressedChars: r.CompressedChars,
		Ratio:           fmt.Sprintf("%.1f%%", r.Ratio*100),
	}
	b, _ := json.MarshalIndent(o, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
