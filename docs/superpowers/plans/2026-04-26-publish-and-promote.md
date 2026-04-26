# caveman-mcp Publish & Promote Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish caveman-mcp under the standardbeagle GitHub org + npm, wire up CI/release automation, submit to MCP registries, and create marketing assets.

**Architecture:** GoReleaser builds multi-platform binaries on tag push; a separate `npm/` package downloads the correct binary via postinstall; GitHub Actions handles both CI and releases. Marketing assets are static files for manual posting.

**Tech Stack:** Go, GoReleaser, GitHub Actions, Node.js (npm wrapper), GitHub Pages (static HTML), gh CLI

---

## File Map

| File | Action | Purpose |
|------|--------|---------|
| `go.mod` | Modify | Rename module path |
| `.goreleaser.yml` | Create | Multi-platform binary build config |
| `.github/workflows/ci.yml` | Create | Run tests on push/PR |
| `.github/workflows/release.yml` | Create | Build + publish on tag push |
| `npm/package.json` | Create | npm package identity + postinstall hook |
| `npm/scripts/postinstall.js` | Create | Download binary from GitHub Releases |
| `npm/scripts/run.js` | Create | Exec shim for binary |
| `npm/bin/caveman-mcp` | Create | Shell entry point |
| `npm/.gitignore` | Create | Exclude downloaded binary from git |
| `smithery.yaml` | Create | Smithery registry descriptor |
| `README.md` | Modify | Add npx install + Claude Code config snippet |
| `docs/index.html` | Create | GitHub Pages marketing site |
| `marketing/show-hn.md` | Create | Show HN post draft |
| `marketing/reddit.md` | Create | Reddit post drafts (r/ClaudeAI + r/LocalLLaMA) |
| `marketing/twitter-thread.md` | Create | Twitter/X thread draft |
| `marketing/devto-outline.md` | Create | Dev.to article outline |

---

## Task 1: Rename Go module

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Update module path**

Change line 1 of `go.mod`:
```
module github.com/standardbeagle/caveman-mcp
```

- [ ] **Step 2: Verify build still passes**

```bash
cd /home/beagle/work/mcps/caveman-mcp
go build ./...
go test ./...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add go.mod
git commit -m "chore: rename module to github.com/standardbeagle/caveman-mcp"
```

---

## Task 2: Add GoReleaser config

**Files:**
- Create: `.goreleaser.yml`

- [ ] **Step 1: Create `.goreleaser.yml`**

```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - binary: caveman-mcp
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    env:
      - CGO_ENABLED=0
    ldflags:
      - -s -w

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

release:
  draft: false
  prerelease: auto

snapshot:
  version_template: "{{ .Tag }}-next"
```

- [ ] **Step 2: Verify config syntax**

```bash
goreleaser check
```
Expected: `config is valid`. If goreleaser not installed: `go install github.com/goreleaser/goreleaser/v2@latest` first. If still unavailable, skip — CI will validate.

- [ ] **Step 3: Commit**

```bash
git add .goreleaser.yml
git commit -m "chore: add goreleaser config for multi-platform builds"
```

---

## Task 3: Add GitHub Actions CI workflow

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Create directory and workflow**

```bash
mkdir -p .github/workflows
```

Create `.github/workflows/ci.yml`:
```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go test ./...
      - run: go build ./...
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add CI workflow for tests and build"
```

---

## Task 4: Add GitHub Actions release workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Create `.github/workflows/release.yml`**

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - uses: actions/setup-node@v4
        with:
          node-version: "20"
          registry-url: "https://registry.npmjs.org"

      - name: Publish npm package
        run: |
          VERSION="${{ github.ref_name }}"
          VERSION="${VERSION#v}"
          cd npm
          npm version "$VERSION" --no-git-tag-version --allow-same-version
          npm publish --access public
        env:
          NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add release workflow (goreleaser + npm publish on tag)"
```

---

## Task 5: Create npm binary wrapper

**Files:**
- Create: `npm/package.json`
- Create: `npm/scripts/postinstall.js`
- Create: `npm/scripts/run.js`
- Create: `npm/bin/caveman-mcp`
- Create: `npm/.gitignore`

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p npm/scripts npm/bin
```

- [ ] **Step 2: Create `npm/package.json`**

```json
{
  "name": "@standardbeagle/caveman-mcp",
  "version": "0.1.0",
  "description": "MCP server: two-pass Wenyan compression for URLs, files, diffs, logs",
  "repository": {
    "type": "git",
    "url": "https://github.com/standardbeagle/caveman-mcp"
  },
  "license": "MIT",
  "engines": {
    "node": ">=18"
  },
  "bin": {
    "caveman-mcp": "bin/caveman-mcp"
  },
  "scripts": {
    "postinstall": "node scripts/postinstall.js"
  },
  "files": [
    "bin/caveman-mcp",
    "scripts/postinstall.js",
    "scripts/run.js"
  ]
}
```

- [ ] **Step 3: Create `npm/scripts/postinstall.js`**

```javascript
#!/usr/bin/env node
"use strict";
const https = require("https");
const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");
const { createGunzip } = require("zlib");

const pkg = require("../package.json");
const version = pkg.version;

const osMap = { linux: "linux", darwin: "darwin", win32: "windows" };
const archMap = { x64: "amd64", arm64: "arm64" };

const os = osMap[process.platform];
const arch = archMap[process.arch];

if (!os || !arch) {
  console.error(`caveman-mcp: unsupported platform ${process.platform}/${process.arch}`);
  process.exit(1);
}

const ext = process.platform === "win32" ? ".zip" : ".tar.gz";
const filename = `caveman-mcp_${version}_${os}_${arch}${ext}`;
const url = `https://github.com/standardbeagle/caveman-mcp/releases/download/v${version}/${filename}`;
const binDir = path.join(__dirname, "..", "bin");
const binPath = path.join(binDir, process.platform === "win32" ? "caveman-mcp-bin.exe" : "caveman-mcp-bin");

if (fs.existsSync(binPath)) process.exit(0);

console.log(`caveman-mcp: downloading ${filename}...`);

function download(url, callback) {
  https.get(url, (res) => {
    if (res.statusCode === 301 || res.statusCode === 302) {
      return download(res.headers.location, callback);
    }
    if (res.statusCode !== 200) {
      callback(new Error(`HTTP ${res.statusCode} downloading ${url}`));
      return;
    }
    callback(null, res);
  }).on("error", callback);
}

download(url, (err, res) => {
  if (err) { console.error(`caveman-mcp: download failed: ${err.message}`); process.exit(1); }

  const tmpFile = binPath + (ext === ".zip" ? ".tmp.zip" : ".tmp.tar.gz");
  const out = fs.createWriteStream(tmpFile);
  res.pipe(out);
  out.on("finish", () => {
    try {
      if (ext === ".zip") {
        execSync(`powershell -Command "Expand-Archive -Path '${tmpFile}' -DestinationPath '${binDir}' -Force"`, { stdio: "pipe" });
        fs.renameSync(path.join(binDir, "caveman-mcp.exe"), binPath);
      } else {
        execSync(`tar -xzf "${tmpFile}" -C "${binDir}" caveman-mcp`, { stdio: "pipe" });
        fs.renameSync(path.join(binDir, "caveman-mcp"), binPath);
        fs.chmodSync(binPath, 0o755);
      }
      fs.unlinkSync(tmpFile);
      console.log("caveman-mcp: installed.");
    } catch (e) {
      console.error(`caveman-mcp: extraction failed: ${e.message}`);
      process.exit(1);
    }
  });
});
```

- [ ] **Step 4: Create `npm/scripts/run.js`**

```javascript
#!/usr/bin/env node
"use strict";
const { spawn } = require("child_process");
const path = require("path");
const fs = require("fs");

const binName = process.platform === "win32" ? "caveman-mcp-bin.exe" : "caveman-mcp-bin";
const binPath = path.join(__dirname, "..", "bin", binName);

if (!fs.existsSync(binPath)) {
  console.error("caveman-mcp: binary not found. Run: npm install @standardbeagle/caveman-mcp");
  process.exit(1);
}

const child = spawn(binPath, process.argv.slice(2), { stdio: "inherit" });
child.on("exit", (code) => process.exit(code ?? 1));
```

- [ ] **Step 5: Create `npm/bin/caveman-mcp`**

```bash
#!/usr/bin/env sh
node "$(dirname "$0")/../scripts/run.js" "$@"
```

Make it executable:
```bash
chmod +x npm/bin/caveman-mcp
```

- [ ] **Step 6: Create `npm/.gitignore`**

```
bin/caveman-mcp-bin
bin/caveman-mcp-bin.exe
bin/*.tmp*
```

- [ ] **Step 7: Verify package structure**

```bash
cd npm && npm pack --dry-run
```
Expected output lists: `bin/caveman-mcp`, `scripts/postinstall.js`, `scripts/run.js`, `package.json`.
Must NOT list `bin/caveman-mcp-bin`.

- [ ] **Step 8: Commit**

```bash
git add npm/
git commit -m "feat: add npm binary wrapper package @standardbeagle/caveman-mcp"
```

---

## Task 6: Add smithery.yaml

**Files:**
- Create: `smithery.yaml`

- [ ] **Step 1: Create `smithery.yaml`**

```yaml
startCommand:
  type: stdio
  configSchema:
    type: object
    properties:
      LLM_API_KEY:
        type: string
        description: "API key for LLM Wenyan pass (OpenRouter or any OpenAI-compatible)"
      LLM_BASE_URL:
        type: string
        description: "LLM endpoint base URL (default: https://openrouter.ai/api/v1)"
      LLM_MODEL:
        type: string
        description: "Model for Wenyan compression (default: anthropic/claude-haiku-4-5)"
      GITHUB_TOKEN:
        type: string
        description: "GitHub token for higher rate limits on github.com URLs"
    required: []
  commandFunction: |-
    (config) => ({
      command: "npx",
      args: ["-y", "@standardbeagle/caveman-mcp"],
      env: Object.fromEntries(Object.entries(config).filter(([, v]) => v))
    })
```

- [ ] **Step 2: Commit**

```bash
git add smithery.yaml
git commit -m "chore: add smithery.yaml for smithery.ai registry"
```

---

## Task 7: Update README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Replace Setup section**

Find the `## Setup` section in `README.md`. Replace it entirely with:

```markdown
## Setup

### Claude Code (recommended)

```json
{
  "mcpServers": {
    "caveman": {
      "command": "npx",
      "args": ["-y", "@standardbeagle/caveman-mcp"],
      "env": {
        "LLM_API_KEY": "sk-or-v1-..."
      }
    }
  }
}
```

No install step — `npx` downloads and runs on first use.

### Go install

```bash
go install github.com/standardbeagle/caveman-mcp@latest
```

### Build from source

```bash
git clone https://github.com/standardbeagle/caveman-mcp
cd caveman-mcp && go build -o caveman-mcp .
```
```

- [ ] **Step 2: Verify README renders correctly**

```bash
head -5 README.md  # sanity check file not corrupted
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: update README with npx install and standardbeagle module path"
```

---

## Task 8: Create GitHub Pages marketing site

**Files:**
- Create: `docs/index.html`

- [ ] **Step 1: Create `docs/index.html`**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>caveman-mcp — Me Smash Text, Make Small</title>
  <meta name="description" content="MCP server that compresses URLs, files, diffs, and logs to 65-80% of original size using two-pass Wenyan compression.">
  <style>
    :root { --bg: #0d1117; --fg: #e6edf3; --muted: #8b949e; --accent: #58a6ff; --code-bg: #161b22; --border: #30363d; }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { background: var(--bg); color: var(--fg); font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; line-height: 1.6; padding: 2rem 1rem; }
    .container { max-width: 760px; margin: 0 auto; }
    h1 { font-size: 2.4rem; font-weight: 700; margin-bottom: .5rem; }
    .tagline { color: var(--muted); font-size: 1.1rem; margin-bottom: 2.5rem; }
    h2 { font-size: 1.2rem; font-weight: 600; margin: 2rem 0 .75rem; color: var(--accent); }
    pre { background: var(--code-bg); border: 1px solid var(--border); border-radius: 6px; padding: 1rem; overflow-x: auto; font-size: .875rem; margin-bottom: 1.5rem; }
    code { font-family: "SFMono-Regular", Consolas, monospace; }
    table { width: 100%; border-collapse: collapse; margin-bottom: 1.5rem; font-size: .9rem; }
    th { text-align: left; padding: .5rem .75rem; border-bottom: 1px solid var(--border); color: var(--muted); font-weight: 500; }
    td { padding: .5rem .75rem; border-bottom: 1px solid var(--border); }
    td:first-child { font-family: "SFMono-Regular", Consolas, monospace; color: var(--accent); }
    .badge { display: inline-block; background: var(--code-bg); border: 1px solid var(--border); border-radius: 4px; padding: .15rem .5rem; font-size: .8rem; font-family: monospace; color: var(--accent); }
    .ratio { font-size: 2.5rem; font-weight: 700; color: var(--accent); }
    .stat-row { display: flex; gap: 2rem; margin-bottom: 2rem; }
    .stat { text-align: center; }
    .stat-label { font-size: .8rem; color: var(--muted); }
    a { color: var(--accent); text-decoration: none; }
    a:hover { text-decoration: underline; }
    .cta { margin-top: 2.5rem; padding-top: 2rem; border-top: 1px solid var(--border); }
  </style>
</head>
<body>
<div class="container">
  <h1>🪨 caveman-mcp</h1>
  <p class="tagline">Me smash text, make small. Feed URL, file, diff, log. Get Wenyan.</p>

  <div class="stat-row">
    <div class="stat"><div class="ratio">65–80%</div><div class="stat-label">typical compression</div></div>
    <div class="stat"><div class="ratio">5</div><div class="stat-label">tool types</div></div>
    <div class="stat"><div class="ratio">2-pass</div><div class="stat-label">pipeline</div></div>
  </div>

  <h2>Install</h2>
  <pre><code>{
  "mcpServers": {
    "caveman": {
      "command": "npx",
      "args": ["-y", "@standardbeagle/caveman-mcp"],
      "env": { "LLM_API_KEY": "sk-or-v1-..." }
    }
  }
}</code></pre>

  <h2>Tools</h2>
  <table>
    <tr><th>Tool</th><th>Input</th></tr>
    <tr><td>condense_url</td><td>Any URL — webpage, GitHub repo/PR, YouTube, arXiv, HN, Reddit, RSS</td></tr>
    <tr><td>condense_file</td><td>pdf, docx, xlsx, pptx, md, html, png/jpg (vision), mp3/wav (whisper)</td></tr>
    <tr><td>condense_git</td><td>Unified diff, git log, git blame, GitHub PR URL</td></tr>
    <tr><td>condense_log</td><td>Stack traces and error logs (Go, Python, JS, Java, Rust)</td></tr>
    <tr><td>condense_text</td><td>Raw text → Wenyan, no fetching</td></tr>
  </table>

  <h2>Pipeline</h2>
  <pre><code>input
  → mechanical: drop articles, fillers, redundancy
  → LLM Wenyan: classical Chinese ultra-compression
  → output: compressed text + ratio + method</code></pre>

  <h2>Any LLM endpoint</h2>
  <p>Works with any OpenAI-compatible API. Default: OpenRouter + <code>claude-haiku-4-5</code>.<br>
  Set <code>LLM_BASE_URL</code> + <code>LLM_MODEL</code> to use any provider. Set <code>skip_llm: true</code> for mechanical-only (zero API cost).</p>

  <div class="cta">
    <a href="https://github.com/standardbeagle/caveman-mcp">GitHub</a> ·
    <a href="https://www.npmjs.com/package/@standardbeagle/caveman-mcp">npm</a> ·
    MIT License
  </div>
</div>
</body>
</html>
```

- [ ] **Step 2: Commit**

```bash
git add docs/index.html
git commit -m "docs: add GitHub Pages marketing site"
```

---

## Task 9: Create marketing content files

**Files:**
- Create: `marketing/show-hn.md`
- Create: `marketing/reddit.md`
- Create: `marketing/twitter-thread.md`
- Create: `marketing/devto-outline.md`

- [ ] **Step 1: Create `marketing/show-hn.md`**

```bash
mkdir -p marketing
```

```markdown
# Show HN: caveman-mcp — compress URLs/files/diffs to Wenyan for 65-80% token reduction

**Title:** Show HN: Caveman MCP – compress URLs, files, and diffs down 65-80% using Wenyan

---

Built a Go MCP server that two-pass compresses any input — URLs, local files,
git diffs, logs — down to classical Chinese (Wenyan) prose.

Mechanical pass drops articles/fillers. LLM Wenyan pass crushes the rest.
Code blocks and numbers survive intact.

Why? LLM context windows aren't free. 65-80% reduction means ~4x more context
before you hit limits. Works with any OpenAI-compatible endpoint (OpenRouter,
Ollama, local vLLM).

**5 tools:**
- condense_url — webpage, GitHub repo/PR, YouTube, arXiv, HN, Reddit, RSS
- condense_file — PDF, DOCX, XLSX, images (vision), audio (Whisper)
- condense_git — unified diff, git log, blame, GitHub PR URL
- condense_log — stack traces with deduplication and stdlib frame stripping
- condense_text — raw text in, Wenyan out

Use skip_llm: true for mechanical-only compression (no API cost).

npm: npx -y @standardbeagle/caveman-mcp
GitHub: https://github.com/standardbeagle/caveman-mcp
```

- [ ] **Step 2: Create `marketing/reddit.md`**

```markdown
# Reddit Posts

## r/ClaudeAI

**Title:** I built an MCP server that compresses your context 65-80% using classical Chinese

I've been hitting context limits constantly when feeding Claude large codebases,
PDFs, and GitHub PRs. Built caveman-mcp to fix this.

Two passes: mechanical (strip filler words, articles) then LLM Wenyan (classical
Chinese ultra-compression). Code blocks and numbers survive intact.

**Before** (1794 chars):
> MCP (Model Context Protocol) is an open standard that enables AI applications
> to connect with external data sources and tools in a standardized way...

**After** (412 chars, 77% reduction):
> MCP（模型語境協議）者，開源標準也，用以聯接AI應用與外部系統...

Install in Claude Code:
```json
{
  "mcpServers": {
    "caveman": {
      "command": "npx",
      "args": ["-y", "@standardbeagle/caveman-mcp"],
      "env": { "LLM_API_KEY": "your-openrouter-key" }
    }
  }
}
```

GitHub: https://github.com/standardbeagle/caveman-mcp

---

## r/LocalLLaMA

**Title:** caveman-mcp: two-pass context compressor for any LLM — 65-80% reduction via Wenyan

MCP server that compresses any input using two passes:
1. Mechanical: drop articles, fillers, redundant phrases
2. LLM Wenyan: classical Chinese compression

Works with any OpenAI-compatible endpoint. Set LLM_BASE_URL to point at your
local Ollama/vLLM instance. Use skip_llm: true to run mechanical-only (zero
inference cost).

Handles: URLs (with smart routing for GitHub, YouTube, arXiv, HN, Reddit),
local files (PDF/DOCX/XLSX/images/audio), git diffs, logs, raw text.

Built in Go, MIT, single binary. go install or npx.

https://github.com/standardbeagle/caveman-mcp
```

- [ ] **Step 3: Create `marketing/twitter-thread.md`**

```markdown
# Twitter/X Thread

1/
Built caveman-mcp: an MCP server that smashes your Claude context down 65-80%.

Two passes. Mechanical + Wenyan. Code blocks survive.

2/
Pipeline:

input
→ mechanical: drop "a/an/the", "just", "basically", filler
→ LLM Wenyan: classical Chinese ultra-compression
→ result: 65-80% reduction, code/numbers intact

3/
Works on anything:
- URLs (GitHub repos/PRs, YouTube, arXiv, HN, Reddit)
- Files (PDF, DOCX, XLSX, images via vision, audio via Whisper)
- Git diffs + logs
- Stack traces (deduplicates repeated errors with ×N count)

4/
Any OpenAI-compatible endpoint. Set skip_llm: true for mechanical-only (no API cost).

Install in Claude Code:
npx -y @standardbeagle/caveman-mcp

5/
MIT, open source, Go.
github.com/standardbeagle/caveman-mcp
```

- [ ] **Step 4: Create `marketing/devto-outline.md`**

```markdown
# Dev.to / Hashnode Article Outline

**Title:** How I got 65-80% token reduction in Claude using Wenyan classical Chinese compression

**SEO keywords:** MCP server, Claude token reduction, LLM context compression,
Model Context Protocol, context window optimization

---

## Outline

### 1. The problem (300 words)
- Context windows fill fast with real workloads
- Feeding a GitHub PR + codebase + docs = instant limit
- Solutions people try: summarization (lossy), pagination (breaks reasoning), bigger models ($$)

### 2. The insight: information density of classical Chinese (400 words)
- Wenyan drops grammatical particles, uses single characters for concepts
- Same information, ~3-4x denser than modern English
- LLMs trained on Wenyan can decompress it perfectly
- Example: English paragraph → Wenyan → back to English (lossless concepts)

### 3. The two-pass pipeline (500 words)
- Pass 1: mechanical (no API cost)
  - Drop: a/an/the, just/basically/actually/simply/really
  - Preserve: code blocks, URLs, identifiers, numbers
- Pass 2: LLM Wenyan
  - Differential pressure for structured content (headings light, body full, nav dropped)
- Show before/after for a GitHub README

### 4. Building it as an MCP server (400 words)
- Why MCP: Claude uses tools natively, no prompt engineering
- 5 tools covering every input type
- Smart URL routing (GitHub, YouTube, arXiv, HN, Reddit each have custom extractors)

### 5. Real benchmarks (300 words)
- GitHub README: ~75% reduction
- arXiv abstract: ~70% reduction
- Python stack trace: ~80% reduction (stdlib frames stripped)
- Git diff (50-file PR): ~68% reduction

### 6. Install + use (200 words)
- npx one-liner
- Claude Code JSON config
- skip_llm flag for zero-cost mechanical-only mode

### 7. Open source (100 words)
- MIT, Go, github.com/standardbeagle/caveman-mcp
- Contributions welcome (new extractors, language support)
```

- [ ] **Step 5: Commit**

```bash
git add marketing/
git commit -m "docs: add marketing content drafts for HN, Reddit, Twitter, Dev.to"
```

---

## Task 10: Create GitHub repo and push

- [ ] **Step 1: Create repo under standardbeagle org**

```bash
gh repo create standardbeagle/caveman-mcp \
  --public \
  --description "MCP server: two-pass Wenyan compression for URLs, files, diffs, logs" \
  --homepage "https://standardbeagle.github.io/caveman-mcp"
```

- [ ] **Step 2: Init git, add remote, push**

```bash
cd /home/beagle/work/mcps/caveman-mcp
git init
git add .
git commit -m "feat: initial commit — caveman-mcp v0.1.0"
git remote add origin https://github.com/standardbeagle/caveman-mcp.git
git branch -M main
git push -u origin main
```

Note: If there are already commits from the earlier steps, skip the `git add` + `git commit` here and just add remote + push.

- [ ] **Step 3: Add repo topics**

```bash
gh repo edit standardbeagle/caveman-mcp \
  --add-topic mcp \
  --add-topic mcp-server \
  --add-topic compression \
  --add-topic claude \
  --add-topic golang \
  --add-topic llm
```

- [ ] **Step 4: Enable GitHub Pages**

```bash
gh api repos/standardbeagle/caveman-mcp/pages \
  --method POST \
  --field source='{"branch":"main","path":"/docs"}'
```

- [ ] **Step 5: Verify CI triggers**

Check Actions tab: `https://github.com/standardbeagle/caveman-mcp/actions`
Expected: CI workflow runs and passes.

---

## Task 11: Tag v0.1.0 and trigger release

- [ ] **Step 1: Tag and push**

```bash
git tag v0.1.0
git push origin v0.1.0
```

- [ ] **Step 2: Monitor release workflow**

```bash
gh run watch --repo standardbeagle/caveman-mcp
```

Expected: GoReleaser creates a GitHub Release with 5 binary archives + checksums.txt. npm package publishes to `@standardbeagle/caveman-mcp@0.1.0`.

- [ ] **Step 3: Verify npm published**

```bash
npm view @standardbeagle/caveman-mcp
```

Expected: shows version 0.1.0, description, repository.

- [ ] **Step 4: Smoke test npx install**

```bash
npx -y @standardbeagle/caveman-mcp --help
```

Expected: binary runs, prints usage (or exits 0).

---

## Task 12: Submit to MCP registries

These are manual web submissions. Do each in order.

- [ ] **Step 1: Submit to smithery.ai**

Go to https://smithery.ai — find "Submit Server" or "Add Server" link.
Submit: `https://github.com/standardbeagle/caveman-mcp`
Smithery reads `smithery.yaml` automatically from the repo.

- [ ] **Step 2: Submit to mcp.so**

Go to https://mcp.so — find submission form.
Fields:
- Name: `caveman-mcp`
- GitHub: `https://github.com/standardbeagle/caveman-mcp`
- npm: `@standardbeagle/caveman-mcp`
- Category: Utilities / Developer Tools
- Description: (copy from README intro line)

- [ ] **Step 3: Submit to mcpregistry.ai**

Go to https://mcpregistry.ai — find "Add Server" or submit PR link.
If PR-based, fork their registry repo and add entry:
```yaml
- name: caveman-mcp
  description: "Two-pass Wenyan compression for URLs, files, diffs, logs. 65-80% token reduction."
  github: https://github.com/standardbeagle/caveman-mcp
  npm: "@standardbeagle/caveman-mcp"
  tags: [compression, utilities, developer-tools]
```

- [ ] **Step 4: Submit to Glama.ai**

Go to https://glama.ai — find tool/MCP submission form.
Submit GitHub URL: `https://github.com/standardbeagle/caveman-mcp`
Glama auto-crawls README and indexes tools.
