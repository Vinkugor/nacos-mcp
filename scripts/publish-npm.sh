#!/usr/bin/env bash
# Assemble and publish nacos-mcp npm packages.
#
# Usage:
#   VERSION=0.1.0 scripts/publish-npm.sh              # full publish
#   VERSION=0.1.0 scripts/publish-npm.sh --dry-run    # local assembly, no publish
#
# Required env:
#   VERSION        — version string without leading 'v' (e.g. "0.1.0")
#
# Optional env:
#   GH_REPO        — GitHub repo slug (default: Vinkugor/nacos-mcp)
#   DIST_DIR       — where GH release artifacts are downloaded (default: ./dist)
#   NPM_DIR        — where final .tgz files go in dry-run (default: ./dist/npm)
#   SKIP_DOWNLOAD  — set to 1 to skip `gh release download` (expects DIST_DIR/ prepopulated)
#   SKIP_UPX       — set to 1 to skip UPX compression (for local macOS where UPX may be missing)

set -euo pipefail

DRY_RUN=0
if [[ "${1:-}" == "--dry-run" ]]; then
  DRY_RUN=1
fi

: "${VERSION:?VERSION env var required (e.g. VERSION=0.1.0)}"

GH_REPO="${GH_REPO:-Vinkugor/nacos-mcp}"
DIST_DIR="${DIST_DIR:-$(pwd)/dist}"
NPM_DIR="${NPM_DIR:-$DIST_DIR/npm}"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Package name → GoReleaser archive suffix mapping
# npm pkg suffix:   goreleaser archive suffix
# darwin-arm64      darwin_arm64
# darwin-amd64      darwin_amd64
# linux-arm64       linux_arm64
# linux-amd64       linux_amd64
# win32-amd64       windows_amd64
# win32-arm64       windows_arm64
#
# Mapping rule: replace "-" with "_", and "win32" with "windows".
# (Avoids `declare -A` so this works on macOS bash 3.2.)
npm_to_gr() {
  local s="${1//-/_}"
  echo "${s/win32/windows}"
}

NPM_SUBPKGS=(darwin-arm64 darwin-amd64 linux-arm64 linux-amd64 win32-amd64 win32-arm64)

echo ">>> VERSION=$VERSION"
echo ">>> GH_REPO=$GH_REPO"
echo ">>> DIST_DIR=$DIST_DIR"
echo ">>> NPM_DIR=$NPM_DIR"
echo ">>> DRY_RUN=$DRY_RUN"

mkdir -p "$DIST_DIR"

# ---------- 1. Download release artifacts ----------
if [[ "${SKIP_DOWNLOAD:-0}" != "1" ]]; then
  echo ">>> Downloading release artifacts from v${VERSION}"
  cd "$DIST_DIR"
  gh release download "v${VERSION}" --repo "$GH_REPO" --clobber
  cd "$REPO_ROOT"
fi

# ---------- 2. Verify checksums ----------
echo ">>> Verifying checksums"
cd "$DIST_DIR"
if [[ ! -f checksums.txt ]]; then
  echo "ERROR: checksums.txt not found in $DIST_DIR" >&2
  exit 1
fi
sha256sum -c checksums.txt
cd "$REPO_ROOT"

# ---------- 3. Extract archives into subpackage bin/ dirs ----------
echo ">>> Extracting archives into npm subpackages"
for subpkg in "${NPM_SUBPKGS[@]}"; do
  gr_suffix="$(npm_to_gr "$subpkg")"
  npm_pkg_dir="$REPO_ROOT/npm/nacos-mcp-${subpkg}"
  bin_dir="$npm_pkg_dir/bin"

  # Clean bin/ but keep directory (rm -f "$bin_dir"/* doesn't match dotfiles)
  rm -rf "$bin_dir"
  mkdir -p "$bin_dir"

  if [[ "$subpkg" == win32-* ]]; then
    archive="$DIST_DIR/nacos-mcp_${VERSION}_${gr_suffix}.zip"
    echo "  - extracting $(basename "$archive") -> $bin_dir/nacos-mcp.exe"
    unzip -j -o "$archive" "nacos-mcp.exe" -d "$bin_dir"
  else
    archive="$DIST_DIR/nacos-mcp_${VERSION}_${gr_suffix}.tar.gz"
    echo "  - extracting $(basename "$archive") -> $bin_dir/nacos-mcp"
    tar -xzf "$archive" -C "$bin_dir" nacos-mcp
  fi
done

# ---------- 4. UPX compression on Linux binaries ----------
if [[ "${SKIP_UPX:-0}" != "1" ]]; then
  if command -v upx >/dev/null 2>&1; then
    echo ">>> Compressing Linux binaries with UPX"
    for subpkg in linux-amd64 linux-arm64; do
      bin="$REPO_ROOT/npm/nacos-mcp-${subpkg}/bin/nacos-mcp"
      echo "  - upx $bin"
      upx --best --lzma "$bin"
    done
  else
    echo "WARNING: upx not installed — skipping Linux binary compression" >&2
  fi
fi

# ---------- 5. Set executable permissions ----------
echo ">>> Setting executable permissions"
for subpkg in "${NPM_SUBPKGS[@]}"; do
  if [[ "$subpkg" != win32-* ]]; then
    chmod +x "$REPO_ROOT/npm/nacos-mcp-${subpkg}/bin/nacos-mcp"
  fi
done

# ---------- 6. Inject version into all 7 package.json ----------
echo ">>> Injecting version=$VERSION into 7 package.json files"
for subpkg in "${NPM_SUBPKGS[@]}"; do
  pkg_dir="$REPO_ROOT/npm/nacos-mcp-${subpkg}"
  (cd "$pkg_dir" && npm version "$VERSION" --no-git-tag-version --allow-same-version >/dev/null)
done

# Main package: update version + all optionalDependencies pins
MAIN_PKG="$REPO_ROOT/npm/nacos-mcp/package.json"
node - "$MAIN_PKG" "$VERSION" <<'NODE'
const fs = require("fs");
const [, , file, version] = process.argv;
const pkg = JSON.parse(fs.readFileSync(file, "utf8"));
pkg.version = version;
for (const dep of Object.keys(pkg.optionalDependencies || {})) {
  pkg.optionalDependencies[dep] = version;
}
fs.writeFileSync(file, JSON.stringify(pkg, null, 2) + "\n");
NODE

# ---------- 7. Dry-run: npm pack each, stop ----------
if [[ "$DRY_RUN" == "1" ]]; then
  echo ">>> DRY RUN: packing all 7 packages into $NPM_DIR"
  mkdir -p "$NPM_DIR"
  rm -f "$NPM_DIR"/*.tgz
  for subpkg in "${NPM_SUBPKGS[@]}"; do
    (cd "$REPO_ROOT/npm/nacos-mcp-${subpkg}" && npm pack --pack-destination "$NPM_DIR" >/dev/null)
  done
  (cd "$REPO_ROOT/npm/nacos-mcp" && npm pack --pack-destination "$NPM_DIR" >/dev/null)
  echo ">>> DRY RUN complete. Packages in $NPM_DIR:"
  ls -la "$NPM_DIR"/*.tgz
  exit 0
fi

# ---------- 8. Smoke test: install main + current-platform subpkg from local tarballs ----------
echo ">>> Smoke test: packing main + current-platform subpkg and running --version"
CURRENT_PLAT=""
case "$(uname -s)" in
  Darwin) CURRENT_PLAT="darwin" ;;
  Linux)  CURRENT_PLAT="linux" ;;
  *)      echo "ERROR: smoke test only runs on darwin/linux runners" >&2; exit 1 ;;
esac
CURRENT_ARCH=""
case "$(uname -m)" in
  x86_64)  CURRENT_ARCH="amd64" ;;
  arm64|aarch64) CURRENT_ARCH="arm64" ;;
  *)       echo "ERROR: unsupported arch $(uname -m)" >&2; exit 1 ;;
esac

SMOKE_DIR="$(mktemp -d)"
trap 'rm -rf "$SMOKE_DIR"' EXIT

(cd "$REPO_ROOT/npm/nacos-mcp" && npm pack --pack-destination "$SMOKE_DIR" >/dev/null)
(cd "$REPO_ROOT/npm/nacos-mcp-${CURRENT_PLAT}-${CURRENT_ARCH}" && npm pack --pack-destination "$SMOKE_DIR" >/dev/null)

MAIN_TGZ=$(ls "$SMOKE_DIR"/vinkugor-nacos-mcp-*.tgz | grep -v -- "-${CURRENT_PLAT}-" | head -1)
SUB_TGZ=$(ls "$SMOKE_DIR"/vinkugor-nacos-mcp-${CURRENT_PLAT}-${CURRENT_ARCH}-*.tgz | head -1)

cd "$SMOKE_DIR"
npm init -y >/dev/null
npm install --no-save "$SUB_TGZ" "$MAIN_TGZ" >/dev/null

OUTPUT=$(./node_modules/.bin/nacos-mcp --version)
if [[ "$OUTPUT" != "$VERSION" ]]; then
  echo "ERROR: smoke test expected '$VERSION', got '$OUTPUT'" >&2
  exit 1
fi
echo ">>> Smoke test passed: --version -> $OUTPUT"
cd "$REPO_ROOT"

# ---------- 9. Verify npm auth ----------
echo ">>> Verifying npm auth"
npm whoami

# ---------- 10. Publish subpackages first, then main ----------
echo ">>> Publishing 6 platform subpackages"
for subpkg in "${NPM_SUBPKGS[@]}"; do
  echo "  - publishing @vinkugor/nacos-mcp-${subpkg}@${VERSION}"
  (cd "$REPO_ROOT/npm/nacos-mcp-${subpkg}" && npm publish --access public --provenance)
done

echo ">>> Publishing main package @vinkugor/nacos-mcp@${VERSION}"
(cd "$REPO_ROOT/npm/nacos-mcp" && npm publish --access public --provenance)

echo ">>> Done. Published @vinkugor/nacos-mcp@${VERSION} (+6 platform packages)."
