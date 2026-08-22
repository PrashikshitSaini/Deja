#!/usr/bin/env node

// Downloads the prebuilt Deja binary for the current platform from GitHub
// Releases and installs it next to this script. No network calls happen at
// runtime; Deja itself is fully local-only.

"use strict";

const { execFileSync } = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");
const https = require("https");
const crypto = require("crypto");

const REPO = "PrashikshitSaini/Deja";
const VERSION = require("./package.json").version;

function fail(message) {
  console.error(`deja-install: ${message}`);
  process.exit(1);
}

function platformTarget() {
  const platform = process.platform;
  const arch = process.arch;
  if (platform === "darwin" && arch === "arm64") return "darwin-arm64";
  if (platform === "darwin" && arch === "x64") return "darwin-amd64";
  if (platform === "linux" && arch === "arm64") return "linux-arm64";
  if (platform === "linux" && arch === "x64") return "linux-amd64";
  fail(
    `unsupported platform ${platform}-${arch}. Deja supports macOS and Linux ` +
      `on arm64/amd64. On Windows use WSL2 with Zsh.`
  );
}

function download(url) {
  return new Promise((resolve, reject) => {
    https
      .get(url, (response) => {
        if (
          response.statusCode >= 300 &&
          response.statusCode < 400 &&
          response.headers.location
        ) {
          resolve(download(response.headers.location));
          response.resume();
          return;
        }
        if (response.statusCode !== 200) {
          response.resume();
          reject(new Error(`HTTP ${response.statusCode} for ${url}`));
          return;
        }
        const chunks = [];
        response.on("data", (chunk) => chunks.push(chunk));
        response.on("end", () => resolve(Buffer.concat(chunks)));
        response.on("error", reject);
      })
      .on("error", reject);
  });
}

async function main() {
  const target = platformTarget();
  const packageName = `deja-v${VERSION}-${target}`;
  const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${packageName}.tar.gz`;

  console.log(`deja-install: downloading Deja v${VERSION} (${target})...`);
  const archive = await download(url).catch((error) =>
    fail(`download failed: ${error.message}`)
  );

  const stage = fs.mkdtempSync(path.join(os.tmpdir(), "deja-"));
  const archivePath = path.join(stage, `${packageName}.tar.gz`);
  fs.writeFileSync(archivePath, archive);

  execFileSync("tar", ["-xzf", archivePath, "-C", stage]);
  const source = path.join(stage, packageName, "bin", "deja");
  if (!fs.existsSync(source)) fail("archive did not contain the deja binary");

  const destination = path.join(__dirname, "bin", "deja");
  fs.mkdirSync(path.dirname(destination), { recursive: true });
  fs.copyFileSync(source, destination);
  fs.chmodSync(destination, 0o755);
  fs.rmSync(stage, { recursive: true, force: true });

  console.log(
    "deja-install: done. Add these lines to ~/.zshrc:\n" +
      `  export PATH="${path.join(__dirname, "bin")}:$PATH"\n` +
      `  export DEJA_CONFIG="${path.join(__dirname, "deja.json")}"\n` +
      `  source "${path.join(__dirname, "shell", "deja.zsh")}"`
  );
}

main();
