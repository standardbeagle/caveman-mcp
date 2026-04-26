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
