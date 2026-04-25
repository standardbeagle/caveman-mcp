package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"
)

type Config struct {
	BaseURL string
	APIKey  string
	Model   string
}

type Result struct {
	Compressed      string
	OriginalChars   int
	CompressedChars int
	Ratio           float64
	Method          string
}

type Compressor struct {
	cfg    Config
	client *http.Client
}

func NewCompressor(cfg Config) *Compressor {
	return &Compressor{cfg: cfg, client: &http.Client{Timeout: 90 * time.Second}}
}

func (c *Compressor) Condense(ctx context.Context, text string, useLLM bool) (*Result, error) {
	original := len(text)
	mech := mechanical(text)

	if !useLLM || c.cfg.APIKey == "" {
		return &Result{
			Compressed:      mech,
			OriginalChars:   original,
			CompressedChars: len(mech),
			Ratio:           ratio(len(mech), original),
			Method:          "mechanical",
		}, nil
	}

	wenyan, err := c.llmWenyan(ctx, mech)
	if err != nil {
		return nil, fmt.Errorf("llm wenyan pass: %w", err)
	}

	return &Result{
		Compressed:      wenyan,
		OriginalChars:   original,
		CompressedChars: len(wenyan),
		Ratio:           ratio(len(wenyan), original),
		Method:          "mechanical+llm",
	}, nil
}

// ── mechanical ────────────────────────────────────────────────────────────────

var (
	codeBlockRe = regexp.MustCompile("(?s)```[\\s\\S]*?```|`[^`\n]+`")
	urlRe       = regexp.MustCompile(`https?://\S+`)
	multiSpaceRe = regexp.MustCompile(`[ \t]{2,}`)
	multiNewlineRe = regexp.MustCompile(`\n{3,}`)

	dropWords = map[string]bool{
		"a": true, "an": true, "the": true,
		"just": true, "really": true, "basically": true,
		"actually": true, "simply": true, "very": true,
		"quite": true, "rather": true, "somewhat": true,
		"essentially": true, "generally": true, "typically": true,
		"usually": true, "often": true,
		"sure": true, "certainly": true, "please": true,
		"honestly": true, "clearly": true,
	}
)

func mechanical(text string) string {
	markers := map[string]string{}
	n := 0

	protect := func(s string) string {
		k := fmt.Sprintf("⟪%d⟫", n)
		n++
		markers[k] = s
		return k
	}

	out := codeBlockRe.ReplaceAllStringFunc(text, protect)
	out = urlRe.ReplaceAllStringFunc(out, protect)

	lines := strings.Split(out, "\n")
	for i, line := range lines {
		words := strings.Fields(line)
		kept := words[:0]
		for _, w := range words {
			bare := strings.TrimFunc(w, func(r rune) bool {
				return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '⟪' && r != '⟫'
			})
			if !dropWords[strings.ToLower(bare)] {
				kept = append(kept, w)
			}
		}
		lines[i] = strings.Join(kept, " ")
	}
	out = strings.Join(lines, "\n")

	out = multiSpaceRe.ReplaceAllString(out, " ")
	out = multiNewlineRe.ReplaceAllString(out, "\n\n")

	for k, v := range markers {
		out = strings.ReplaceAll(out, k, v)
	}

	return strings.TrimSpace(out)
}

// ── llm wenyan pass ───────────────────────────────────────────────────────────

const wenyanPrompt = `You are a Wenyan compression engine. Convert input text to ultra-compressed 文言文 (classical Chinese literary form).

STRICT RULES:
1. PRESERVE EXACTLY (no translation, no modification):
   - Code blocks: ` + "```...```" + ` and inline ` + "`code`" + `
   - URLs
   - camelCase/snake_case/PascalCase identifiers
   - File paths, error messages in quotes, version strings, numbers
2. Convert ALL natural language prose → dense 文言文
3. Classical structure: noun-verb-object, classical particles (之乎者也矣焉耳哉)
4. MAXIMUM compression — every character must earn its place
5. Output ONLY the compressed text — zero preamble, zero explanation

Text to compress:
`

type llmReq struct {
	Model    string       `json:"model"`
	Messages []llmMessage `json:"messages"`
}

type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llmResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

func (c *Compressor) llmWenyan(ctx context.Context, text string) (string, error) {
	body, _ := json.Marshal(llmReq{
		Model: c.cfg.Model,
		Messages: []llmMessage{
			{Role: "user", Content: wenyanPrompt + text},
		},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var r llmResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", fmt.Errorf("decode response: %w (body: %.200s)", err, raw)
	}
	if r.Error != nil {
		return "", fmt.Errorf("llm api error: %s", r.Error.Message)
	}
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("no choices in response (body: %.200s)", raw)
	}

	return strings.TrimSpace(r.Choices[0].Message.Content), nil
}

func ratio(compressed, original int) float64 {
	if original == 0 {
		return 0
	}
	return float64(compressed) / float64(original)
}
