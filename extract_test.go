package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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

func TestDescribeImageMock(t *testing.T) {
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
		var parts []interface{}
		if err := json.Unmarshal(req.Messages[0].Content, &parts); err != nil {
			t.Errorf("vision content is not array: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"圖示系統架構三層：前端、後端、資料庫"}}]}`))
	}))
	defer srv.Close()

	cfg := Config{BaseURL: srv.URL, APIKey: "test", Model: "test-vision"}

	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE,
		0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, 0x54,
		0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01,
		0xE2, 0x21, 0xBC, 0x33,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
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

func TestTranscribeAudioSizeGuard(t *testing.T) {
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

func TestExtractPPTX(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

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
	csvData := "name,age,city\nAlice,30,NYC\nBob,25,LA\nCarol,35,Chicago\n"
	tmp, _ := os.CreateTemp("", "test-*.csv")
	tmp.WriteString(csvData)
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
	if strings.Contains(result, "/usr/lib/python") {
		t.Errorf("stdlib path not stripped; got:\n%s", result)
	}
}

func TestDeduplicateErrors(t *testing.T) {
	single := `goroutine 5 [running]:
main.connect(...)
	/app/db.go:22 +0x88

`
	trace := strings.Repeat("ERROR connection refused\n"+single, 3)

	result := ParseLog(trace)
	if !strings.Contains(result, "×3") && !strings.Contains(result, "x3") {
		t.Errorf("dedup count missing; got:\n%s", result)
	}
}

func TestCondenseGitValidation(t *testing.T) {
	args := CondenseGitArgs{Text: "some diff", Path: "/tmp/diff.txt"}
	_, err := resolveGitInput(context.Background(), args)
	if err == nil {
		t.Error("expected error for multiple inputs")
	}
	if !strings.Contains(err.Error(), "exactly one") {
		t.Errorf("wrong error: %v", err)
	}

	args2 := CondenseGitArgs{}
	_, err2 := resolveGitInput(context.Background(), args2)
	if err2 == nil {
		t.Error("expected error for no input")
	}
}
