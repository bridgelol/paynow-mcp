# PayNow MCP Server

A Go MCP server for managing a PayNow store through the PayNow Management API.

This project intentionally has no npm dependency and no third-party Go runtime dependency. It speaks MCP over stdio with the Go standard library and calls PayNow with `net/http`.

## Features

- Store management: list, get, create, update, and delete stores.
- Products, coupons, customers, orders, payments, subscriptions, webhooks, and game server tools.
- Checkout session creation.
- Multi-profile support for multiple stores or API keys in one MCP server.
- Optional `PAYNOW_STORE_ID` or per-profile `store_id` so store-scoped tools can omit `store_id`.
- Generated tools from PayNow's bundled OpenAPI specs. Management API tools are enabled by default; Storefront and Game Server APIs are opt-in.
- Guarded destructive operations: deletes, refunds, subscription cancellation, and game server token resets require `confirm=true`.
- `paynow_api_request` for any PayNow Management API path that does not have a dedicated tool yet.

## Install

Fast client setup:

- Claude Desktop: download `paynow-mcp_claude-desktop.mcpb` from the [latest release](https://github.com/bridgelol/paynow-mcp/releases/latest) and open it.
- Claude Code:

  ```sh
  export PAYNOW_API_KEY="APIKey your_token_here"
  curl -fsSL https://raw.githubusercontent.com/bridgelol/paynow-mcp/main/scripts/install-claude-code.sh | bash
  ```

- Codex:

  ```sh
  export PAYNOW_API_KEY="APIKey your_token_here"
  curl -fsSL https://raw.githubusercontent.com/bridgelol/paynow-mcp/main/scripts/install-codex.sh | bash
  ```

See [docs/INSTALL.md](docs/INSTALL.md) for manual config snippets and installer options.

Developer install:

```sh
go install github.com/bridgelol/paynow-mcp/cmd/paynow-mcp@latest
```

Or run from a clone:

```sh
git clone https://github.com/bridgelol/paynow-mcp.git
cd paynow-mcp
go run ./cmd/paynow-mcp
```

## Configuration

Create a PayNow Management API key in the PayNow dashboard, then set:

```sh
export PAYNOW_API_KEY="APIKey your_token_here"
```

You may also set `PAYNOW_API_KEY` to just the raw token. In that case this server sends it as `Authorization: APIKey <token>`.

Optional environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `PAYNOW_BASE_URL` | `https://api.paynow.gg` | PayNow API base URL. |
| `PAYNOW_AUTH_PREFIX` | `APIKey` | Authorization prefix used when `PAYNOW_API_KEY` is a raw token. |
| `PAYNOW_AUTH_KIND` | `apikey` | Convenience auth kind for raw tokens: `apikey`, `customer`, or `gameserver`. |
| `PAYNOW_STORE_ID` | none | Default store ID for store-scoped tools. |
| `PAYNOW_PROFILES` | none | JSON object of named profiles for multiple stores or API keys. |
| `PAYNOW_DEFAULT_PROFILE` | `default` or first profile | Default profile name. |
| `PAYNOW_INCLUDE_APIS` | `management` | Comma-separated OpenAPI toolsets: `management`, `storefront`, `gameserver`; use `none` to disable generated tools. |
| `PAYNOW_OPENAPI_TOOLS` | enabled | Set to `false` to disable generated OpenAPI tools. |
| `PAYNOW_TIMEOUT_SECONDS` | `30` | HTTP request timeout. |

Multi-profile example:

```sh
export PAYNOW_PROFILES='{
  "prod": {
    "api_key": "APIKey prod_token_here",
    "store_id": "prod_store_id"
  },
  "staging": {
    "api_key": "APIKey staging_token_here",
    "store_id": "123"
  }
}'
export PAYNOW_DEFAULT_PROFILE=prod
```

Then pass `"profile": "staging"` to any tool to target that profile.

## MCP Client Setup

Example MCP server config:

```json
{
  "mcpServers": {
    "paynow": {
      "command": "paynow-mcp",
      "env": {
        "PAYNOW_API_KEY": "APIKey your_token_here"
      }
    }
  }
}
```

If `paynow-mcp` is not on your `PATH`, use the full path from:

```sh
which paynow-mcp
```

## Tools

Core:

- `paynow_help`
- `paynow_api_request`
- `paynow_list_stores`
- `paynow_get_store`
- `paynow_create_store`
- `paynow_update_store`
- `paynow_delete_store`

Store resources:

- Products: `paynow_list_products`, `paynow_get_product`, `paynow_create_product`, `paynow_update_product`, `paynow_delete_product`
- Coupons: `paynow_list_coupons`, `paynow_get_coupon`, `paynow_create_coupon`, `paynow_update_coupon`, `paynow_delete_coupon`
- Customers: `paynow_list_customers`, `paynow_get_customer`, `paynow_lookup_customer`, `paynow_create_customer`, `paynow_update_customer`, `paynow_bulk_create_customers`, `paynow_create_customer_token`, `paynow_invalidate_customer_tokens`
- Orders and payments: `paynow_list_orders`, `paynow_get_order`, `paynow_refund_order`, `paynow_list_payments`, `paynow_get_payment`
- Subscriptions: `paynow_list_subscriptions`, `paynow_get_subscription`, `paynow_cancel_subscription`, `paynow_preview_subscription_change`, `paynow_change_subscription`
- Webhooks: `paynow_list_webhooks`, `paynow_create_webhook`, `paynow_update_webhook`, `paynow_delete_webhook`, `paynow_resend_webhook`, `paynow_list_webhook_history`, `paynow_list_webhook_variables`
- Game servers: `paynow_list_game_servers`, `paynow_get_game_server`, `paynow_create_game_server`, `paynow_update_game_server`, `paynow_delete_game_server`, `paynow_reset_game_server_token`
- Checkout: `paynow_create_checkout_session`

Use `paynow_api_request` for other PayNow Management API endpoints:

```json
{
  "method": "GET",
  "path": "/v1/stores/{storeId}/sales",
  "query": {
    "limit": 50
  }
}
```

For DELETE:

```json
{
  "method": "DELETE",
  "path": "/v1/stores/{storeId}/products/{productId}",
  "confirm": true
}
```

## Development

```sh
go test ./...
go run ./cmd/paynow-mcp
```

Manual MCP smoke test:

```sh
export PAYNOW_API_KEY="APIKey your_token_here"
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"manual","version":"0"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | go run ./cmd/paynow-mcp
```

## Sources

- PayNow docs: <https://docs.paynow.gg/>
- PayNow OpenAPI specifications: <https://github.com/paynow-gg/openapi>
- MCP specification: <https://modelcontextprotocol.io/specification/2025-11-25/schema>
- Claude MCPB docs: <https://claude.com/docs/connectors/building/mcpb>
- Codex MCP docs: <https://developers.openai.com/codex/cli/reference#codex-mcp>
