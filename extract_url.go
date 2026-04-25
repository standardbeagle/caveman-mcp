package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// ExtractURL routes to specialized extractor based on URL pattern.
// Replaces the function previously in extract.go.
func ExtractURL(ctx context.Context, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}
	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")

	switch {
	case host == "youtube.com" && strings.HasPrefix(u.Path, "/watch"),
		host == "youtu.be":
		return extractYouTube(ctx, rawURL)
	case host == "github.com":
		return extractGitHub(ctx, u)
	case host == "arxiv.org":
		return extractArXiv(ctx, rawURL)
	case host == "news.ycombinator.com":
		return extractHN(ctx, u.Query().Get("id"))
	case host == "reddit.com":
		return extractReddit(ctx, rawURL)
	default:
		return extractHTMLPage(ctx, rawURL)
	}
}
