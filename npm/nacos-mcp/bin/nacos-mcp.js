#!/usr/bin/env node

const { spawnSync } = require("child_process");
const { existsSync, chmodSync, readFileSync } = require("fs");
const path = require("path");

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "win32",
};

const ARCH_MAP = {
  arm64: "arm64",
  x64: "amd64",
};

function getBinaryPath() {
  const platform = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];

  if (!platform || !arch) {
    console.error(
      `nacos-mcp: unsupported platform ${process.platform}/${process.arch}\n` +
        `Supported: darwin/linux/win32 x arm64/x64`
    );
    process.exit(1);
  }

  const pkgName = `@vinkugor/nacos-mcp-${platform}-${arch}`;
  const binaryName = process.platform === "win32" ? "nacos-mcp.exe" : "nacos-mcp";

  let pkgDir;
  try {
    const pkgJson = require.resolve(`${pkgName}/package.json`);
    pkgDir = path.dirname(pkgJson);
  } catch {
    const { version } = JSON.parse(
      readFileSync(path.join(__dirname, "..", "package.json"), "utf8")
    );
    console.error(
      `nacos-mcp: no prebuilt binary found for ${process.platform}/${process.arch}\n\n` +
        `Package ${pkgName} is not installed. This usually means:\n` +
        `  - you installed with --ignore-scripts or --no-optional\n` +
        `  - your package manager suppressed optional deps\n\n` +
        `Workarounds:\n` +
        `  - reinstall without --no-optional\n` +
        `  - download manually from:\n` +
        `    https://github.com/Vinkugor/nacos-mcp/releases/tag/v${version}`
    );
    process.exit(1);
  }

  return path.join(pkgDir, "bin", binaryName);
}

function main() {
  const binary = getBinaryPath();

  if (!existsSync(binary)) {
    console.error(`nacos-mcp: binary not found at ${binary}`);
    process.exit(1);
  }

  if (process.platform !== "win32") {
    try {
      chmodSync(binary, 0o755);
    } catch {
      // chmod failures are non-fatal; exec will surface the real error
    }
  }

  // MCP stdio transport requires direct stdin/stdout passthrough; do not switch to "pipe".
  const result = spawnSync(binary, process.argv.slice(2), {
    stdio: "inherit",
    env: process.env,
  });

  if (result.error) {
    console.error(`nacos-mcp: failed to start binary: ${result.error.message}`);
    process.exit(1);
  }

  if (result.signal) {
    const sigNum = require("os").constants.signals[result.signal] || 1;
    process.exit(128 + sigNum);
  }

  process.exit(result.status ?? 1);
}

main();
