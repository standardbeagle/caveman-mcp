# caveman-mcp v2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expand caveman-mcp from 3 generic tools to 4 smart-routing tools covering 15+ source types with differential Wenyan compression.

**Architecture:** New tools `condense_git` and `condense_log` handle developer artifacts. Existing `condense_url` gains URL-pattern routing (YouTube, GitHub, arXiv, HN, Reddit, RSS, enhanced HTML). Existing `condense_file` gains image (vision LLM), audio (Whisper), PPTX, and XLSX/CSV support. All new extractors follow the pattern: extract text → mechanical pass → LLM Wenyan pass.

**Tech Stack:** Go 1.25, `github.com/modelcontextprotocol/go-sdk v1.5.0`, `github.com/JohannesKaufmann/html-to-markdown/v2 v2.5.0`, `github.com/xuri/excelize/v2 v2.10.1`, OpenAI-compatible HTTP (no SDK), stdlib `archive/zip`, `encoding/xml`.

**Spec:** `docs/superpowers/specs/2026-04-25-caveman-mcp-v2-design.md`

---

## File Map

| File | Status | Responsibility |
|------|--------|---------------|
| `main.go` | **Modify** | Add `CondenseGitArgs`, `CondenseLogArgs` structs; register `condense_git`, `condense_log` tools; update `condense_file` handler for image/audio |
| `compress.go` | **Unchanged** | Mechanical pass + LLM Wenyan — do not touch |
| `extract.go` | **Modify** | Add PPTX, XLSX, CSV routing; move `ExtractURL` call to `extract_url.go` |
| `extract_url.go` | **Create** | URL router; enhanced HTML pipeline (readability → html-to-markdown → signal annotation) |
| `extract_sources.go` | **Create** | YouTube, GitHub repo+PR, arXiv, HN, Reddit, RSS extractors; vision LLM; Whisper audio |
| `extract_git.go` | **Create** | Unified diff parser; git log parser; git blame parser; GitHub PR fetcher |
| `extract_log.go` | **Create** | Stack trace parser (Go/Python/JS/Java/Rust); error deduplication |
| `extract_test.go` | **Create** | All extractor unit tests (httptest mocks for HTTP sources) |

---

## Task 1: Scaffold new files + verify deps

**Files:** `extract_url.go` (create), `extract_sources.go` (create), `extract_git.go` (create), `extract_log.go` (create), `extract_test.go` (create)

- [ ] **Step 1: Verify new deps build**

```bash
go build ./...
```
Expected: no errors. Deps `html-to-markdown/v2` and `excelize/v2` already added via `go get`.

- [ ] **Step 2: Create extract_url.go stub**

```go
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
```

- [ ] **Step 3: Create extract_sources.go stub**

```go
package main

import "context"

func extractYouTube(ctx context.Context, rawURL string) (string, error) {
	return "", nil // TODO
}

func extractGitHub(ctx context.Context, u interface{ Path() string }) (string, error) {
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
```

Wait — `u interface{ Path() string }` is wrong; pass `*url.URL`. Correct stub:

```go
package main

import (
	"context"
	"net/url"
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

func DescribeImage(ctx context.Context, path string, cfg Config) (string, error) {
	return "", nil // TODO
}

func TranscribeAudio(ctx context.Context, path string, cfg Config) (string, error) {
	return "", nil // TODO
}
```

- [ ] **Step 4: Create extract_git.go stub**

```go
package main

import "context"

func ParseGitDiff(text string) (string, error)  { return "", nil }
func ParseGitLog(text string) (string, error)   { return "", nil }
func ParseGitBlame(text string) (string, error) { return "", nil }
func FetchGitHubPR(ctx context.Context, prURL string) (string, error) { return "", nil }
```

- [ ] **Step 5: Create extract_log.go stub**

```go
package main

func ParseLog(text string) string { return "" }
```

- [ ] **Step 6: Create extract_test.go with a placeholder**

```go
package main

import "testing"

func TestPlaceholder(t *testing.T) {}
```

- [ ] **Step 7: Remove old ExtractURL from extract.go**

Delete the `ExtractURL` function from `extract.go` (it now lives in `extract_url.go`). Also remove unused imports it required (`bytes`, `net/url`, `net/http`, `time` — check if still used by remaining functions before removing).

- [ ] **Step 8: Build to confirm no compile errors**

```bash
go build ./...
```
Expected: compiles cleanly.

- [ ] **Step 9: Commit scaffold**

```bash
git init && git add -A
git commit -m "feat: scaffold v2 file structure with stubs"
```

---

## Task 2: Enhanced HTML pipeline

**Files:** `extract_url.go` (implement `extractHTMLPage`), `extract_test.go`

- [ ] **Step 1: Write failing test**

In `extract_test.go`:

```go
package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractHTMLPipeline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><body>
			<nav><a href="/">Home</a><a href="/about">About</a></nav>
			<article>
				<h1>Getting Started</h1>
				<h2>Installation</h2>
				<p>Run this command to install the package on your system.</p>
				<pre><code>npm install my-package</code></pre>
				<aside>Advertisement: Buy our pro plan today!</aside>
				<h3>Configuration</h3>
				<p>Set the API_KEY environment variable before starting.</p>
			</article>
			<footer>Copyright 2026. All rights reserved.</footer>
		</body></html>`))
	}))
	defer srv.Close()

	text, err := ExtractURL(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	// Structure preserved as markdown
	if !strings.Contains(text, "# Getting Started") {
		t.Errorf("h1 not in output; got:\n%s", text)
	}
	if !strings.Contains(text, "## Installation") {
		t.Errorf("h2 not in output; got:\n%s", text)
	}
	// Code block preserved
	if !strings.Contains(text, "npm install my-package") {
		t.Errorf("code not in output; got:\n%s", text)
	}
	// Nav/footer stripped
	if strings.Contains(text, "Copyright 2026") {
		t.Errorf("footer text leaked into output; got:\n%s", text)
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
go test -v -run TestExtractHTMLPipeline
```
Expected: FAIL (extractHTMLPage not implemented, returns "").

- [ ] **Step 3: Implement extractHTMLPage in extract_url.go**

```go
import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown/v2/htmltomarkdown"
	"github.com/go-shiori/go-readability"
	"golang.org/x/net/html"
)

var htmlClient = &http.Client{Timeout: 30 * time.Second}

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

// stripHTMLElements removes specific tags from HTML string using the html package.
func stripHTMLElements(htmlStr string, tags ...string) string {
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return htmlStr // parse failed, return as-is
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
```

- [ ] **Step 4: Run test to verify PASS**

```bash
go test -v -run TestExtractHTMLPipeline
```
Expected: PASS.

- [ ] **Step 5: Run all existing tests to check no regressions**

```bash
go test -v ./...
```

- [ ] **Step 6: Commit**

```bash
git add extract_url.go extract_test.go
git commit -m "feat: enhanced HTML pipeline with html-to-markdown and noise stripping"
```

---

## Task 3: YouTube extractor

**Files:** `extract_sources.go`, `extract_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestExtractYouTubeVideoID(t *testing.T) {
	cases := []struct {
		url string
		id  string
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=42s", "dQw4w9WgXcQ"},
		{"https://www.youtube.com/embed/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
	}
	for _, c := range cases {
		id, err := youtubeVideoID(c.url)
		if err != nil {
			t.Errorf("url %s: %v", c.url, err)
			continue
		}
		if id != c.id {
			t.Errorf("url %s: got %q, want %q", c.url, id, c.id)
		}
	}
}

func TestParseVTT(t *testing.T) {
	vtt := `WEBVTT

00:00:00.000 --> 00:00:03.000
Hello world this is a test

00:00:03.000 --> 00:00:06.000
<c.en>Second line of</c.en> the transcript

00:00:06.000 --> 00:00:09.000
Third line here`

	text := parseVTT(vtt)
	if !strings.Contains(text, "Hello world") {
		t.Errorf("missing first line; got: %q", text)
	}
	if !strings.Contains(text, "Third line") {
		t.Errorf("missing third line; got: %q", text)
	}
	if strings.Contains(text, "00:00") {
		t.Errorf("timestamps not stripped; got: %q", text)
	}
	if strings.Contains(text, "<c.") {
		t.Errorf("VTT tags not stripped; got: %q", text)
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
go test -v -run "TestExtractYouTube|TestParseVTT"
```

- [ ] **Step 3: Implement in extract_sources.go**

```go
import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
)

var (
	vttTimestampRe = regexp.MustCompile(`\d{2}:\d{2}:\d{2}\.\d{3} --> \d{2}:\d{2}:\d{2}\.\d{3}.*`)
	vttTagRe       = regexp.MustCompile(`<[^>]+>`)
	captionTracksRe = regexp.MustCompile(`"captionTracks":(\[.*?\])`)
	captionBaseURLRe = regexp.MustCompile(`"baseUrl":"(https://[^"]+)"`)
)

func youtubeVideoID(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	switch host {
	case "youtube.com":
		if id := u.Query().Get("v"); id != "" {
			return id, nil
		}
		// /embed/{id}
		parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
		if len(parts) == 2 && parts[0] == "embed" {
			return parts[1], nil
		}
	case "youtu.be":
		return strings.TrimPrefix(u.Path, "/"), nil
	}
	return "", fmt.Errorf("cannot extract video ID from %s", rawURL)
}

func parseVTT(vtt string) string {
	var sb strings.Builder
	for _, line := range strings.Split(vtt, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "WEBVTT" || vttTimestampRe.MatchString(line) {
			continue
		}
		line = vttTagRe.ReplaceAllString(line, "")
		if line != "" {
			sb.WriteString(line)
			sb.WriteRune('\n')
		}
	}
	return strings.TrimSpace(sb.String())
}

func extractYouTube(ctx context.Context, rawURL string) (string, error) {
	videoID, err := youtubeVideoID(rawURL)
	if err != nil {
		return "", err
	}

	// Step 1: fetch watch page to get signed caption URL
	body, _, err := fetchHTML(ctx, rawURL)
	if err != nil {
		return "", fmt.Errorf("fetch youtube page: %w", err)
	}

	// Extract og:title for prepend
	title := extractOGTitle(string(body))

	// Find captionTracks in ytInitialPlayerResponse
	m := captionTracksRe.FindSubmatch(body)
	if m == nil || len(m[1]) == 0 {
		return ytdlpFallback(rawURL, title)
	}

	// Extract first English baseUrl
	baseURLMatch := captionBaseURLRe.FindSubmatch(m[1])
	if baseURLMatch == nil {
		return ytdlpFallback(rawURL, title)
	}
	captionURL := string(baseURLMatch[1]) + "&fmt=vtt"
	captionURL = strings.ReplaceAll(captionURL, `&`, "&")

	// Fetch VTT
	req, _ := http.NewRequestWithContext(ctx, "GET", captionURL, nil)
	resp, err := htmlClient.Do(req)
	if err != nil {
		return ytdlpFallback(rawURL, title)
	}
	defer resp.Body.Close()
	vttBytes, err := io.ReadAll(resp.Body)
	if err != nil || len(strings.TrimSpace(string(vttBytes))) == 0 {
		return ytdlpFallback(rawURL, title)
	}

	transcript := parseVTT(string(vttBytes))
	if title != "" {
		return "Title: " + title + "\n\n" + transcript, nil
	}
	return transcript, nil
	_ = videoID // used for fallback context only
}

var ogTitleRe = regexp.MustCompile(`<meta property="og:title" content="([^"]+)"`)

func extractOGTitle(html string) string {
	m := ogTitleRe.FindStringSubmatch(html)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func ytdlpFallback(rawURL, title string) (string, error) {
	ytdlp, err := exec.LookPath("yt-dlp")
	if err != nil {
		return "", fmt.Errorf("no captions available and yt-dlp not installed; install with: pip install yt-dlp")
	}
	// Write to temp, read back
	tmp, _ := os.MkdirTemp("", "caveman-yt-*")
	defer os.RemoveAll(tmp)
	cmd := exec.Command(ytdlp, "--write-auto-sub", "--skip-download", "--sub-format", "vtt",
		"--sub-lang", "en", "-o", filepath.Join(tmp, "%(id)s.%(ext)s"), rawURL)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("yt-dlp failed: %w\n%s", err, out)
	}
	files, _ := filepath.Glob(filepath.Join(tmp, "*.vtt"))
	if len(files) == 0 {
		return "", fmt.Errorf("yt-dlp produced no VTT file for %s", rawURL)
	}
	vttBytes, err := os.ReadFile(files[0])
	if err != nil {
		return "", err
	}
	transcript := parseVTT(string(vttBytes))
	if title != "" {
		return "Title: " + title + "\n\n" + transcript, nil
	}
	return transcript, nil
}
```

Add `"os"` and `"path/filepath"` to imports.

- [ ] **Step 4: Run tests**

```bash
go test -v -run "TestExtractYouTube|TestParseVTT"
```
Expected: PASS.

- [ ] **Step 5: Build check**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add extract_sources.go extract_test.go
git commit -m "feat: YouTube transcript extractor with VTT parsing and yt-dlp fallback"
```

---

## Task 4: GitHub repo + PR extractors

**Files:** `extract_sources.go`, `extract_git.go`, `extract_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestExtractGitHubRouting(t *testing.T) {
	// Test that github.com URL routes correctly based on path structure
	cases := []struct {
		path    string
		wantPR  bool
		wantRepo bool
	}{
		{"/owner/repo", false, true},
		{"/owner/repo/", false, true},
		{"/owner/repo/pull/42", true, false},
		{"/owner/repo/tree/main", false, true}, // non-PR subpath → repo
	}
	for _, c := range cases {
		u, _ := url.Parse("https://github.com" + c.path)
		isPR := isGitHubPR(u)
		if isPR != c.wantPR {
			t.Errorf("path %s: isPR=%v, want %v", c.path, isPR, c.wantPR)
		}
	}
}

func TestGitHubAPIHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github.v3+json" {
			t.Errorf("missing Accept header; got %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"testrepo","description":"A test repo","language":"Go","stargazers_count":42}`))
	}))
	defer srv.Close()

	// Override base URL for testing
	oldGHBase := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldGHBase }()

	_, err := fetchGitHubRepoMeta(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
go test -v -run "TestExtractGitHub|TestGitHubAPI"
```

- [ ] **Step 3: Implement in extract_sources.go**

```go
var githubAPIBase = "https://api.github.com"

func isGitHubPR(u *url.URL) bool {
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	return len(parts) == 4 && parts[2] == "pull"
}

type ghRepoMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Stars       int    `json:"stargazers_count"`
	Topics      []string `json:"topics"`
}

func fetchGitHubRepoMeta(ctx context.Context, owner, repo string) (*ghRepoMeta, error) {
	return ghAPIGet[ghRepoMeta](ctx, fmt.Sprintf("/repos/%s/%s", owner, repo))
}

func ghAPIGet[T any](ctx context.Context, path string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", githubAPIBase+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "caveman-mcp/0.2")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("GitHub API rate limit exceeded (set GITHUB_TOKEN for higher limits)")
	}
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("GitHub resource not found: %s%s", githubAPIBase, path)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GitHub API error %d for %s", resp.StatusCode, path)
	}
	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode GitHub response: %w", err)
	}
	return &result, nil
}

func extractGitHub(ctx context.Context, u *url.URL) (string, error) {
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid GitHub URL: %s", u.String())
	}
	owner, repo := parts[0], parts[1]

	if isGitHubPR(u) {
		prNum := parts[3]
		return extractGitHubPR(ctx, owner, repo, prNum)
	}
	return extractGitHubRepo(ctx, owner, repo)
}

func extractGitHubRepo(ctx context.Context, owner, repo string) (string, error) {
	meta, err := fetchGitHubRepoMeta(ctx, owner, repo)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s/%s\n\n", owner, repo))
	if meta.Description != "" {
		sb.WriteString(meta.Description + "\n\n")
	}
	sb.WriteString(fmt.Sprintf("Language: %s | Stars: %d\n\n", meta.Language, meta.Stars))

	// README
	for _, branch := range []string{"main", "master"} {
		readmeURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/README.md", owner, repo, branch)
		req, _ := http.NewRequestWithContext(ctx, "GET", readmeURL, nil)
		resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
		if err == nil && resp.StatusCode == 200 {
			readmeBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			sb.WriteString("## README\n\n")
			sb.Write(readmeBytes)
			sb.WriteRune('\n')
			break
		}
	}

	// File tree
	type treeResp struct {
		Tree      []struct{ Path string `json:"path"` } `json:"tree"`
		Truncated bool `json:"truncated"`
	}
	tree, err := ghAPIGet[treeResp](ctx, fmt.Sprintf("/repos/%s/%s/git/trees/HEAD?recursive=1", owner, repo))
	if err == nil {
		if tree.Truncated {
			sb.WriteString("\n[Note: file tree truncated — repo has >100k objects]\n")
		}
		sb.WriteString("\n## Files (top 50)\n\n")
		count := 0
		for _, f := range tree.Tree {
			if count >= 50 {
				break
			}
			sb.WriteString("- " + f.Path + "\n")
			count++
		}
	}

	return sb.String(), nil
}

func extractGitHubPR(ctx context.Context, owner, repo, prNum string) (string, error) {
	type prMeta struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		State string `json:"state"`
		User  struct{ Login string `json:"login"` } `json:"user"`
		Head  struct{ SHA string `json:"sha"` } `json:"head"`
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
		ChangedFiles int `json:"changed_files"`
	}
	pr, err := ghAPIGet[prMeta](ctx, fmt.Sprintf("/repos/%s/%s/pulls/%s", owner, repo, prNum))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# PR #%s: %s\n\n", prNum, pr.Title))
	sb.WriteString(fmt.Sprintf("Author: %s | State: %s | +%d -%d in %d files\n\n",
		pr.User.Login, pr.State, pr.Additions, pr.Deletions, pr.ChangedFiles))
	if pr.Body != "" {
		sb.WriteString("## Description\n\n" + pr.Body + "\n\n")
	}

	// Check for truncation via files endpoint
	type filesResp struct {
		Filename string `json:"filename"`
	}
	filesURL := fmt.Sprintf("/repos/%s/%s/pulls/%s/files?per_page=300", owner, repo, prNum)
	req, _ := http.NewRequestWithContext(ctx, "GET", githubAPIBase+filesURL, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err == nil {
		defer resp.Body.Close()
		// Check Link header for rel="next" (truncation indicator)
		if link := resp.Header.Get("Link"); strings.Contains(link, `rel="next"`) {
			sb.WriteString("[Warning: PR has >300 changed files; diff may be truncated]\n\n")
		}
	}

	// Fetch raw diff
	diffReq, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%s", owner, repo, prNum), nil)
	diffReq.Header.Set("Accept", "application/vnd.github.diff")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		diffReq.Header.Set("Authorization", "Bearer "+tok)
	}
	diffResp, err := (&http.Client{Timeout: 30 * time.Second}).Do(diffReq)
	if err == nil {
		defer diffResp.Body.Close()
		diffBytes, _ := io.ReadAll(diffResp.Body)
		if len(diffBytes) > 0 {
			diffSummary, _ := ParseGitDiff(string(diffBytes))
			sb.WriteString("## Diff Summary\n\n" + diffSummary + "\n")
		}
	}

	return sb.String(), nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test -v -run "TestExtractGitHub|TestGitHubAPI"
```

- [ ] **Step 5: Build**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add extract_sources.go extract_test.go
git commit -m "feat: GitHub repo and PR extractors"
```

---

## Task 5: arXiv + HN + Reddit + RSS extractors

**Files:** `extract_sources.go`, `extract_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestParseHNComments(t *testing.T) {
	// HN story response with 3 kids
	storyJSON := `{"id":12345,"title":"Test Post","kids":[11,22,33],"type":"story","url":"https://example.com"}`
	comment1 := `{"id":11,"by":"alice","text":"First comment here","kids":[44],"type":"comment"}`
	comment2 := `{"id":22,"by":"bob","text":"Second comment","type":"comment"}`
	comment3 := `{"id":33,"by":"carol","text":"Third comment","type":"comment"}`
	reply    := `{"id":44,"by":"dave","text":"Reply to alice","type":"comment"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/12345"):
			w.Write([]byte(storyJSON))
		case strings.Contains(r.URL.Path, "/11"):
			w.Write([]byte(comment1))
		case strings.Contains(r.URL.Path, "/22"):
			w.Write([]byte(comment2))
		case strings.Contains(r.URL.Path, "/33"):
			w.Write([]byte(comment3))
		case strings.Contains(r.URL.Path, "/44"):
			w.Write([]byte(reply))
		}
	}))
	defer srv.Close()

	oldBase := hnAPIBase
	hnAPIBase = srv.URL
	defer func() { hnAPIBase = oldBase }()

	text, err := extractHN(context.Background(), "12345")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "alice") {
		t.Errorf("missing comment author; got:\n%s", text)
	}
	if !strings.Contains(text, "First comment here") {
		t.Errorf("missing comment text; got:\n%s", text)
	}
}

func TestParseRSS(t *testing.T) {
	rssFeed := `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>Article One</title>
      <description>Description of article one with details.</description>
    </item>
    <item>
      <title>Article Two</title>
      <description>Description of article two.</description>
    </item>
  </channel>
</rss>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(rssFeed))
	}))
	defer srv.Close()

	text, err := extractRSS(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Article One") {
		t.Errorf("missing item title; got:\n%s", text)
	}
	if !strings.Contains(text, "Description of article one") {
		t.Errorf("missing item description; got:\n%s", text)
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
go test -v -run "TestParseHN|TestParseRSS|TestParseReddit"
```

- [ ] **Step 3: Implement arXiv in extract_sources.go**

```go
func extractArXiv(ctx context.Context, rawURL string) (string, error) {
	// Normalize: arxiv.org/abs/{id} or arxiv.org/pdf/{id}
	u, _ := url.Parse(rawURL)
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("cannot parse arXiv ID from %s", rawURL)
	}
	paperID := parts[len(parts)-1]
	paperID = strings.TrimSuffix(paperID, ".pdf")

	absURL := "https://export.arxiv.org/abs/" + paperID
	body, _, err := fetchHTML(ctx, absURL)
	if err != nil {
		return "", fmt.Errorf("fetch arXiv: %w", err)
	}

	text := extractHTMLText(string(body))
	return text, nil
}

// extractHTMLText strips all HTML tags and returns plain text.
func extractHTMLText(htmlStr string) string {
	var sb strings.Builder
	dec := html.NewTokenizer(strings.NewReader(htmlStr))
	for {
		tt := dec.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.TextToken {
			sb.Write(dec.Text())
		}
	}
	return strings.TrimSpace(sb.String())
}
```

- [ ] **Step 4: Implement HN in extract_sources.go**

```go
var hnAPIBase = "https://hacker-news.firebaseio.com/v0"

type hnItem struct {
	ID    int    `json:"id"`
	By    string `json:"by"`
	Text  string `json:"text"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Kids  []int  `json:"kids"`
	Type  string `json:"type"`
}

func hnGetItem(ctx context.Context, id int) (*hnItem, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/item/%d.json", hnAPIBase, id), nil)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var item hnItem
	return &item, json.NewDecoder(resp.Body).Decode(&item)
}

func extractHN(ctx context.Context, id string) (string, error) {
	var itemID int
	fmt.Sscanf(id, "%d", &itemID)
	if itemID == 0 {
		return "", fmt.Errorf("invalid HN item ID: %q", id)
	}
	story, err := hnGetItem(ctx, itemID)
	if err != nil {
		return "", fmt.Errorf("fetch HN story: %w", err)
	}

	var sb strings.Builder
	if story.Title != "" {
		sb.WriteString("# " + story.Title + "\n\n")
	}
	if story.URL != "" {
		sb.WriteString("URL: " + story.URL + "\n\n")
	}
	sb.WriteString("## Comments\n\n")

	// Top 20 from kids array (HN-ranked order)
	limit := 20
	if len(story.Kids) < limit {
		limit = len(story.Kids)
	}

	// Fetch comments in parallel
	type result struct {
		idx  int
		item *hnItem
		err  error
	}
	results := make([]result, limit)
	ch := make(chan result, limit)
	for i, kid := range story.Kids[:limit] {
		go func(idx, kid int) {
			item, err := hnGetItem(ctx, kid)
			ch <- result{idx, item, err}
		}(i, kid)
	}
	for range limit {
		r := <-ch
		results[r.idx] = r
	}

	for _, r := range results {
		if r.err != nil || r.item == nil {
			continue
		}
		text := html.UnescapeString(vttTagRe.ReplaceAllString(r.item.Text, ""))
		sb.WriteString(fmt.Sprintf("**%s:** %s\n\n", r.item.By, text))
		// Top 2 replies
		replyLimit := 2
		if len(r.item.Kids) < replyLimit {
			replyLimit = len(r.item.Kids)
		}
		for _, replyID := range r.item.Kids[:replyLimit] {
			reply, err := hnGetItem(ctx, replyID)
			if err == nil && reply != nil {
				replyText := html.UnescapeString(vttTagRe.ReplaceAllString(reply.Text, ""))
				sb.WriteString(fmt.Sprintf("  > **%s:** %s\n\n", reply.By, replyText))
			}
		}
	}

	return sb.String(), nil
}
```

Import `"html"` (stdlib) for `html.UnescapeString`.

- [ ] **Step 5: Implement Reddit in extract_sources.go**

```go
func extractReddit(ctx context.Context, rawURL string) (string, error) {
	jsonURL := strings.TrimSuffix(rawURL, "/") + ".json?limit=100"
	req, _ := http.NewRequestWithContext(ctx, "GET", jsonURL, nil)
	req.Header.Set("User-Agent", "script:caveman-mcp:v0.2 (by /u/caveman-mcp-bot)")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch reddit: %w", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		return "", fmt.Errorf("Reddit returned non-JSON (Cloudflare block or subreddit restriction); Content-Type: %s", ct)
	}

	var raw []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil || len(raw) < 2 {
		return "", fmt.Errorf("parse reddit response: %w", err)
	}

	// First element is post, second is comments
	type redditChild struct {
		Kind string `json:"kind"`
		Data struct {
			Title    string  `json:"title"`
			Selftext string  `json:"selftext"`
			Author   string  `json:"author"`
			Score    int     `json:"score"`
			Body     string  `json:"body"`
			Children []struct {
				Kind string `json:"kind"`
				Data struct {
					Author   string  `json:"author"`
					Body     string  `json:"body"`
					Score    int     `json:"score"`
					Replies  json.RawMessage `json:"replies"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	type listing struct {
		Data struct {
			Children []redditChild `json:"children"`
		} `json:"data"`
	}

	var post, comments listing
	json.Unmarshal(raw[0], &post)
	json.Unmarshal(raw[1], &comments)

	var sb strings.Builder
	if len(post.Data.Children) > 0 {
		p := post.Data.Children[0].Data
		sb.WriteString("# " + p.Title + "\n\n")
		if p.Selftext != "" {
			sb.WriteString(p.Selftext + "\n\n")
		}
	}
	sb.WriteString("## Comments\n\n")

	// Sort by score
	type scoredComment struct {
		author string
		body   string
		score  int
	}
	var scored []scoredComment
	for _, c := range comments.Data.Children {
		if c.Kind != "t1" {
			continue
		}
		scored = append(scored, scoredComment{c.Data.Author, c.Data.Body, c.Data.Score})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
	limit := 20
	if len(scored) < limit {
		limit = len(scored)
	}
	for _, c := range scored[:limit] {
		sb.WriteString(fmt.Sprintf("[%d] u/%s: %s\n\n", c.Score, c.Author, c.Body))
	}

	return sb.String(), nil
}
```

Add `"sort"` to imports.

- [ ] **Step 6: Implement RSS in extract_sources.go**

```go
func extractRSS(ctx context.Context, rawURL string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	req.Header.Set("User-Agent", "caveman-mcp/0.2")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch RSS: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	type item struct {
		Title       string `xml:"title"`
		Description string `xml:"description"`
		Content     string `xml:"encoded"`
	}
	type channel struct {
		Title string `xml:"title"`
		Items []item `xml:"item"`
		Entries []struct {
			Title   string `xml:"title"`
			Summary string `xml:"summary"`
			Content string `xml:"content"`
		} `xml:"entry"`
	}
	type feed struct {
		Channel channel `xml:"channel"`
		// Atom top-level
		Title   string `xml:"title"`
		Entries []struct {
			Title   string `xml:"title"`
			Summary string `xml:"summary"`
		} `xml:"entry"`
	}

	var f feed
	if err := xml.Unmarshal(body, &f); err != nil {
		return "", fmt.Errorf("parse RSS/Atom: %w", err)
	}

	var sb strings.Builder
	title := f.Channel.Title
	if title == "" {
		title = f.Title
	}
	if title != "" {
		sb.WriteString("# " + title + "\n\n")
	}

	// RSS items
	for _, item := range f.Channel.Items {
		sb.WriteString("## " + item.Title + "\n\n")
		content := item.Content
		if content == "" {
			content = item.Description
		}
		sb.WriteString(content + "\n\n")
	}

	// Atom entries
	for _, e := range f.Entries {
		sb.WriteString("## " + e.Title + "\n\n")
		sb.WriteString(e.Summary + "\n\n")
	}

	return strings.TrimSpace(sb.String()), nil
}
```

Add `"encoding/xml"` to imports.

Also update `extract_url.go` to route RSS by content-type and detect `.rss`/`.xml`:

```go
// In ExtractURL, before the switch, check for RSS content-type:
// (Add this after fetchHTML call or inline in the default case)
// Actually check URL suffix first:
case strings.HasSuffix(u.Path, ".rss"),
	strings.HasSuffix(u.Path, ".xml") && isRSSURL(u):
	return extractRSS(ctx, rawURL)
```

And add helper in `extract_url.go`:
```go
func isRSSURL(u *url.URL) bool {
	return strings.Contains(u.Path, "feed") || strings.Contains(u.Path, "rss")
}
```

For content-type-based RSS detection, do an initial HEAD request in `extractHTMLPage` and redirect to `extractRSS` if the content type is `application/rss+xml` or `application/atom+xml`.

- [ ] **Step 7: Run all tests**

```bash
go test -v -run "TestParseHN|TestParseRSS|TestExtractArXiv"
```

- [ ] **Step 8: Build**

```bash
go build ./...
```

- [ ] **Step 9: Commit**

```bash
git add extract_sources.go extract_url.go extract_test.go
git commit -m "feat: arXiv, HN, Reddit, RSS extractors"
```

---

## Task 6: Vision LLM (image support)

**Files:** `extract_sources.go`, `extract.go`, `main.go`, `extract_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestDescribeImageMock(t *testing.T) {
	// Mock LLM server that validates vision request format
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Messages) == 0 {
			t.Error("no messages in vision request")
		}
		// Content must be an array (not a string)
		var parts []interface{}
		if err := json.Unmarshal(req.Messages[0].Content, &parts); err != nil {
			t.Errorf("vision content is not array: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"圖示系統架構三層：前端、後端、資料庫"}}]}`))
	}))
	defer srv.Close()

	cfg := Config{BaseURL: srv.URL, APIKey: "test", Model: "test-vision"}
	
	// Create a tiny 1x1 PNG
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR length + type
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, // bit depth, color, crc
		0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, 0x54, // IDAT
		0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01,
		0xE2, 0x21, 0xBC, 0x33,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82, // IEND
	}
	tmp, _ := os.CreateTemp("", "test-*.png")
	tmp.Write(pngData)
	tmp.Close()
	defer os.Remove(tmp.Name())

	desc, err := DescribeImage(context.Background(), tmp.Name(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if desc == "" {
		t.Error("empty description")
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
go test -v -run TestDescribeImageMock
```

- [ ] **Step 3: Implement DescribeImage in extract_sources.go**

```go
import "encoding/base64"

type llmVisionMsg struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type llmVisionReq struct {
	Model    string         `json:"model"`
	Messages []llmVisionMsg `json:"messages"`
}

type visionContentPart struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *visionImgURL `json:"image_url,omitempty"`
}

type visionImgURL struct {
	URL string `json:"url"`
}

const visionPrompt = `Describe this image in ≤40 classical Chinese wenyan characters.
Diagrams/charts: extract structure and data values.
UI screenshots: list key components and layout.
Photos: scene + key subjects.
Preserve ALL text, numbers, identifiers exactly.`

func DescribeImage(ctx context.Context, path string, cfg Config) (string, error) {
	imgBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	mimeType := map[string]string{
		".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".gif": "image/gif", ".webp": "image/webp", ".bmp": "image/bmp",
	}[ext]
	if mimeType == "" {
		mimeType = "image/png"
	}

	b64 := base64.StdEncoding.EncodeToString(imgBytes)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)

	parts := []visionContentPart{
		{Type: "image_url", ImageURL: &visionImgURL{URL: dataURL}},
		{Type: "text", Text: visionPrompt},
	}
	contentJSON, _ := json.Marshal(parts)

	body, _ := json.Marshal(llmVisionReq{
		Model: cfg.Model,
		Messages: []llmVisionMsg{
			{Role: "user", Content: contentJSON},
		},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("vision LLM: %w", err)
	}
	defer resp.Body.Close()

	var r llmResp
	raw, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", fmt.Errorf("decode vision response: %w (body: %.200s)", err, raw)
	}
	if r.Error != nil {
		return "", fmt.Errorf("vision LLM error: %s", r.Error.Message)
	}
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("no choices in vision response")
	}
	return strings.TrimSpace(r.Choices[0].Message.Content), nil
}
```

- [ ] **Step 4: Update condense_file handler in main.go**

Add image extension detection and routing. In `main.go`, update the `condense_file` handler:

```go
var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".webp": true, ".bmp": true,
}

// In the condense_file handler, before ExtractFile:
ext := strings.ToLower(filepath.Ext(args.Path))
if imageExts[ext] {
    desc, err := DescribeImage(ctx, args.Path, comp.cfg)
    if err != nil {
        return nil, nil, fmt.Errorf("describe image %s: %w", args.Path, err)
    }
    r := &Result{
        Compressed:      desc,
        OriginalChars:   len(desc),
        CompressedChars: len(desc),
        Ratio:           1.0,
        Method:          "vision",
    }
    return resultContent(r), nil, nil
}
```

Add `"path/filepath"` and `"strings"` imports to `main.go`.

- [ ] **Step 5: Run tests**

```bash
go test -v -run TestDescribeImageMock
```
Expected: PASS.

- [ ] **Step 6: Build**

```bash
go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add extract_sources.go main.go extract_test.go
git commit -m "feat: image description via vision LLM"
```

---

## Task 7: Audio transcription (Whisper)

**Files:** `extract_sources.go`, `main.go`, `extract_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestTranscribeAudioSizeGuard(t *testing.T) {
	// Create a file just over 25MB
	tmp, _ := os.CreateTemp("", "big-*.mp3")
	tmp.Write(make([]byte, 26*1024*1024))
	tmp.Close()
	defer os.Remove(tmp.Name())

	cfg := Config{APIKey: "test", BaseURL: "http://localhost", Model: "test"}
	_, err := TranscribeAudio(context.Background(), tmp.Name(), cfg)
	if err == nil {
		t.Error("expected error for oversized file")
	}
	if !strings.Contains(err.Error(), "25MB") {
		t.Errorf("error should mention 25MB limit; got: %v", err)
	}
}

func TestTranscribeAudioMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || !strings.Contains(r.URL.Path, "transcriptions") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("expected multipart; got %s", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"text":"Hello this is a test transcript."}`))
	}))
	defer srv.Close()

	cfg := Config{BaseURL: srv.URL, APIKey: "test", Model: "whisper-1"}
	tmp, _ := os.CreateTemp("", "test-*.mp3")
	tmp.Write([]byte("fake mp3 data"))
	tmp.Close()
	defer os.Remove(tmp.Name())

	text, err := TranscribeAudio(context.Background(), tmp.Name(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "test transcript") {
		t.Errorf("unexpected transcript: %q", text)
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
go test -v -run TestTranscribeAudio
```

- [ ] **Step 3: Implement TranscribeAudio in extract_sources.go**

```go
import (
	"bytes"
	"mime/multipart"
	"path/filepath"
)

const maxAudioBytes = 25 * 1024 * 1024 // 25MB Whisper API limit

func TranscribeAudio(ctx context.Context, path string, cfg Config) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat audio file: %w", err)
	}
	if info.Size() > maxAudioBytes {
		return "", fmt.Errorf("audio file exceeds 25MB Whisper API limit: %.1fMB",
			float64(info.Size())/(1024*1024))
	}

	audioBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read audio: %w", err)
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", err
	}
	fw.Write(audioBytes)
	w.WriteField("model", "whisper-1")
	w.Close()

	apiKey := os.Getenv("WHISPER_API_KEY")
	if apiKey == "" {
		apiKey = cfg.APIKey
	}
	baseURL := os.Getenv("WHISPER_BASE_URL")
	if baseURL == "" {
		baseURL = cfg.BaseURL
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/audio/transcriptions", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("whisper API: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Text  string `json:"text"`
		Error *struct{ Message string `json:"message"` } `json:"error,omitempty"`
	}
	raw, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("decode whisper response: %w (body: %.200s)", err, raw)
	}
	if result.Error != nil {
		return "", fmt.Errorf("whisper error: %s", result.Error.Message)
	}
	return result.Text, nil
}
```

- [ ] **Step 4: Update condense_file handler in main.go** for audio routing

```go
var audioExts = map[string]bool{
	".mp3": true, ".wav": true, ".m4a": true, ".ogg": true, ".flac": true,
}

// In condense_file handler, after image check:
if audioExts[ext] {
    transcript, err := TranscribeAudio(ctx, args.Path, comp.cfg)
    if err != nil {
        return nil, nil, fmt.Errorf("transcribe audio %s: %w", args.Path, err)
    }
    r, err := comp.Condense(ctx, transcript, !args.SkipLLM)
    if err != nil {
        return nil, nil, err
    }
    r.Method = "whisper+" + r.Method
    return resultContent(r), nil, nil
}
```

- [ ] **Step 5: Run tests**

```bash
go test -v -run TestTranscribeAudio
```
Expected: both PASS.

- [ ] **Step 6: Build + commit**

```bash
go build ./...
git add extract_sources.go main.go extract_test.go
git commit -m "feat: Whisper audio transcription with 25MB guard"
```

---

## Task 8: PPTX + XLSX/CSV support

**Files:** `extract.go`, `extract_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestExtractPPTX(t *testing.T) {
	// PPTX is a zip. Create minimal one with one slide.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// slide1.xml with title and body text
	slideXML := `<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
       xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree>
    <p:sp><p:txBody><a:p><a:r><a:t>Slide Title Here</a:t></a:r></a:p></p:txBody></p:sp>
    <p:sp><p:txBody><a:p><a:r><a:t>Bullet point one</a:t></a:r></a:p></p:txBody></p:sp>
  </p:spTree></p:cSld>
</p:sld>`
	fw, _ := zw.Create("ppt/slides/slide1.xml")
	fw.Write([]byte(slideXML))

	// [Content_Types].xml (required by zip format)
	fw2, _ := zw.Create("[Content_Types].xml")
	fw2.Write([]byte(`<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"></Types>`))
	zw.Close()

	tmp, _ := os.CreateTemp("", "test-*.pptx")
	tmp.Write(buf.Bytes())
	tmp.Close()
	defer os.Remove(tmp.Name())

	text, err := ExtractFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Slide Title Here") {
		t.Errorf("missing slide title; got:\n%s", text)
	}
	if !strings.Contains(text, "Bullet point one") {
		t.Errorf("missing bullet; got:\n%s", text)
	}
}

func TestExtractCSV(t *testing.T) {
	csv := "name,age,city\nAlice,30,NYC\nBob,25,LA\nCarol,35,Chicago\n"
	tmp, _ := os.CreateTemp("", "test-*.csv")
	tmp.WriteString(csv)
	tmp.Close()
	defer os.Remove(tmp.Name())

	text, err := ExtractFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "name") {
		t.Errorf("missing column header; got:\n%s", text)
	}
	if !strings.Contains(text, "Alice") || !strings.Contains(text, "Bob") {
		t.Errorf("missing rows; got:\n%s", text)
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
go test -v -run "TestExtractPPTX|TestExtractCSV"
```

- [ ] **Step 3: Implement extractPPTX in extract.go**

```go
func extractPPTX(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open pptx: %w", err)
	}
	defer r.Close()

	// Collect slide files sorted by name
	var slideFiles []*zip.File
	var noteFiles []*zip.File
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			slideFiles = append(slideFiles, f)
		}
		if strings.HasPrefix(f.Name, "ppt/notesSlides/notesSlide") && strings.HasSuffix(f.Name, ".xml") {
			noteFiles = append(noteFiles, f)
		}
	}
	sort.Slice(slideFiles, func(i, j int) bool { return slideFiles[i].Name < slideFiles[j].Name })
	sort.Slice(noteFiles, func(i, j int) bool { return noteFiles[i].Name < noteFiles[j].Name })

	var sb strings.Builder
	for i, f := range slideFiles {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		text := extractXMLText(rc, "t")
		rc.Close()
		sb.WriteString(fmt.Sprintf("## Slide %d\n\n%s\n\n", i+1, text))

		// Speaker notes
		if i < len(noteFiles) {
			rc2, err := noteFiles[i].Open()
			if err == nil {
				notes := extractXMLText(rc2, "t")
				rc2.Close()
				if notes != "" {
					sb.WriteString(fmt.Sprintf("[Notes: %s]\n\n", notes))
				}
			}
		}
	}
	return strings.TrimSpace(sb.String()), nil
}

// extractXMLText extracts text from XML elements with given local name.
func extractXMLText(r io.Reader, localName string) string {
	var sb strings.Builder
	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == localName {
			var text string
			if err := dec.DecodeElement(&text, &se); err == nil && text != "" {
				sb.WriteString(text)
				sb.WriteRune(' ')
			}
		}
	}
	return strings.TrimSpace(sb.String())
}
```

- [ ] **Step 4: Implement CSV extractor in extract.go**

```go
import "encoding/csv"

func extractCSV(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return "", fmt.Errorf("parse csv: %w", err)
	}
	if len(records) == 0 {
		return "", nil
	}

	headers := records[0]
	rows := records[1:]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CSV: %d rows × %d columns\n\n", len(rows), len(headers)))
	sb.WriteString("Columns: " + strings.Join(headers, ", ") + "\n\n")

	// Show first 5 rows as sample
	limit := 5
	if len(rows) < limit {
		limit = len(rows)
	}
	sb.WriteString(fmt.Sprintf("Sample (%d of %d rows):\n", limit, len(rows)))
	for _, row := range rows[:limit] {
		sb.WriteString("  " + strings.Join(row, " | ") + "\n")
	}

	return sb.String(), nil
}
```

- [ ] **Step 5: Add XLSX extractor using excelize**

```go
import "github.com/xuri/excelize/v2"

func extractXLSX(path string) (string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("XLSX: %d sheet(s)\n\n", len(sheets)))

	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("## Sheet: %s (%d rows)\n\n", sheet, len(rows)-1))
		if len(rows) > 0 {
			sb.WriteString("Columns: " + strings.Join(rows[0], ", ") + "\n\n")
		}
		limit := 5
		if len(rows)-1 < limit {
			limit = len(rows) - 1
		}
		if limit > 0 {
			sb.WriteString(fmt.Sprintf("Sample (%d rows):\n", limit))
			for _, row := range rows[1 : 1+limit] {
				sb.WriteString("  " + strings.Join(row, " | ") + "\n")
			}
		}
		sb.WriteRune('\n')
	}
	return strings.TrimSpace(sb.String()), nil
}
```

- [ ] **Step 6: Add routing in ExtractFile switch in extract.go**

```go
case ".pptx":
    return extractPPTX(path)
case ".xlsx":
    return extractXLSX(path)
case ".csv":
    return extractCSV(path)
```

- [ ] **Step 7: Run tests**

```bash
go test -v -run "TestExtractPPTX|TestExtractCSV"
```
Expected: PASS.

- [ ] **Step 8: Build + commit**

```bash
go build ./...
git add extract.go extract_test.go
git commit -m "feat: PPTX, XLSX, CSV file extractors"
```

---

## Task 9: condense_git tool

**Files:** `extract_git.go`, `main.go`, `extract_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestParseGitDiff(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
index abc123..def456 100644
--- a/main.go
+++ b/main.go
@@ -10,7 +10,8 @@ func main() {
 	ctx := context.Background()
-	server := mcp.NewServer("old", nil)
+	server := mcp.NewServer(&mcp.Implementation{Name: "caveman-mcp"}, nil)
+	server.SetVersion("0.2.0")
 	server.Run(ctx, &mcp.StdioTransport{})
diff --git a/compress.go b/compress.go
index 111..222 100644
--- a/compress.go
+++ b/compress.go
@@ -5,3 +5,4 @@ package main
 import "strings"
+import "unicode"
`

	summary, err := ParseGitDiff(diff)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "main.go") {
		t.Errorf("missing file name; got:\n%s", summary)
	}
	if !strings.Contains(summary, "compress.go") {
		t.Errorf("missing second file; got:\n%s", summary)
	}
	// Should have stat line
	if !strings.Contains(summary, "+") || !strings.Contains(summary, "-") {
		t.Errorf("missing diff stats; got:\n%s", summary)
	}
}

func TestParseGitLog(t *testing.T) {
	log := `commit abc12345def67890
Author: Alice <alice@example.com>
Date:   Thu Apr 25 10:00:00 2026 -0700

    feat: add YouTube extractor

commit bcd23456efa78901
Author: Bob <bob@example.com>
Date:   Wed Apr 24 09:00:00 2026 -0700

    fix: handle empty caption tracks

commit cde34567fab89012
Author: Alice <alice@example.com>
Date:   Tue Apr 23 08:00:00 2026 -0700

    feat: add arXiv support
`

	summary, err := ParseGitLog(log)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "feat") {
		t.Errorf("missing feat group; got:\n%s", summary)
	}
	if !strings.Contains(summary, "fix") {
		t.Errorf("missing fix group; got:\n%s", summary)
	}
}

func TestCondenseGitValidation(t *testing.T) {
	// Both text and path provided → error
	args := CondenseGitArgs{Text: "some diff", Path: "/tmp/diff.txt"}
	_, err := resolveGitInput(context.Background(), args)
	if err == nil {
		t.Error("expected error for multiple inputs")
	}
	if !strings.Contains(err.Error(), "exactly one") {
		t.Errorf("wrong error: %v", err)
	}

	// No input → error
	args2 := CondenseGitArgs{}
	_, err2 := resolveGitInput(context.Background(), args2)
	if err2 == nil {
		t.Error("expected error for no input")
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
go test -v -run "TestParseGit|TestCondenseGit"
```

- [ ] **Step 3: Implement extract_git.go**

```go
package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// ParseGitDiff parses a unified diff and returns a per-file summary with stats.
func ParseGitDiff(text string) (string, error) {
	lines := strings.Split(text, "\n")
	type fileChange struct {
		path string
		adds int
		dels int
	}
	var files []fileChange
	var current *fileChange

	diffFileRe := regexp.MustCompile(`^diff --git a/(.+) b/(.+)`)
	addRe := regexp.MustCompile(`^\+[^+]`)
	delRe := regexp.MustCompile(`^-[^-]`)

	for _, line := range lines {
		if m := diffFileRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				files = append(files, *current)
			}
			current = &fileChange{path: m[2]}
			continue
		}
		if current == nil {
			continue
		}
		if addRe.MatchString(line) {
			current.adds++
		} else if delRe.MatchString(line) {
			current.dels++
		}
	}
	if current != nil {
		files = append(files, *current)
	}

	if len(files) == 0 {
		return text, nil // not a diff, return as-is
	}

	totalAdds, totalDels := 0, 0
	for _, f := range files {
		totalAdds += f.adds
		totalDels += f.dels
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[+%d -%d in %d files]\n\n", totalAdds, totalDels, len(files)))
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("%s: +%d -%d\n", f.path, f.adds, f.dels))
	}
	return sb.String(), nil
}

// ParseGitLog parses git log output and clusters by conventional commit prefix.
func ParseGitLog(text string) (string, error) {
	commitRe := regexp.MustCompile(`^commit [0-9a-f]+`)
	prefixRe := regexp.MustCompile(`^(feat|fix|chore|refactor|docs|test|perf|ci|build|style)`)

	groups := map[string][]string{}
	var other []string
	var currentSubject string

	for _, line := range strings.Split(text, "\n") {
		if commitRe.MatchString(line) || strings.HasPrefix(line, "Author:") || strings.HasPrefix(line, "Date:") {
			continue
		}
		subject := strings.TrimSpace(line)
		if subject == "" || currentSubject == subject {
			continue
		}
		currentSubject = subject
		if m := prefixRe.FindString(subject); m != "" {
			groups[m] = append(groups[m], subject)
		} else {
			other = append(other, subject)
		}
	}

	// Sort group keys for deterministic output
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		subjects := groups[k]
		sb.WriteString(fmt.Sprintf("%s (%d):\n", k, len(subjects)))
		for _, s := range subjects {
			sb.WriteString("  - " + s + "\n")
		}
		sb.WriteRune('\n')
	}
	if len(other) > 0 {
		sb.WriteString(fmt.Sprintf("other (%d):\n", len(other)))
		for _, s := range other {
			sb.WriteString("  - " + s + "\n")
		}
	}
	return strings.TrimSpace(sb.String()), nil
}

// ParseGitBlame parses git blame output.
func ParseGitBlame(text string) (string, error) {
	authors := map[string]int{}
	lineRe := regexp.MustCompile(`^\^?[0-9a-f]+ \(([^)]+)\s+\d{4}-\d{2}-\d{2}`)
	for _, line := range strings.Split(text, "\n") {
		if m := lineRe.FindStringSubmatch(line); m != nil {
			author := strings.TrimSpace(m[1])
			// author field may be "Name YYYY-MM-DD HH:MM:SS TZ N" — extract just name
			nameParts := strings.Fields(author)
			if len(nameParts) > 0 {
				authors[nameParts[0]]++
			}
		}
	}
	if len(authors) == 0 {
		return text, nil
	}
	type pair struct{ name string; count int }
	var pairs []pair
	for n, c := range authors {
		pairs = append(pairs, pair{n, c})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].count > pairs[j].count })

	var sb strings.Builder
	sb.WriteString("Blame by author:\n")
	for _, p := range pairs {
		sb.WriteString(fmt.Sprintf("  %s: %d lines\n", p.name, p.count))
	}
	return sb.String(), nil
}

// FetchGitHubPR fetches a GitHub PR URL and returns its diff+description.
func FetchGitHubPR(ctx context.Context, prURL string) (string, error) {
	u, err := url.Parse(prURL)
	if err != nil {
		return "", fmt.Errorf("parse PR URL: %w", err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 4 || parts[2] != "pull" {
		return "", fmt.Errorf("not a GitHub PR URL: %s", prURL)
	}
	return extractGitHubPR(ctx, parts[0], parts[1], parts[3])
}

// resolveGitInput validates args and returns the raw git text.
func resolveGitInput(ctx context.Context, args CondenseGitArgs) (string, error) {
	count := 0
	if args.Text != "" { count++ }
	if args.Path != "" { count++ }
	if args.PRUrl != "" { count++ }
	if count != 1 {
		return "", fmt.Errorf("condense_git requires exactly one of: text, path, pr_url (got %d)", count)
	}
	if args.Text != "" {
		return args.Text, nil
	}
	if args.Path != "" {
		b, err := os.ReadFile(args.Path)
		if err != nil {
			return "", fmt.Errorf("read git file: %w", err)
		}
		return string(b), nil
	}
	return FetchGitHubPR(ctx, args.PRUrl)
}

// detectAndParse auto-detects git content type and parses accordingly.
func detectAndParse(text string) (string, error) {
	if strings.Contains(text, "diff --git") {
		return ParseGitDiff(text)
	}
	if regexp.MustCompile(`^commit [0-9a-f]{7,}`).MatchString(text) {
		return ParseGitLog(text)
	}
	if regexp.MustCompile(`^\^?[0-9a-f]+ \(`).MatchString(text) {
		return ParseGitBlame(text)
	}
	return text, nil // unknown, return as-is
}
```

Add `"net/url"` import.

- [ ] **Step 4: Add CondenseGitArgs and tool registration to main.go**

```go
type CondenseGitArgs struct {
	Text    string `json:"text,omitempty"     jsonschema:"Raw git diff, log, or blame text"`
	Path    string `json:"path,omitempty"     jsonschema:"Path to file containing diff/log/blame"`
	PRUrl   string `json:"pr_url,omitempty"   jsonschema:"GitHub PR URL (public repos, no auth required)"`
	SkipLLM bool   `json:"skip_llm,omitempty" jsonschema:"Skip LLM Wenyan pass; use mechanical compression only"`
}
```

```go
mcp.AddTool(server, &mcp.Tool{
    Name:        "condense_git",
    Description: "Condense git diffs, logs, blame, or GitHub PRs to Wenyan. Provide one of: text (raw), path (file), pr_url (GitHub PR URL).",
}, func(ctx context.Context, _ *mcp.CallToolRequest, args CondenseGitArgs) (*mcp.CallToolResult, any, error) {
    raw, err := resolveGitInput(ctx, args)
    if err != nil {
        return nil, nil, err
    }
    parsed, err := detectAndParse(raw)
    if err != nil {
        return nil, nil, err
    }
    r, err := comp.Condense(ctx, parsed, !args.SkipLLM)
    if err != nil {
        return nil, nil, err
    }
    return resultContent(r), nil, nil
})
```

- [ ] **Step 5: Run tests**

```bash
go test -v -run "TestParseGit|TestCondenseGit"
```
Expected: PASS.

- [ ] **Step 6: Build + commit**

```bash
go build ./...
git add extract_git.go main.go extract_test.go
git commit -m "feat: condense_git tool with diff/log/blame/PR support"
```

---

## Task 10: condense_log tool

**Files:** `extract_log.go`, `main.go`, `extract_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestParseGoStackTrace(t *testing.T) {
	trace := `panic: runtime error: index out of range [5] with length 3

goroutine 1 [running]:
main.processItems(...)
	/home/user/app/main.go:42 +0x1a3
main.handler(0xc000014080)
	/home/user/app/handler.go:18 +0x65
net/http.HandlerFunc.ServeHTTP(...)
	/usr/local/go/src/net/http/server.go:2136 +0x44
net/http.(*ServeMux).ServeHTTP(...)
	/usr/local/go/src/net/http/server.go:2514 +0x149
`

	result := ParseLog(trace)
	if !strings.Contains(result, "main.processItems") {
		t.Errorf("missing app frame; got:\n%s", result)
	}
	if !strings.Contains(result, "main.go:42") {
		t.Errorf("missing file:line; got:\n%s", result)
	}
	// stdlib frames stripped
	if strings.Contains(result, "net/http.HandlerFunc") {
		t.Errorf("stdlib frame not stripped; got:\n%s", result)
	}
}

func TestParsePythonStackTrace(t *testing.T) {
	trace := `Traceback (most recent call last):
  File "app.py", line 42, in handler
    result = process(data)
  File "lib/process.py", line 15, in process
    return parse(raw)
  File "/usr/lib/python3.9/json/__init__.py", line 346, in loads
    return _default_decoder.decode(s)
ValueError: No JSON object could be decoded`

	result := ParseLog(trace)
	if !strings.Contains(result, "ValueError") {
		t.Errorf("missing exception type; got:\n%s", result)
	}
	if !strings.Contains(result, "app.py") {
		t.Errorf("missing app frame; got:\n%s", result)
	}
	// stdlib stripped
	if strings.Contains(result, "/usr/lib/python") {
		t.Errorf("stdlib path not stripped; got:\n%s", result)
	}
}

func TestDeduplicateErrors(t *testing.T) {
	// Same error repeated 3 times
	trace := strings.Repeat(`ERROR connection refused
goroutine 5 [running]:
main.connect(...)
	/app/db.go:22 +0x88

`, 3)

	result := ParseLog(trace)
	// Should see ×3 dedup marker
	if !strings.Contains(result, "×3") && !strings.Contains(result, "x3") {
		t.Errorf("dedup count missing; got:\n%s", result)
	}
}
```

- [ ] **Step 2: Run to verify FAIL**

```bash
go test -v -run "TestParse.*StackTrace|TestDeduplicate"
```

- [ ] **Step 3: Implement extract_log.go**

```go
package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type logEvent struct {
	header string   // error type + message
	frames []string // app-only frames
	lang   string
}

var (
	goGoroutineRe  = regexp.MustCompile(`^goroutine \d+ \[`)
	goFrameRe      = regexp.MustCompile(`^\t(.+\.go:\d+)`)
	goFuncRe       = regexp.MustCompile(`^(\S+)\(`)
	pyTracebackRe  = regexp.MustCompile(`^Traceback \(most recent call last\):`)
	pyFrameRe      = regexp.MustCompile(`^\s+File "(.+)", line (\d+), in (.+)`)
	pyExceptionRe  = regexp.MustCompile(`^(\w+(?:\.\w+)*Error|Exception|Warning|KeyError|ValueError|TypeError|RuntimeError|AttributeError|ImportError|NameError|IndexError|OSError|IOError|FileNotFoundError|PermissionError|StopIteration):? `)
	jsErrorRe      = regexp.MustCompile(`^(?:\w+)?Error:`)
	jsFrameRe      = regexp.MustCompile(`^\s+at (\S+) \((.+:\d+:\d+)\)`)
	javaExRe       = regexp.MustCompile(`^Exception in thread|^\w[\w.]+Exception:`)
	javaFrameRe    = regexp.MustCompile(`^\s+at ([\w.$]+)\((.+\.java:\d+)\)`)
	rustPanicRe    = regexp.MustCompile(`^thread '.+' panicked at`)

	// stdlib/framework path patterns to strip
	stdlibPatterns = []*regexp.Regexp{
		regexp.MustCompile(`/usr/local/go/src/`),
		regexp.MustCompile(`/usr/lib/python`),
		regexp.MustCompile(`^node:internal/`),
		regexp.MustCompile(`node_modules/`),
		regexp.MustCompile(`\bjava\.`),
		regexp.MustCompile(`\bsun\.`),
		regexp.MustCompile(`\bcom\.sun\.`),
		regexp.MustCompile(`^net/http\.`),
		regexp.MustCompile(`^runtime\.`),
		regexp.MustCompile(`^reflect\.`),
		regexp.MustCompile(`^testing\.`),
	}
)

func isStdlibFrame(frame string) bool {
	for _, re := range stdlibPatterns {
		if re.MatchString(frame) {
			return true
		}
	}
	return false
}

func detectLang(text string) string {
	switch {
	case goGoroutineRe.MatchString(text):
		return "go"
	case pyTracebackRe.MatchString(text):
		return "python"
	case jsErrorRe.MatchString(text):
		return "javascript"
	case javaExRe.MatchString(text):
		return "java"
	case rustPanicRe.MatchString(text):
		return "rust"
	default:
		return "generic"
	}
}

func ParseLog(text string) string {
	lang := detectLang(text)
	events := splitEvents(text, lang)
	if len(events) == 0 {
		return text
	}

	// Deduplicate
	counts := map[string]int{}
	var order []string
	for _, e := range events {
		key := e.header + "|" + strings.Join(e.frames, "|")
		if counts[key] == 0 {
			order = append(order, key)
		}
		counts[key]++
	}

	// Sort by frequency
	sort.Slice(order, func(i, j int) bool {
		return counts[order[i]] > counts[order[j]]
	})

	// Rebuild event map for output
	eventMap := map[string]*logEvent{}
	for _, e := range events {
		key := e.header + "|" + strings.Join(e.frames, "|")
		if eventMap[key] == nil {
			eventMap[key] = e
		}
	}

	var sb strings.Builder
	for _, key := range order {
		e := eventMap[key]
		count := counts[key]
		sb.WriteString(e.header)
		if count > 1 {
			sb.WriteString(fmt.Sprintf(" ×%d", count))
		}
		sb.WriteRune('\n')
		for _, f := range e.frames {
			sb.WriteString("  " + f + "\n")
		}
		sb.WriteRune('\n')
	}
	return strings.TrimSpace(sb.String())
}

func splitEvents(text, lang string) []*logEvent {
	var events []*logEvent
	var current *logEvent
	var currentFunc string

	flush := func() {
		if current != nil && current.header != "" {
			events = append(events, current)
			current = nil
		}
	}

	for _, line := range strings.Split(text, "\n") {
		switch lang {
		case "go":
			if goGoroutineRe.MatchString(line) || strings.HasPrefix(line, "panic:") || strings.HasPrefix(line, "ERROR ") || strings.HasPrefix(line, "FATAL ") {
				flush()
				current = &logEvent{header: strings.TrimSpace(line), lang: "go"}
				currentFunc = ""
			} else if current != nil {
				if m := goFuncRe.FindStringSubmatch(line); m != nil && !isStdlibFrame(m[1]) {
					currentFunc = m[1]
				} else if m := goFrameRe.FindStringSubmatch(line); m != nil && currentFunc != "" && !isStdlibFrame(line) {
					current.frames = append(current.frames, currentFunc+" "+m[1])
					currentFunc = ""
				}
			}

		case "python":
			if pyTracebackRe.MatchString(line) {
				flush()
				current = &logEvent{header: "Traceback", lang: "python"}
			} else if pyExceptionRe.MatchString(line) {
				if current != nil {
					current.header = strings.TrimSpace(line)
				} else {
					flush()
					current = &logEvent{header: strings.TrimSpace(line), lang: "python"}
				}
			} else if current != nil {
				if m := pyFrameRe.FindStringSubmatch(line); m != nil && !isStdlibFrame(m[1]) {
					current.frames = append(current.frames, m[3]+" "+m[1]+":"+m[2])
				}
			}

		case "javascript":
			if jsErrorRe.MatchString(line) {
				flush()
				current = &logEvent{header: strings.TrimSpace(line), lang: "javascript"}
			} else if current != nil {
				if m := jsFrameRe.FindStringSubmatch(line); m != nil && !isStdlibFrame(m[1]) {
					current.frames = append(current.frames, m[1]+" "+m[2])
				}
			}

		case "java":
			if javaExRe.MatchString(line) {
				flush()
				current = &logEvent{header: strings.TrimSpace(line), lang: "java"}
			} else if current != nil {
				if m := javaFrameRe.FindStringSubmatch(line); m != nil && !isStdlibFrame(m[1]) {
					current.frames = append(current.frames, m[1]+"("+m[2]+")")
				}
			}

		default: // generic / rust
			if rustPanicRe.MatchString(line) || strings.HasPrefix(line, "ERROR") || strings.HasPrefix(line, "FATAL") || strings.HasPrefix(line, "PANIC") {
				flush()
				current = &logEvent{header: strings.TrimSpace(line), lang: "generic"}
			}
		}
	}
	flush()
	return events
}
```

- [ ] **Step 4: Add CondenseLogArgs and tool to main.go**

```go
type CondenseLogArgs struct {
	Text    string `json:"text,omitempty" jsonschema:"Raw log or stack trace text"`
	Path    string `json:"path,omitempty" jsonschema:"Path to log file"`
	SkipLLM bool   `json:"skip_llm,omitempty" jsonschema:"Skip LLM Wenyan pass"`
}
```

```go
mcp.AddTool(server, &mcp.Tool{
    Name:        "condense_log",
    Description: "Parse and condense error logs and stack traces (Go/Python/JS/Java/Rust). Deduplicates repeated errors. Provide text or path.",
}, func(ctx context.Context, _ *mcp.CallToolRequest, args CondenseLogArgs) (*mcp.CallToolResult, any, error) {
    if (args.Text == "") == (args.Path == "") {
        return nil, nil, fmt.Errorf("condense_log requires exactly one of: text, path")
    }
    raw := args.Text
    if args.Path != "" {
        b, err := os.ReadFile(args.Path)
        if err != nil {
            return nil, nil, fmt.Errorf("read log file: %w", err)
        }
        raw = string(b)
    }
    parsed := ParseLog(raw)
    r, err := comp.Condense(ctx, parsed, !args.SkipLLM)
    if err != nil {
        return nil, nil, err
    }
    return resultContent(r), nil, nil
})
```

Add `"os"` import to main.go if not already present.

- [ ] **Step 5: Run tests**

```bash
go test -v -run "TestParse.*StackTrace|TestDeduplicate"
```
Expected: PASS.

- [ ] **Step 6: Run full test suite**

```bash
go test -v ./...
```
Expected: all PASS.

- [ ] **Step 7: Build final binary**

```bash
go build -o caveman-mcp .
```

- [ ] **Step 8: Commit**

```bash
git add extract_log.go main.go extract_test.go
git commit -m "feat: condense_log tool with multi-language stack trace parsing and dedup"
```

---

## Task 11: Update version + integration smoke test

**Files:** `main.go`

- [ ] **Step 1: Bump version in main.go**

Change `Version: "0.1.0"` → `Version: "0.2.0"`.

- [ ] **Step 2: Run end-to-end smoke test**

```bash
python3 - << 'EOF'
import subprocess, json, select, os

env = dict(os.environ)
# Use actual key or skip LLM
env['OPENROUTER_API_KEY'] = os.environ.get('OPENROUTER_API_KEY', '')

proc = subprocess.Popen(['./caveman-mcp'], stdin=subprocess.PIPE,
    stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, env=env)

def send(msg):
    proc.stdin.write((json.dumps(msg) + '\n').encode()); proc.stdin.flush()

def recv(t=5):
    r, _, _ = select.select([proc.stdout], [], [], t)
    if r:
        line = proc.stdout.readline().decode().strip()
        return json.loads(line) if line else None
    return None

send({"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0"}}})
recv()
send({"jsonrpc":"2.0","method":"notifications/initialized","params":{}})

send({"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}})
r = recv()
tools = [t['name'] for t in r['result']['tools']]
expected = {'condense_url','condense_file','condense_text','condense_git','condense_log'}
missing = expected - set(tools)
if missing:
    print(f"FAIL: missing tools: {missing}")
else:
    print(f"PASS: all 5 tools present: {tools}")

# Mechanical-only test for condense_git
send({"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"condense_git","arguments":{"text":"diff --git a/main.go b/main.go\nindex abc..def 100644\n--- a/main.go\n+++ b/main.go\n@@ -1,3 +1,4 @@\n package main\n+import \"fmt\"\n","skip_llm":True}}})
r = recv()
if r['result'].get('isError'):
    print(f"FAIL condense_git: {r['result']['content'][0]['text']}")
else:
    result = json.loads(r['result']['content'][0]['text'])
    print(f"PASS condense_git: ratio={result['ratio']}, method={result['method']}")

# condense_log
send({"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"condense_log","arguments":{"text":"goroutine 1 [running]:\nmain.process(...)\n\t/app/main.go:42 +0x88\nnet/http.(*Server).Serve(...)\n\t/usr/local/go/src/net/http/server.go:3086","skip_llm":True}}})
r = recv()
if r['result'].get('isError'):
    print(f"FAIL condense_log: {r['result']['content'][0]['text']}")
else:
    result = json.loads(r['result']['content'][0]['text'])
    print(f"PASS condense_log: {result['compressed'][:80]}")

proc.kill()
EOF
```

Expected output:
```
PASS: all 5 tools present: [...]
PASS condense_git: ratio=..., method=mechanical
PASS condense_log: ...
```

- [ ] **Step 3: Final commit**

```bash
git add main.go
git commit -m "feat: caveman-mcp v0.2.0 — URL routing, 15+ source types, condense_git, condense_log"
```

---

## Notes

- `compress.go` is never touched — all changes are additive
- Image/audio routing lives in `main.go`'s `condense_file` handler (needs `ctx` + `cfg`); text-only file formats stay in `ExtractFile`
- All HTTP clients use `context.Context` for cancellation; tool handler `ctx` propagates through
- `extractGitHubPR` calls `ParseGitDiff` which is defined in `extract_git.go` — ensure `extract_git.go` is created before `extract_sources.go` compiles
- RSS content-type detection: add a HEAD-then-redirect approach inside `extractHTMLPage` if `Content-Type` is `application/rss+xml` or `application/atom+xml`
