const test = require("node:test");
const assert = require("node:assert");
const { spawnSync } = require("node:child_process");
const path = require("node:path");
const fs = require("node:fs");
const os = require("node:os");

const SHIM = path.join(__dirname, "..", "bin", "nacos-mcp.js");

function runShim(env = {}, args = []) {
  return spawnSync(process.execPath, [SHIM, ...args], {
    env: { ...process.env, ...env },
    encoding: "utf8",
  });
}

test("shim exits 1 with clear error when no platform subpackage is installed", () => {
  // 默认情况下测试目录没有任何 @vinkugor/nacos-mcp-* 子包
  const result = runShim();
  assert.strictEqual(result.status, 1, "expected exit code 1");
  assert.match(
    result.stderr,
    /no prebuilt binary found/,
    "expected 'no prebuilt binary found' in stderr"
  );
  assert.match(
    result.stderr,
    /github\.com\/Vinkugor\/nacos-mcp\/releases/,
    "expected release URL hint in stderr"
  );
});

test("shim execs the platform binary when installed (happy path)", () => {
  const platform = process.platform;
  const arch = process.arch;
  const platformKey = { darwin: "darwin", linux: "linux", win32: "win32" }[
    platform
  ];
  const archKey = { arm64: "arm64", x64: "amd64" }[arch];

  if (!platformKey || !archKey) {
    // 未知平台，跳过 happy-path（CI 覆盖主流平台即可）
    return;
  }

  const pkgName = `@vinkugor/nacos-mcp-${platformKey}-${archKey}`;
  const fakeDir = fs.mkdtempSync(path.join(os.tmpdir(), "shim-test-"));
  let result;
  try {
    // 在 fakeDir 下构造 node_modules/@vinkugor/nacos-mcp-<plat>-<arch>/
    const pkgDir = path.join(
      fakeDir,
      "node_modules",
      "@vinkugor",
      `nacos-mcp-${platformKey}-${archKey}`
    );
    const binDir = path.join(pkgDir, "bin");
    fs.mkdirSync(binDir, { recursive: true });

    // 写入一个可执行脚本当"二进制"
    const binName = platform === "win32" ? "nacos-mcp.exe" : "nacos-mcp";
    const binPath = path.join(binDir, binName);
    const script =
      platform === "win32"
        ? `@echo off\r\necho mock-binary-ran\r\nexit /b 0\r\n`
        : `#!/bin/sh\necho mock-binary-ran\nexit 0\n`;
    fs.writeFileSync(binPath, script);
    if (platform !== "win32") fs.chmodSync(binPath, 0o755);

    // 必须有 package.json，require.resolve 才能定位
    fs.writeFileSync(
      path.join(pkgDir, "package.json"),
      JSON.stringify({ name: pkgName, version: "0.0.0" })
    );

    result = spawnSync(process.execPath, [SHIM], {
      cwd: fakeDir,
      env: { ...process.env, NODE_PATH: path.join(fakeDir, "node_modules") },
      encoding: "utf8",
    });
  } finally {
    fs.rmSync(fakeDir, { recursive: true, force: true });
  }

  assert.strictEqual(result.status, 0, `expected 0, got ${result.status} (stderr: ${result.stderr})`);
  assert.match(result.stdout, /mock-binary-ran/);
});

test("shim passes argv through to the binary", () => {
  const platform = process.platform;
  const arch = process.arch;
  const platformKey = { darwin: "darwin", linux: "linux", win32: "win32" }[
    platform
  ];
  const archKey = { arm64: "arm64", x64: "amd64" }[arch];
  if (!platformKey || !archKey) return;

  const fakeDir = fs.mkdtempSync(path.join(os.tmpdir(), "shim-test-"));
  let result;
  try {
    const pkgDir = path.join(
      fakeDir,
      "node_modules",
      "@vinkugor",
      `nacos-mcp-${platformKey}-${archKey}`
    );
    const binDir = path.join(pkgDir, "bin");
    fs.mkdirSync(binDir, { recursive: true });

    const binName = platform === "win32" ? "nacos-mcp.exe" : "nacos-mcp";
    const binPath = path.join(binDir, binName);
    const script =
      platform === "win32"
        ? `@echo off\r\necho arg1=%1 arg2=%2\r\nexit /b 0\r\n`
        : `#!/bin/sh\necho "arg1=$1 arg2=$2"\nexit 0\n`;
    fs.writeFileSync(binPath, script);
    if (platform !== "win32") fs.chmodSync(binPath, 0o755);

    fs.writeFileSync(
      path.join(pkgDir, "package.json"),
      JSON.stringify({
        name: `@vinkugor/nacos-mcp-${platformKey}-${archKey}`,
        version: "0.0.0",
      })
    );

    result = spawnSync(
      process.execPath,
      [SHIM, "--version", "extra"],
      {
        cwd: fakeDir,
        env: { ...process.env, NODE_PATH: path.join(fakeDir, "node_modules") },
        encoding: "utf8",
      }
    );
  } finally {
    fs.rmSync(fakeDir, { recursive: true, force: true });
  }

  assert.strictEqual(result.status, 0);
  assert.match(result.stdout, /arg1=--version arg2=extra/);
});
