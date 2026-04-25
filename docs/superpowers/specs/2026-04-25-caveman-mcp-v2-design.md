# caveman-mcp v2 Design

## Goal

Expand caveman-mcp from 3 generic tools to 4 smart-routing tools that handle web pages, files, developer artifacts, and logs — all compressed to Wenyan classical Chinese via a two-pass pipeline (mechanical pre-pass + LLM differential compression).

## Current State

- `condense_url` — readability TextContent (flat, loses structure) + wenyan LLM
- `condense_file` — md/txt/html/pdf/docx + wenyan LLM
- `condense_text` — raw text + wenyan LLM
- No image, audio, YouTube, GitHub, git, or log support

## Tool Surface (v2)

4 tools. `condense_text` unchanged.

### `condense_url(url, skip_llm?)`

Routes by URL pattern:

| Pattern | Extractor |
|---------|-----------|
| `youtube.com/watch`, `youtu.be/` | YouTube transcript |
| `github.com/{owner}/{repo}` (no subpath) | GitHub repo summary |
| `github.com/{owner}/{repo}/pull/{n}` | GitHub PR diff + description |
| `arxiv.org/abs/{id}` | arXiv abstract + sections |
| `news.ycombinator.com/item?id=` | HN top-20 comments (HN-ranked order) |
| `reddit.com/r/*/comments/*` | Reddit top-20 comments by score |
| RSS/Atom content-type or `.rss`/`.xml` suffix | Feed batch summary |
| * (fallback) | Enhanced generic HTML pipeline |

### `condense_file(path, skip_llm?)`

Routes by file extension:

| Extension | Extractor |
|-----------|-----------|
| `.png .jpg .jpeg .webp .gif .bmp` | Vision LLM → wenyan description |
| `.mp3 .wav .m4a .ogg .flac` | Whisper API → text → condense |
| `.pptx` | Slide XML extractor |
| `.xlsx .csv` | Tabular summarizer |
| `.md .txt .mdx .rst` | Raw text (existing) |
| `.html .htm` | HTML pipeline (existing, enhanced) |
| `.pdf` | PDF text (existing) |
| `.docx` | DOCX XML (existing) |

### `condense_git(text?, path?, pr_url?, skip_llm?)`

Exactly one of `text`, `path`, or `pr_url` required. If zero or multiple provided, return error: `"condense_git requires exactly one of: text, path, pr_url"`.

Arg struct:
```go
type CondenseGitArgs struct {
    Text    string `json:"text,omitempty"    jsonschema:"Raw git diff, log, or blame text"`
    Path    string `json:"path,omitempty"    jsonschema:"Path to file containing diff/log/blame"`
    PRUrl   string `json:"pr_url,omitempty"  jsonschema:"GitHub PR URL (public repos only)"`
    SkipLLM bool   `json:"skip_llm,omitempty" jsonschema:"Skip LLM Wenyan pass"`
}
```
Empty string (`""`) = not provided. Validation checks `len(text) > 0`, `len(path) > 0`, `len(pr_url) > 0` — counts non-empty fields, errors if count != 1.

Auto-detects format:
- **Unified diff** (`diff --git a/` header present): per-file summary + stat line
- **Git log** (`commit [sha]` lines present): cluster commits by topic/prefix
- **Git blame**: file + annotated authorship summary
- **PR URL**: fetch via GitHub API (public repos, no auth)

### `condense_log(text?, path?, skip_llm?)`

Exactly one of `text` or `path` required.

- Detects stack trace language: Go, Python, JavaScript/Node, Java, Rust
- Strips stdlib/framework frames, keeps app frames
- Deduplicates repeated error patterns, appends `×N` count
- Preserves: exception type, message, file:line identifiers exact

---

## Enhanced HTML Pipeline

Used by `condense_url` generic branch and `condense_file` `.html` files.

```
fetch HTML
→ readability(article.Content)          # returns HTML not flat text
→ html-to-markdown                      # preserves h1-h6, ul/ol, tables, code, blockquote
→ signal annotation:
    h1/h2             → [HIGH]
    h3-h6, p          → [MED]
    aside, figcaption → [LOW]
    nav, footer, .ad  → [DROP]
→ mechanical pass: remove [DROP] nodes, strip filler words, collapse whitespace
→ LLM prompt includes signal map:
    HIGH = preserve structure, light wenyan
    MED  = full wenyan prose compression
    LOW  = ultra-compress or omit if redundant
```

Replaces current `article.TextContent` approach which loses all document structure.

HTML-to-markdown library: `github.com/JohannesKaufmann/html-to-markdown/v2`

---

## Vision Pipeline (images via `condense_file`)

```
input path → read bytes (local) or HTTP GET (URL-like path)
→ detect format: png/jpg/webp/gif/bmp
→ base64 encode
→ LLM vision call (model must support vision: claude-haiku, gpt-4o-mini, etc.)
    prompt: "Describe in ≤40 wenyan chars.
             Diagrams/charts: extract structure and data values.
             UI screenshots: list key components and layout.
             Photos: scene + key subjects.
             Preserve ALL text, numbers, identifiers exactly."
→ return: {description, format, size_bytes}
```

Vision requires a model that supports image input. Server checks model capability at startup and logs warning if vision-incompatible model configured.

**Vision request format**: `compress.go` and `llmMessage` are unchanged. Vision calls use a separate `llmVisionReq` struct defined in `extract_sources.go`:
```go
type llmVisionMsg struct {
    Role    string          `json:"role"`
    Content json.RawMessage `json:"content"` // array of content parts
}
type llmVisionReq struct {
    Model    string         `json:"model"`
    Messages []llmVisionMsg `json:"messages"`
}
```
Content is a JSON array:
```json
[
  {"type": "image_url", "image_url": {"url": "data:image/png;base64,{b64}"}},
  {"type": "text", "text": "{prompt}"}
]
```
This leaves `compress.go` and all existing text-only paths untouched.

---

## Source-Specific Extractors

### YouTube

1. Extract video ID from URL (watch?v=, youtu.be/, /embed/)
2. Fetch `https://www.youtube.com/watch?v={id}` — parse `ytInitialPlayerResponse` JSON embedded in page (regex `"captionTracks":(\[.*?\])`)
3. Extract `baseUrl` from first English caption track; append `&fmt=vtt`
4. Fetch signed caption URL → parse VTT: strip timestamps and positioning markup, join lines into paragraphs
5. Fallback: `yt-dlp --write-auto-sub --skip-download --sub-format vtt` subprocess if `captionTracks` absent or empty. Check `exec.LookPath("yt-dlp")` first; error explicitly if not installed.
6. Trigger fallback on: empty body (not just 404) or missing captionTracks
7. Prepend: video title (from og:title in page head)

### GitHub Repo

1. Fetch `https://api.github.com/repos/{owner}/{repo}` — description, language, stars, topics
2. Fetch README: try `main`, `master` branches via raw.githubusercontent.com
3. File tree: `https://api.github.com/repos/{owner}/{repo}/git/trees/HEAD?recursive=1` — top 50 files by path. Check `response.truncated`; if true, log warning and proceed with partial tree.
4. Recent activity: last 10 commits via `/commits` endpoint
5. No auth required for public repos; `GITHUB_TOKEN` env var used if set

### GitHub PR

1. Parse owner/repo/PR number from URL
2. Fetch `https://api.github.com/repos/{owner}/{repo}/pulls/{n}` — title, body, stats
3. Fetch `https://api.github.com/repos/{owner}/{repo}/pulls/{n}/files?per_page=300` — detect truncation by checking `Link` response header for `rel="next"` (standard GitHub pagination), or if returned file count == 300; surface warning if truncated.
4. Fetch full diff: `Accept: application/vnd.github.diff` header (chunked, no Content-Length). Route through `condense_git` diff handler.
5. Prepend PR title + description (condensed)

### arXiv

1. Extract paper ID from URL (`abs/`, `pdf/` patterns)
2. Fetch `https://export.arxiv.org/abs/{id}` HTML
3. Extract: title, authors, abstract, section headings + first paragraph per section
4. Preserve citation keys `[AuthorYYYY]` and formula identifiers exact
5. Strip LaTeX-heavy content (equations) with `[EQ]` placeholder

### Hacker News

1. Extract item ID from URL
2. Fetch `https://hacker-news.firebaseio.com/v0/item/{id}.json`
3. For story: `kids` array contains comment IDs already in HN-ranked order
4. Take first 20 comment IDs from `kids`, fetch each in parallel
5. For each comment fetch top 2 child replies (first 2 of their `kids`)
6. Note: HN comments have no `score` field — use HN's own ranking (kids array order)
7. Format: `author: text` per comment (no score available)

### Reddit

1. Fetch `https://www.reddit.com/r/{sub}/comments/{id}.json`
   User-Agent **must** be Reddit script format: `script:caveman-mcp:v0.2 (by /u/caveman-mcp-bot)` — plain UA is Cloudflare-blocked
2. Check response `Content-Type`: if not `application/json`, return error `"Reddit returned non-JSON (Cloudflare block or subreddit restriction)"` — do not attempt JSON parse
3. Parse comment tree, sort by score descending
3. Top 20 comments, depth ≤ 2
4. Format: `[score] u/author: text` per comment

### RSS/Atom Feed

1. Detect: `Content-Type: application/rss+xml` or `application/atom+xml` or `.rss`/`.xml` suffix with feed root element
2. Parse XML: extract `<item>` or `<entry>` elements
3. Per item: title + description/summary (prefer `content:encoded`)
4. Condense each item independently, return list with titles as headers

### Audio (Whisper API)

1. Check file size: reject > 25MB with error `"audio file exceeds 25MB Whisper API limit: {size}MB"`
2. Read file bytes
3. POST multipart/form-data to `/audio/transcriptions` (OpenAI-compatible)
4. Model: `whisper-1`
5. Timeout: 120s covers entire upload + inference round trip
6. Get transcript text
7. Condense transcript through normal mechanical + LLM pipeline
8. `WHISPER_API_KEY` env var (defaults to `LLM_API_KEY`)

### PPTX

1. Unzip `.pptx` (same approach as `.docx`)
2. Parse `ppt/slides/slide{N}.xml` files in order
3. Extract: `<a:t>` text nodes (title + body), speaker notes from `ppt/notesSlides/notesSlide{N}.xml`
4. Format: `## Slide N: {title}\n{body}\n[Notes: {notes}]`

### XLSX/CSV

1. CSV: parse columns, infer types, compute per-column: count, min, max, mean, top-5 values
2. XLSX: unzip, parse `xl/worksheets/sheet1.xml`, same stats
3. Return structured summary: schema + statistics, not raw data
4. Library: `github.com/xuri/excelize/v2` for XLSX

---

## Git Diff Handler

```
parse unified diff:
  - count files changed, insertions, deletions (stat line)
  - per file: old path, new path, hunk summaries
    - hunk: context lines for locating change, added/removed lines
  - detect: rename (similarity index), mode change, binary file
output format:
  [+N -M in K files]
  path/to/file.go: {one-line summary of what changed}
  path/to/other.go: {one-line summary}
→ LLM wenyan compression of summaries
```

### Git Log Handler

```
parse commits: sha (short 8), author, date, subject, body
group by conventional commit prefix: feat/fix/chore/refactor/docs/test
output format:
  feat (N): {subjects joined}
  fix  (M): {subjects joined}
  ...
→ LLM wenyan compression
```

---

## Log / Stack Trace Handler

Language detection heuristics:

| Language | Signal |
|----------|--------|
| Go | `goroutine \d+ \[`, `runtime/` frames |
| Python | `Traceback (most recent call last):`, `.py:` |
| JavaScript | `at \w+ \(.*\.js:\d+`, `Error:` prefix |
| Java | `at com.\|org.\|java.` frames, `Exception in thread` |
| Rust | `thread '.*' panicked at` |

Processing:
1. Split into error events (blank line or new exception header)
2. Per event: extract type + message (preserve exact) + app frames (skip stdlib)
3. Deduplicate identical events, append `×N`
4. Sort by frequency descending

---

## Error Handling

- URL fetch timeout: 30s (web), 10s (APIs)
- Audio: reject > 25MB before upload; 120s timeout covers full upload + inference
- Vision LLM: 120s timeout
- YouTube: empty captionTracks or empty body → try yt-dlp; `exec.LookPath("yt-dlp")` fails → error "yt-dlp not installed; install with pip install yt-dlp"
- GitHub API 403/404 → surface error with reason (rate limit vs not found)
- Vision model incapable → error with model name and suggestion
- All errors: fail fast, no dummy data, no silent fallback

---

## Configuration (env vars)

| Var | Default | Purpose |
|-----|---------|---------|
| `LLM_API_KEY` | — | Primary LLM key (OpenRouter or MiniMax) |
| `OPENROUTER_API_KEY` | — | Alias for LLM_API_KEY |
| `LLM_BASE_URL` | `https://openrouter.ai/api/v1` | LLM endpoint |
| `LLM_MODEL` | `anthropic/claude-haiku-4-5` | Text + vision model |
| `WHISPER_API_KEY` | falls back to `LLM_API_KEY` | Audio transcription |
| `WHISPER_BASE_URL` | `https://openrouter.ai/api/v1` | Whisper endpoint |
| `GITHUB_TOKEN` | — | GitHub API (higher rate limit) |

---

## File Structure

```
caveman-mcp/
├── main.go              # server setup, tool registration
├── compress.go          # mechanical pass + LLM wenyan (unchanged)
├── extract.go           # file routing (enhanced)
├── extract_url.go       # URL routing + generic HTML pipeline
├── extract_sources.go   # youtube, github, arxiv, hn, reddit, rss
├── extract_git.go       # diff, log, blame, PR
├── extract_log.go       # stack trace + log parsing
├── compress_test.go     # existing tests
└── extract_test.go      # new extractor tests
```

---

## Out of Scope (v2)

- JavaScript SPA rendering (requires headless browser)
- Notion/Linear/Jira API integrations (need OAuth)
- Slack/Discord exports
- Chunking for very large inputs (>100k chars) — error instead
- Streaming output
