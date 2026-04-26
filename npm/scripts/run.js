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
