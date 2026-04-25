# 🪨 Caveman MCP — Me Smash Text, Make Small

> **TL;DR**: Feed URL, file, image, diff, log. Get Wenyan. Big text become tiny text. Code block survive. Number survive. Everything else get crushed by rock.

Two-pass compression: mechanical filler-word drop → LLM Wenyan classical Chinese conversion. Any OpenAI-compatible endpoint (OpenRouter, MiniMax, etc).

---

## Tools

### `condense_url`
Fetch any URL → extract main content → compress.

Smart routing by URL pattern:

| URL | Extractor |
|-----|-----------|
| `youtube.com/watch`, `youtu.be/` | Transcript (VTT captions) |
| `github.com/{owner}/{repo}` | README + file tree + activity |
| `github.com/{owner}/{repo}/pull/{n}` | PR diff + description |
| `arxiv.org/abs/{id}` | Abstract + section headings |
| `news.ycombinator.com/item?id=` | Top 20 comments (HN-ranked) |
| `reddit.com/r/*/comments/*` | Top 20 comments by score |
| RSS/Atom feed | Batch item summaries |
| Anything else | Enhanced HTML → markdown pipeline |

### `condense_file`
Read any local file → extract text → compress.

| Extension | Extractor |
|-----------|-----------|
| `.png .jpg .jpeg .webp .gif` | Vision LLM → wenyan description |
| `.mp3 .wav .m4a .ogg .flac` | Whisper API → transcript → compress |
| `.pptx` | Slide text + speaker notes |
| `.xlsx .csv` | Column stats + top values summary |
| `.pdf` | Text extraction |
| `.docx` | XML text extraction |
| `.md .txt .html .htm .rst` | Raw text / readability |

### `condense_git`
Compress git artifacts. One of `text`, `path`, or `pr_url` required.

Auto-detects format: unified diff → per-file summaries, git log → topic clusters, git blame → authorship summary, GitHub PR URL → fetch + compress.

### `condense_log`
Compress error logs and stack traces.

Detects language (Go / Python / JS / Java / Rust), strips stdlib frames, deduplicates repeated errors with `×N` count, preserves exception type + message + `file:line` exact.

### `condense_text`
Raw text in. Wenyan out. No fetching.

---

## Compression Pipeline

```
input text
  → mechanical pass: drop articles (a/an/the), fillers (just/basically/actually/...),
                     preserve code blocks, URLs, identifiers, numbers
  → LLM Wenyan pass: classical Chinese ultra-compression
      HTML/structured content uses differential pressure:
        h1/h2  → light compression (landmark)
        body   → full wenyan prose
        aside  → ultra-compress or omit
        nav/ad → drop
  → result: {compressed, original_chars, compressed_chars, ratio, method}
```

Typical ratios: 65–80% of original. Code blocks survive intact.

---

## Setup

```bash
go install github.com/beagle/caveman-mcp@latest
# or build from source:
git clone ...
cd caveman-mcp && go build -o caveman-mcp .
```

### Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `LLM_API_KEY` | — | Primary API key |
| `OPENROUTER_API_KEY` | — | Alias for `LLM_API_KEY` |
| `LLM_BASE_URL` | `https://openrouter.ai/api/v1` | LLM endpoint |
| `LLM_MODEL` | `anthropic/claude-haiku-4-5` | Text + vision model |
| `WHISPER_API_KEY` | falls back to `LLM_API_KEY` | Audio transcription |
| `WHISPER_BASE_URL` | `https://openrouter.ai/api/v1` | Whisper endpoint |
| `GITHUB_TOKEN` | — | GitHub API (higher rate limits) |

Set `skip_llm: true` on any tool call for mechanical-only compression (no API cost).

### Claude Code Config

```json
{
  "mcpServers": {
    "caveman": {
      "command": "/path/to/caveman-mcp",
      "env": {
        "OPENROUTER_API_KEY": "sk-or-v1-..."
      }
    }
  }
}
```

---

## Examples

**Condense a webpage:**
```json
{"name": "condense_url", "arguments": {"url": "https://modelcontextprotocol.io/introduction"}}
```
```
MCP（模型語境協議）者，開源標準也，用以聯接AI應用與外部系統...
ratio: 77.1% | 1794 → 1384 chars | mechanical+llm
```

**Condense a git diff:**
```json
{"name": "condense_git", "arguments": {"path": "/tmp/changes.diff"}}
```

**Compress logs:**
```json
{"name": "condense_log", "arguments": {"text": "panic: runtime error: index out of range...\ngoroutine 1 [running]:\nmain.process(...)..."}}
```

**Image description:**
```json
{"name": "condense_file", "arguments": {"path": "/tmp/architecture-diagram.png"}}
```

---

## Optional Dependencies

- **`yt-dlp`** — fallback for YouTube videos without auto-captions (`pip install yt-dlp`)

---

## MCP SDK

Built on the [official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) v1.5.0. Stdio transport.
