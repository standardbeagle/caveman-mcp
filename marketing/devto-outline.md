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
