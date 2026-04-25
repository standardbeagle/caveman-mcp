package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
