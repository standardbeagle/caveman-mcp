package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPlaceholder(t *testing.T) {}

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

func TestExtractGitHubRouting(t *testing.T) {
	cases := []struct {
		path   string
		wantPR bool
	}{
		{"/owner/repo", false},
		{"/owner/repo/", false},
		{"/owner/repo/pull/42", true},
		{"/owner/repo/tree/main", false},
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

	oldGHBase := githubAPIBase
	githubAPIBase = srv.URL
	defer func() { githubAPIBase = oldGHBase }()

	_, err := fetchGitHubRepoMeta(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseHNComments(t *testing.T) {
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
	if !strings.Contains(text, "Getting Started") {
		t.Errorf("h1 not in output; got:\n%s", text)
	}
	if !strings.Contains(text, "Installation") {
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
