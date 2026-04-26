# caveman-mcp: Publish & Promote Design

**Date:** 2026-04-26  
**Scope:** GitHub org migration, npm binary wrapper, MCP registry submissions, marketing content

---

## 1. Go Module + GitHub Repository

### Module Rename
- `go.mod` module path: `github.com/beagle/caveman-mcp` → `github.com/standardbeagle/caveman-mcp`
- All internal imports updated to match
- No API changes; pure path rename

### GitHub
- Create public repo: `standardbeagle/caveman-mcp`
- Push all current code + history
- Repo settings: public, description = "MCP server: two-pass Wenyan compression for URLs, files, diffs, logs"
- Topics: `mcp`, `mcp-server`, `compression`, `claude`, `llm`, `golang`

### CI/CD (GitHub Actions)
**`.github/workflows/ci.yml`** — runs on push/PR to main:
```
- go test ./...
- go build ./...
```

**`.github/workflows/release.yml`** — runs on `v*` tag push:
```
- uses: goreleaser/goreleaser-action
- env: GITHUB_TOKEN
- creates GitHub Release with binaries
```

**`.goreleaser.yml`** — build targets:
| OS | Arch | Archive |
|----|------|---------|
| linux | amd64 | `.tar.gz` |
| linux | arm64 | `.tar.gz` |
| darwin | amd64 | `.tar.gz` |
| darwin | arm64 | `.tar.gz` |
| windows | amd64 | `.zip` |

Asset naming pattern: `caveman-mcp_{{ .Os }}_{{ .Arch }}` (goreleaser default).  
Checksum file included automatically.

---

## 2. npm Binary Wrapper

### Package identity
- Name: `@standardbeagle/caveman-mcp`
- Scope: `standardbeagle` (public npm org)
- Version: mirrors Go release tag (e.g. `1.0.0` for `v1.0.0`)
- Published with `ORG_NPM_TOKEN` secret in GitHub Actions

### File layout
```
npm/
  package.json
  scripts/
    postinstall.js     ← download binary from GitHub Releases
    run.js             ← exec shim (used by Claude Code via stdio)
  bin/
    caveman-mcp        ← thin shell: node scripts/run.js
```

### `scripts/postinstall.js` logic
1. Read `version` from `package.json`
2. Detect `process.platform` + `process.arch` → map to goreleaser asset name
3. Fetch `https://github.com/standardbeagle/caveman-mcp/releases/download/v${version}/caveman-mcp_${os}_${arch}.tar.gz` (or `.zip` on Windows)
4. Extract binary to `bin/caveman-mcp-bin`
5. `chmod 0755` on posix
6. Error out if platform unsupported (no silent fallback)

### `scripts/run.js` logic
- Resolve `bin/caveman-mcp-bin` relative to `__dirname`
- `child_process.spawn(binary, process.argv.slice(2), { stdio: 'inherit' })`
- Exit with child's exit code

### `package.json` key fields
```json
{
  "name": "@standardbeagle/caveman-mcp",
  "bin": { "caveman-mcp": "bin/caveman-mcp" },
  "scripts": { "postinstall": "node scripts/postinstall.js" },
  "files": ["bin/caveman-mcp", "scripts/"],
  "engines": { "node": ">=18" }
}
```

### npm release automation
Add to `.github/workflows/release.yml` after goreleaser step:
```yaml
- name: Publish npm
  run: |
    cd npm
    npm version ${{ github.ref_name }}  # strips v prefix
    npm publish --access public
  env:
    NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

### Claude Code install snippet (in README)
```json
{
  "mcpServers": {
    "caveman": {
      "command": "npx",
      "args": ["-y", "@standardbeagle/caveman-mcp"],
      "env": { "OPENROUTER_API_KEY": "sk-or-v1-..." }
    }
  }
}
```

---

## 3. MCP Registry Submissions

### smithery.ai
Add `smithery.yaml` to repo root:
```yaml
startCommand:
  type: stdio
  configSchema:
    properties:
      OPENROUTER_API_KEY:
        type: string
        description: "OpenRouter API key for LLM Wenyan pass"
    required: []
  commandFunction: |-
    (config) => ({
      command: "npx",
      args: ["-y", "@standardbeagle/caveman-mcp"],
      env: config
    })
```
Submit via smithery.ai web form with repo URL.

### mcp.so
Web form submission:
- Name: caveman-mcp
- Category: Utilities / Developer Tools
- GitHub URL, npm URL, description from README

### mcpregistry.ai
PR to their registry repo with YAML entry:
```yaml
name: caveman-mcp
description: "Two-pass Wenyan compression for URLs, files, diffs, logs. 65-80% token reduction."
github: https://github.com/standardbeagle/caveman-mcp
npm: "@standardbeagle/caveman-mcp"
tags: [compression, wenyan, mcp, utilities]
```

### Glama.ai
Web form: submit GitHub URL. Glama auto-crawls README and indexes tools.

---

## 4. GitHub Pages Marketing Site

- Source: `docs/` folder, `main` branch, GitHub Pages enabled in repo settings
- Single `docs/index.html` — no build step, no framework
- Content: hero tagline, tools table, install snippet, live compression stats badge
- Google crawls `standardbeagle.github.io/caveman-mcp` without custom domain
- Custom domain optional for SEO uplift; not required for indexing

---

## 5. Marketing Content

### Show HN (Hacker News)
**Title:** `Show HN: Caveman MCP – compress URLs/files/diffs to Wenyan for 65-80% token reduction`

**Body:**
```
Built a Go MCP server that two-pass compresses any input — URLs, local files,
git diffs, logs — down to classical Chinese (Wenyan) prose.

Mechanical pass drops articles/fillers. LLM Wenyan pass crushes the rest.
Code blocks and numbers survive intact.

Why? LLM context windows aren't free. 65-80% reduction means ~4x more context
before you hit limits. Works with any OpenAI-compatible endpoint.

Tools: condense_url, condense_file, condense_git, condense_log, condense_text

npm: npx -y @standardbeagle/caveman-mcp
GitHub: https://github.com/standardbeagle/caveman-mcp
```

### Reddit — r/ClaudeAI
**Title:** `I built an MCP server that compresses your context by 65-80% using classical Chinese`

**Body:** Conversational tone. Show before/after example. Link to GitHub + npm install snippet.

### Reddit — r/LocalLLaMA
**Title:** `caveman-mcp: two-pass context compressor for any LLM — Wenyan pass gives 65-80% reduction`

**Body:** Technical tone. Focus on OpenAI-compatible endpoint support, skip_llm flag, compression pipeline internals. Benchmarks if available.

### Twitter/X Thread
```
1/ Built caveman-mcp: an MCP server that smashes your context down to 65-80% of its original size.

2/ Two passes: mechanical (strip filler words) → LLM Wenyan (classical Chinese ultra-compression). Code blocks + numbers survive intact.

3/ Works on URLs, local files, git diffs, error logs, raw text. Any OpenAI-compatible endpoint.

4/ Install in Claude Code:
npx -y @standardbeagle/caveman-mcp

5/ Open source, MIT. Built in Go.
github.com/standardbeagle/caveman-mcp
```

### Dev.to / Hashnode Article
**Title:** "How I got 65-80% token reduction in Claude using Wenyan classical Chinese compression"

**Outline:**
1. Problem: context windows fill up fast
2. Two-pass pipeline explanation with examples
3. Wenyan as compression format (why it works)
4. MCP integration + install
5. Benchmarks across content types
6. Open source + contribution welcome

**SEO keywords:** MCP server, Claude token reduction, LLM context compression, Model Context Protocol tools

---

## Resolved Decisions

- `@standardbeagle` npm org: exists.
- GitHub org secret: `NPM_TOKEN`.
- Initial version: `v0.1.0`. No full release semantics — tag and ship, no ceremony.

