#!/usr/bin/env bash
#
# Assemble the two AgentKit zips exactly as .github/workflows/release.yml does, so a kit can be
# produced locally without a tag:
#
#   dist/AgentKit-<version>-win.zip   kit/ + ff-agent.exe (windows/amd64)
#   dist/AgentKit-<version>-mac.zip   kit/ + ff-agent (universal arm64+amd64, executable)
#
# Each zip's ROOT is the kit folder's contents; the game's scripts/fetch-agent-kit.sh unzips them
# straight into Tools/AgentKit/{win,mac}/ and relies on these exact asset names.
#
# Usage: scripts/build-kit.sh [--strict] [--expect-version <v>] [--out <dir>]
#   --strict            fail (instead of warn) when a kit document is missing — the release mode
#   --expect-version    fail unless kit/VERSION's first line equals this (a leading v is ignored)
#
# Needs macOS for the universal binary (lipo).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

STRICT=0
EXPECT=""
OUT="$ROOT/dist"
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --expect-version) EXPECT="${2#v}"; shift 2 ;;
    --out) OUT="$2"; shift 2 ;;
    -h|--help) sed -n '3,16p' "$0"; exit 0 ;;
    *) echo "ERROR: unknown argument '$1'" >&2; exit 2 ;;
  esac
done

VERSION="$(head -n 1 kit/VERSION | tr -d '[:space:]')"
if [ -z "$VERSION" ]; then
  echo "ERROR: kit/VERSION has no version on its first line" >&2
  exit 1
fi
if [ -n "$EXPECT" ] && [ "$EXPECT" != "$VERSION" ]; then
  echo "ERROR: tag says ${EXPECT} but kit/VERSION says ${VERSION}; bump kit/VERSION first" >&2
  exit 1
fi

for f in .mcp.json VERSION; do
  [ -f "kit/$f" ] || { echo "ERROR: kit/$f is missing" >&2; exit 1; }
done
for f in CLAUDE.md HowToPlay.md commands.md; do
  if [ ! -f "kit/$f" ]; then
    if [ "$STRICT" = 1 ]; then
      echo "ERROR: kit/$f is missing (required for a release)" >&2
      exit 1
    fi
    echo "WARNING: kit/$f is missing; this kit is incomplete" >&2
  fi
done

if ! command -v lipo > /dev/null 2>&1; then
  echo "ERROR: the universal mac binary needs lipo (run this on macOS)" >&2
  exit 1
fi

STAGE="$OUT/stage"
rm -rf "$STAGE"
mkdir -p "$STAGE/win" "$STAGE/mac" "$STAGE/bin"
for platform in win mac; do
  cp -R kit/. "$STAGE/$platform/"
  # Nothing machine-local ships: Finder litter, placeholders, a developer's local Claude settings.
  find "$STAGE/$platform" \( -name .DS_Store -o -name .gitkeep -o -name settings.local.json \) -delete
done

export CGO_ENABLED=0
build() { GOOS="$1" GOARCH="$2" go build -trimpath -ldflags "-s -w" -o "$3" ./cmd/ff-agent; }
build windows amd64 "$STAGE/win/ff-agent.exe"
build darwin arm64 "$STAGE/bin/ff-agent-arm64"
build darwin amd64 "$STAGE/bin/ff-agent-amd64"
lipo -create -output "$STAGE/mac/ff-agent" "$STAGE/bin/ff-agent-arm64" "$STAGE/bin/ff-agent-amd64"
# arm64 macOS refuses to run an unsigned binary; re-sign the fat file ad hoc so both slices carry
# a valid signature after lipo.
if command -v codesign > /dev/null 2>&1; then
  codesign --force --sign - "$STAGE/mac/ff-agent"
fi
chmod 755 "$STAGE/mac/ff-agent"

# Windows has no extensionless executables: name the .exe outright rather than rely on the MCP
# client's spawn adding it.
sed 's#"\./ff-agent"#"./ff-agent.exe"#' kit/.mcp.json > "$STAGE/win/.mcp.json"
grep -q '"./ff-agent.exe"' "$STAGE/win/.mcp.json" || { echo "ERROR: could not point win/.mcp.json at ff-agent.exe" >&2; exit 1; }

for platform in win mac; do
  zipfile="$OUT/AgentKit-${VERSION}-${platform}.zip"
  rm -f "$zipfile"
  (cd "$STAGE/$platform" && zip -qrX "$zipfile" .)
done
rm -rf "$STAGE/bin"

echo "AgentKit ${VERSION}:"
echo "  $OUT/AgentKit-${VERSION}-win.zip"
echo "  $OUT/AgentKit-${VERSION}-mac.zip ($(lipo -archs "$STAGE/mac/ff-agent"))"
