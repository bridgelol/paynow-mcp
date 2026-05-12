#!/usr/bin/env bash
set -euo pipefail

repo="${PAYNOW_MCP_REPO:-bridgelol/paynow-mcp}"
version="${PAYNOW_MCP_VERSION:-latest}"
server_name="${PAYNOW_MCP_NAME:-paynow}"
install_dir="${PAYNOW_MCP_INSTALL_DIR:-$HOME/.local/bin}"
scope="${CLAUDE_MCP_SCOPE:-user}"

fail() {
  printf 'error: %s\n' "$1" >&2
  exit 1
}

need_command() {
  command -v "$1" >/dev/null 2>&1 || fail "$1 is required"
}

detect_asset() {
  local os arch ext

  case "$(uname -s)" in
    Darwin) os="darwin" ;;
    Linux) os="linux" ;;
    MINGW*|MSYS*|CYGWIN*) os="windows" ;;
    *) fail "unsupported operating system: $(uname -s)" ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
  esac

  if [ "$os" = "windows" ]; then
    ext="zip"
  else
    ext="tar.gz"
  fi

  printf 'paynow-mcp_%s_%s.%s' "$os" "$arch" "$ext"
}

download_binary() {
  need_command curl

  local asset url tmp binary
  asset="$(detect_asset)"
  mkdir -p "$install_dir"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT

  if [ "$version" = "latest" ]; then
    url="https://github.com/${repo}/releases/latest/download/${asset}"
  else
    url="https://github.com/${repo}/releases/download/${version}/${asset}"
  fi

  printf 'Downloading %s...\n' "$url" >&2
  curl -fsSL "$url" -o "$tmp/$asset"

  case "$asset" in
    *.tar.gz)
      tar -xzf "$tmp/$asset" -C "$tmp"
      binary="$tmp/paynow-mcp/paynow-mcp"
      ;;
    *.zip)
      need_command unzip
      unzip -q "$tmp/$asset" -d "$tmp"
      binary="$tmp/paynow-mcp/paynow-mcp.exe"
      ;;
    *)
      fail "unknown asset format: $asset"
      ;;
  esac

  install -m 0755 "$binary" "$install_dir/$(basename "$binary")"
  printf '%s\n' "$install_dir/$(basename "$binary")"
}

[ -n "${PAYNOW_API_KEY:-}" ] || fail "set PAYNOW_API_KEY first"
need_command claude

if [ -n "${PAYNOW_MCP_BIN:-}" ]; then
  binary="$PAYNOW_MCP_BIN"
else
  binary="$(download_binary)"
fi

[ -x "$binary" ] || fail "PayNow MCP binary is not executable: $binary"

env_args=(-e "PAYNOW_API_KEY=${PAYNOW_API_KEY}")
[ -z "${PAYNOW_BASE_URL:-}" ] || env_args+=(-e "PAYNOW_BASE_URL=${PAYNOW_BASE_URL}")
[ -z "${PAYNOW_STORE_ID:-}" ] || env_args+=(-e "PAYNOW_STORE_ID=${PAYNOW_STORE_ID}")
[ -z "${PAYNOW_AUTH_PREFIX:-}" ] || env_args+=(-e "PAYNOW_AUTH_PREFIX=${PAYNOW_AUTH_PREFIX}")
[ -z "${PAYNOW_AUTH_KIND:-}" ] || env_args+=(-e "PAYNOW_AUTH_KIND=${PAYNOW_AUTH_KIND}")
[ -z "${PAYNOW_INCLUDE_APIS:-}" ] || env_args+=(-e "PAYNOW_INCLUDE_APIS=${PAYNOW_INCLUDE_APIS}")
[ -z "${PAYNOW_OPENAPI_TOOLS:-}" ] || env_args+=(-e "PAYNOW_OPENAPI_TOOLS=${PAYNOW_OPENAPI_TOOLS}")
[ -z "${PAYNOW_TIMEOUT_SECONDS:-}" ] || env_args+=(-e "PAYNOW_TIMEOUT_SECONDS=${PAYNOW_TIMEOUT_SECONDS}")

claude mcp remove --scope "$scope" "$server_name" >/dev/null 2>&1 || true
claude mcp add --scope "$scope" --transport stdio "${env_args[@]}" "$server_name" -- "$binary"

printf 'Installed PayNow MCP as Claude Code server "%s" in %s scope. Restart Claude Code sessions to load it.\n' "$server_name" "$scope"
