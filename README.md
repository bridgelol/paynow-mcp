# PayNow MCP Server

A Go MCP server for managing a PayNow store through the PayNow Management API.

This project intentionally has no npm dependency and no third-party Go runtime dependency. It speaks MCP over stdio with the Go standard library and calls PayNow with `net/http`.

## Features

- Store management: list, get, create, update, and delete stores.
- Products, coupons, customers, orders, payments, subscriptions, webhooks, and game server tools.
- Checkout session creation.
- Guarded destructive operations: deletes, refunds, subscription cancellation, and game server token resets require `confirm=true`.
- `paynow_api_request` for any PayNow Management API path that does not have a dedicated tool yet.

## Install

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
| `PAYNOW_TIMEOUT_SECONDS` | `30` | HTTP request timeout. |

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
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"manual","version":"0"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | go run ./cmd/paynow-mcp
```

## Sources

- PayNow docs: <https://docs.paynow.gg/>
- PayNow OpenAPI specifications: <https://github.com/paynow-gg/openapi>
- MCP specification: <https://modelcontextprotocol.io/specification/2025-11-25/schema>
