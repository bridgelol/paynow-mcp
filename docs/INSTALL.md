# Installation

This server is a local stdio MCP server. The easiest setup depends on the client.

## Claude Desktop

Download `paynow-mcp_claude-desktop.mcpb` from the latest release:

<https://github.com/bridgelol/paynow-mcp/releases/latest>

Open the `.mcpb` file with Claude Desktop and enter your PayNow Management API key when prompted. The bundle includes a universal macOS binary and a Windows amd64 binary, so Claude Desktop users do not need Go, npm, or manual JSON editing.

Claude documents MCPB as a single-click local MCP install format for Claude Desktop.

## Claude Code

```sh
export PAYNOW_API_KEY="APIKey your_token_here"
curl -fsSL https://raw.githubusercontent.com/bridgelol/paynow-mcp/main/scripts/install-claude-code.sh | bash
```

The installer downloads the latest release binary into `~/.local/bin`, then registers it with:

```sh
claude mcp add --scope user --transport stdio -e PAYNOW_API_KEY="$PAYNOW_API_KEY" paynow -- /path/to/paynow-mcp
```

Set `CLAUDE_MCP_SCOPE=local` or `CLAUDE_MCP_SCOPE=project` before running the installer if you do not want user-level configuration.

## Codex

```sh
export PAYNOW_API_KEY="APIKey your_token_here"
curl -fsSL https://raw.githubusercontent.com/bridgelol/paynow-mcp/main/scripts/install-codex.sh | bash
```

The installer downloads the latest release binary into `~/.local/bin`, then registers it with:

```sh
codex mcp add --env PAYNOW_API_KEY="$PAYNOW_API_KEY" paynow -- /path/to/paynow-mcp
```

Codex stores MCP server entries in `~/.codex/config.toml`.

## Manual Binary Install

Download the right binary archive from the latest release:

<https://github.com/bridgelol/paynow-mcp/releases/latest>

Then configure any stdio MCP client with:

```json
{
  "mcpServers": {
    "paynow": {
      "command": "/absolute/path/to/paynow-mcp",
      "env": {
        "PAYNOW_API_KEY": "APIKey your_token_here"
      }
    }
  }
}
```

## Go Install

Developers with Go installed can still use:

```sh
go install github.com/bridgelol/paynow-mcp/cmd/paynow-mcp@latest
```

## Notes

The Claude Code and Codex installers store the PayNow API key in the client MCP configuration. Claude Desktop MCPB marks the API key as sensitive, so Claude Desktop stores it through its extension configuration flow.

Optional environment variables:

- `PAYNOW_BASE_URL`
- `PAYNOW_STORE_ID`
- `PAYNOW_PROFILES`
- `PAYNOW_DEFAULT_PROFILE`
- `PAYNOW_INCLUDE_APIS`
- `PAYNOW_TIMEOUT_SECONDS`
- `PAYNOW_MCP_VERSION`, for example `v0.2.0`
- `PAYNOW_MCP_INSTALL_DIR`, default `~/.local/bin`
- `PAYNOW_MCP_NAME`, default `paynow`

Multi-profile example:

```sh
export PAYNOW_PROFILES='{
  "prod": {"api_key":"APIKey prod_token_here","store_id":"prod_store_id"},
  "staging": {"api_key":"APIKey staging_token_here","store_id":"123"}
}'
export PAYNOW_DEFAULT_PROFILE=prod
```

For Codex and Claude Code installers, `PAYNOW_PROFILES` is not embedded automatically because it is often large and sensitive. Add it manually to your client config or register separate MCP server entries per store.
