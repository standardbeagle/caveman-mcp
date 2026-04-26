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
