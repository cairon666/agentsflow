#!/usr/bin/env node
"use strict";

const { spawnSync } = require("node:child_process");
const path = require("node:path");

const packageByPlatform = {
  "darwin arm64": "agentsflow-darwin-arm64",
  "darwin x64": "agentsflow-darwin-x64",
  "linux arm64": "agentsflow-linux-arm64",
  "linux x64": "agentsflow-linux-x64",
  "win32 arm64": "agentsflow-win32-arm64",
  "win32 x64": "agentsflow-win32-x64"
};

const key = process.platform + " " + process.arch;
const packageName = packageByPlatform[key];

if (!packageName) {
  console.error("agentsflow does not provide a prebuilt binary for " + key + ".");
  process.exit(1);
}

let binary;
try {
  const packageJson = require.resolve(packageName + "/package.json");
  const executable = process.platform === "win32" ? "agentsflow.exe" : "agentsflow";
  binary = path.join(path.dirname(packageJson), "bin", executable);
} catch (error) {
  console.error("The optional dependency " + packageName + " was not installed.");
  console.error("Reinstall agentsflow without --omit=optional or --no-optional.");
  process.exit(1);
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}

if (result.signal) {
  process.kill(process.pid, result.signal);
}

process.exit(result.status ?? 1);
