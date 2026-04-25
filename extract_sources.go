package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	htmlpkg "html"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	nethtml "golang.org/x/net/html"
)

var (
	vttTimestampRe   = regexp.MustCompile(`\d{2}:\d{2}:\d{2}\.\d{3} --> \d{2}:\d{2}:\d{2}\.\d{3}.*`)
	vttTagRe         = regexp.MustCompile(`<[^>]+>`)
	captionTracksRe  = regexp.MustCompile(`"captionTracks":(\[.*?\])`)
	captionBaseURLRe = regexp.MustCompile(`"baseUrl":"(https://[^"]+)"`)
	ogTitleRe        = regexp.MustCompile(`<meta property="og:title" content="([^"]+)"`)
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

func extractYouTube(ctx context.Context, rawURL string) (string, error) {
	_, err := youtubeVideoID(rawURL)
	if err != nil {
		return "", err
	}

	body, _, err := fetchHTML(ctx, rawURL)
	if err != nil {
		return "", fmt.Errorf("fetch youtube page: %w", err)
	}

	title := extractOGTitle(string(body))

	m := captionTracksRe.FindSubmatch(body)
	if m == nil || len(m[1]) == 0 {
		return ytdlpFallback(rawURL, title)
	}

	baseURLMatch := captionBaseURLRe.FindSubmatch(m[1])
	if baseURLMatch == nil {
		return ytdlpFallback(rawURL, title)
	}
	captionURL := string(baseURLMatch[1]) + "&fmt=vtt"
	// Unescape JSON unicode sequences (& → &)
	captionURL = strings.ReplaceAll(captionURL, `&`, "&")

	req, _ := http.NewRequestWithContext(ctx, "GET", captionURL, nil)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
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
}

var githubAPIBase = "https://api.github.com"

func isGitHubPR(u *url.URL) bool {
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	return len(parts) == 4 && parts[2] == "pull"
}

type ghRepoMeta struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Language    string   `json:"language"`
	Stars       int      `json:"stargazers_count"`
	Topics      []string `json:"topics"`
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

func fetchGitHubRepoMeta(ctx context.Context, owner, repo string) (*ghRepoMeta, error) {
	return ghAPIGet[ghRepoMeta](ctx, fmt.Sprintf("/repos/%s/%s", owner, repo))
}

func extractGitHub(ctx context.Context, u *url.URL) (string, error) {
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid GitHub URL: %s", u.String())
	}
	owner, repo := parts[0], parts[1]
	if isGitHubPR(u) {
		return extractGitHubPR(ctx, owner, repo, parts[3])
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
		Truncated bool                                   `json:"truncated"`
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
		Title        string `json:"title"`
		Body         string `json:"body"`
		State        string `json:"state"`
		User         struct{ Login string `json:"login"` } `json:"user"`
		Additions    int    `json:"additions"`
		Deletions    int    `json:"deletions"`
		ChangedFiles int    `json:"changed_files"`
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
	filesURL := fmt.Sprintf("/repos/%s/%s/pulls/%s/files?per_page=300", owner, repo, prNum)
	req, _ := http.NewRequestWithContext(ctx, "GET", githubAPIBase+filesURL, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err == nil {
		defer resp.Body.Close()
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

func extractArXiv(ctx context.Context, rawURL string) (string, error) {
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
	return extractHTMLText(string(body)), nil
}

func extractHTMLText(htmlStr string) string {
	var sb strings.Builder
	dec := nethtml.NewTokenizer(strings.NewReader(htmlStr))
	for {
		tt := dec.Next()
		if tt == nethtml.ErrorToken {
			break
		}
		if tt == nethtml.TextToken {
			sb.Write(dec.Text())
		}
	}
	return strings.TrimSpace(sb.String())
}

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

	limit := 20
	if len(story.Kids) < limit {
		limit = len(story.Kids)
	}

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
		text := htmlpkg.UnescapeString(vttTagRe.ReplaceAllString(r.item.Text, ""))
		sb.WriteString(fmt.Sprintf("**%s:** %s\n\n", r.item.By, text))
		replyLimit := 2
		if len(r.item.Kids) < replyLimit {
			replyLimit = len(r.item.Kids)
		}
		for _, replyID := range r.item.Kids[:replyLimit] {
			reply, err := hnGetItem(ctx, replyID)
			if err == nil && reply != nil {
				replyText := htmlpkg.UnescapeString(vttTagRe.ReplaceAllString(reply.Text, ""))
				sb.WriteString(fmt.Sprintf("  > **%s:** %s\n\n", reply.By, replyText))
			}
		}
	}

	return sb.String(), nil
}

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

	type redditChild struct {
		Kind string `json:"kind"`
		Data struct {
			Title    string          `json:"title"`
			Selftext string          `json:"selftext"`
			Author   string          `json:"author"`
			Score    int             `json:"score"`
			Body     string          `json:"body"`
			Replies  json.RawMessage `json:"replies"`
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
		sb.WriteString(fmt.Sprintf("[%d] u/%s: %s\n\n", c.score, c.author, c.body))
	}

	return sb.String(), nil
}

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
	}
	type feed struct {
		Channel channel `xml:"channel"`
		Title   string  `xml:"title"`
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

	for _, item := range f.Channel.Items {
		sb.WriteString("## " + item.Title + "\n\n")
		content := item.Content
		if content == "" {
			content = item.Description
		}
		sb.WriteString(content + "\n\n")
	}
	for _, e := range f.Entries {
		sb.WriteString("## " + e.Title + "\n\n")
		sb.WriteString(e.Summary + "\n\n")
	}

	return strings.TrimSpace(sb.String()), nil
}

func DescribeImage(ctx context.Context, path string, cfg Config) (string, error) {
	return "", nil // TODO
}

func TranscribeAudio(ctx context.Context, path string, cfg Config) (string, error) {
	return "", nil // TODO
}
