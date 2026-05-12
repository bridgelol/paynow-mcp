#!/usr/bin/env bash
set -euo pipefail

version="${1:-$(git describe --tags --always --dirty)}"
version="${version#v}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="$root/dist"

checksum() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1"
  else
    shasum -a 256 "$1"
  fi
}

rm -rf "$dist"
mkdir -p "$dist"

targets=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

for target in "${targets[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  package_root="$dist/package-${goos}-${goarch}"
  package_dir="$package_root/paynow-mcp"
  binary_name="paynow-mcp"
  archive_base="paynow-mcp_${goos}_${goarch}"

  if [ "$goos" = "windows" ]; then
    binary_name="paynow-mcp.exe"
  fi

  mkdir -p "$package_dir"
  (
    cd "$root"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
      -trimpath \
      -ldflags "-s -w -X main.version=${version}" \
      -o "$package_dir/$binary_name" \
      ./cmd/paynow-mcp
  )
  cp "$root/README.md" "$root/LICENSE" "$package_dir/"

  if [ "$goos" = "windows" ]; then
    (
      cd "$package_root"
      zip -qr "$dist/${archive_base}.zip" paynow-mcp
    )
  else
    tar -C "$package_root" -czf "$dist/${archive_base}.tar.gz" paynow-mcp
  fi
done

rm -rf "$dist"/package-*
(
  cd "$dist"
  for file in paynow-mcp_*; do
    checksum "$file"
  done > checksums.txt
)
