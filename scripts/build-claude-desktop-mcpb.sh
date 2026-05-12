#!/usr/bin/env bash
set -euo pipefail

version="${1:-$(git describe --tags --always --dirty)}"
version="${version#v}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="$root/dist"
workdir="$(mktemp -d)"

cleanup() {
  rm -rf "$workdir"
}
trap cleanup EXIT

if [ "$(uname -s)" != "Darwin" ]; then
  printf 'error: build-claude-desktop-mcpb.sh must run on macOS so lipo can create a universal macOS binary\n' >&2
  exit 1
fi

need_command() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'error: %s is required\n' "$1" >&2
    exit 1
  }
}

need_command go
need_command lipo
need_command zip

bundle="$workdir/paynow-mcp"
mkdir -p "$bundle/server" "$dist"

(
  cd "$root"
  CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${version}" \
    -o "$workdir/paynow-mcp-darwin-amd64" \
    ./cmd/paynow-mcp
  CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${version}" \
    -o "$workdir/paynow-mcp-darwin-arm64" \
    ./cmd/paynow-mcp
  CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${version}" \
    -o "$bundle/server/paynow-mcp.exe" \
    ./cmd/paynow-mcp
)

lipo -create \
  "$workdir/paynow-mcp-darwin-amd64" \
  "$workdir/paynow-mcp-darwin-arm64" \
  -output "$bundle/server/paynow-mcp"
chmod 0755 "$bundle/server/paynow-mcp"

sed "s/\"version\": \"0.0.0\"/\"version\": \"${version}\"/" \
  "$root/packaging/mcpb/manifest.json" > "$bundle/manifest.json"
cp "$root/LICENSE" "$bundle/LICENSE"

(
  cd "$bundle"
  zip -qr "$dist/paynow-mcp_claude-desktop.mcpb" .
)
