#!/usr/bin/env node
"use strict";
const https = require("https");
const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

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
