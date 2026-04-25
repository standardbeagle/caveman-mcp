package main

import (
	"context"
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
