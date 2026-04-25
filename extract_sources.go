package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-shiori/go-readability"
)

func extractYouTube(ctx context.Context, rawURL string) (string, error) {
	return "", nil // TODO
}

func extractGitHub(ctx context.Context, u *url.URL) (string, error) {
	return "", nil // TODO
}

func extractArXiv(ctx context.Context, rawURL string) (string, error) {
	return "", nil // TODO
}

func extractHN(ctx context.Context, id string) (string, error) {
	return "", nil // TODO
}

func extractReddit(ctx context.Context, rawURL string) (string, error) {
	return "", nil // TODO
}

func extractRSS(ctx context.Context, rawURL string) (string, error) {
	return "", nil // TODO
}

func extractHTMLPage(ctx context.Context, rawURL string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; caveman-mcp/0.1)")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	article, err := readability.FromReader(bytes.NewReader(body), parsed)
	if err != nil {
		return "", fmt.Errorf("readability: %w", err)
	}

	text := strings.TrimSpace(article.TextContent)
	if text == "" {
		text = strings.TrimSpace(article.Content)
	}
	return text, nil
}

func DescribeImage(ctx context.Context, path string, cfg Config) (string, error) {
	return "", nil // TODO
}

func TranscribeAudio(ctx context.Context, path string, cfg Config) (string, error) {
	return "", nil // TODO
}
