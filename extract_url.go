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

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/go-shiori/go-readability"
	"golang.org/x/net/html"
)

var htmlClient = &http.Client{Timeout: 30 * time.Second}

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
	case strings.HasSuffix(u.Path, ".rss"),
		strings.HasSuffix(u.Path, ".xml") && isRSSURL(u):
		return extractRSS(ctx, rawURL)
	default:
		return extractHTMLPage(ctx, rawURL)
	}
}

func isRSSURL(u *url.URL) bool {
	return strings.Contains(u.Path, "feed") || strings.Contains(u.Path, "rss")
}

func fetchHTML(ctx context.Context, rawURL string) ([]byte, *url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; caveman-mcp/0.2)")
	resp, err := htmlClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	parsed, _ := url.Parse(rawURL)
	return body, parsed, nil
}

func extractHTMLPage(ctx context.Context, rawURL string) (string, error) {
	// HEAD check for RSS/Atom feeds
	headReq, _ := http.NewRequestWithContext(ctx, "HEAD", rawURL, nil)
	if headResp, err := htmlClient.Do(headReq); err == nil {
		ct := headResp.Header.Get("Content-Type")
		headResp.Body.Close()
		if strings.Contains(ct, "application/rss+xml") || strings.Contains(ct, "application/atom+xml") {
			return extractRSS(ctx, rawURL)
		}
	}

	body, parsed, err := fetchHTML(ctx, rawURL)
	if err != nil {
		return "", err
	}
	article, err := readability.FromReader(bytes.NewReader(body), parsed)
	if err != nil {
		return "", fmt.Errorf("readability: %w", err)
	}
	// Use structured HTML content, not flat TextContent
	src := article.Content
	if src == "" {
		src = string(body)
	}
	// Strip noisy elements before markdown conversion
	src = stripHTMLElements(src, "nav", "footer", "aside", "script", "style")
	md, err := htmltomarkdown.ConvertString(src)
	if err != nil {
		return "", fmt.Errorf("html-to-markdown: %w", err)
	}
	return strings.TrimSpace(md), nil
}

// stripHTMLElements removes specific tags from HTML string.
func stripHTMLElements(htmlStr string, tags ...string) string {
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return htmlStr
	}
	var strip func(*html.Node)
	strip = func(n *html.Node) {
		var toRemove []*html.Node
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && tagSet[c.Data] {
				toRemove = append(toRemove, c)
			} else {
				strip(c)
			}
		}
		for _, c := range toRemove {
			n.RemoveChild(c)
		}
	}
	strip(doc)
	var buf strings.Builder
	for c := doc.FirstChild; c != nil; c = c.NextSibling {
		html.Render(&buf, c)
	}
	return buf.String()
}
