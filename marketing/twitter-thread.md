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
