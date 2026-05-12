package paynow

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/bridgelol/paynow-mcp/internal/mcp"
)

type toolBuilder struct {
	client *Client
}

func Tools(client *Client) []mcp.Tool {
	builder := toolBuilder{client: client}

	return []mcp.Tool{
		builder.help(),
		builder.apiRequest(),
		builder.listStores(),
		builder.getStore(),
		builder.createStore(),
		builder.updateStore(),
		builder.deleteStore(),
		builder.listProducts(),
		builder.getProduct(),
		builder.createProduct(),
		builder.updateProduct(),
		builder.deleteProduct(),
		builder.createCheckoutSession(),
		builder.listCoupons(),
		builder.getCoupon(),
		builder.createCoupon(),
		builder.updateCoupon(),
		builder.deleteCoupon(),
		builder.listCustomers(),
		builder.getCustomer(),
		builder.lookupCustomer(),
		builder.createCustomer(),
		builder.updateCustomer(),
		builder.bulkCreateCustomers(),
		builder.createCustomerToken(),
		builder.invalidateCustomerTokens(),
		builder.listOrders(),
		builder.getOrder(),
		builder.refundOrder(),
		builder.listPayments(),
		builder.getPayment(),
		builder.listSubscriptions(),
		builder.getSubscription(),
		builder.cancelSubscription(),
		builder.previewSubscriptionChange(),
		builder.changeSubscription(),
		builder.listWebhooks(),
		builder.createWebhook(),
		builder.updateWebhook(),
		builder.deleteWebhook(),
		builder.resendWebhook(),
		builder.listWebhookHistory(),
		builder.listWebhookVariables(),
		builder.listGameServers(),
		builder.getGameServer(),
		builder.createGameServer(),
		builder.updateGameServer(),
		builder.deleteGameServer(),
		builder.resetGameServerToken(),
	}
}

func (b toolBuilder) help() mcp.Tool {
	return mcp.Tool{
		Name:        "paynow_help",
		Title:       "PayNow MCP help",
		Description: "Show available PayNow MCP capabilities and common Management API paths.",
		InputSchema: emptySchema(),
		Handler: func(context.Context, map[string]any) (mcp.CallToolResult, error) {
			return mcp.NewToolResult(map[string]any{
				"authentication": "Set PAYNOW_API_KEY to a PayNow Management API key. Raw tokens are sent as 'APIKey <token>'; a value that already contains a space is sent as-is.",
				"base_url":       "Set PAYNOW_BASE_URL to override the default https://api.paynow.gg.",
				"destructive_operations": []string{
					"DELETE requests require confirm=true.",
					"Refunds, subscription cancellation, and game server token resets require confirm=true.",
				},
				"common_paths": []string{
					"GET /v1/stores",
					"GET /v1/stores/{storeId}/products",
					"GET /v1/stores/{storeId}/orders",
					"GET /v1/stores/{storeId}/customers",
					"POST /v1/stores/{storeId}/checkouts",
					"GET /v1/stores/{storeId}/subscriptions",
					"GET /v1/stores/{storeId}/webhooks",
				},
				"raw_access": "Use paynow_api_request for PayNow Management API endpoints that do not have a dedicated MCP tool yet.",
			}), nil
		},
	}
}

func (b toolBuilder) apiRequest() mcp.Tool {
	return mcp.Tool{
		Name:        "paynow_api_request",
		Title:       "PayNow API request",
		Description: "Call any PayNow Management API endpoint. Use a relative API path such as /v1/stores/{storeId}/products. DELETE requires confirm=true.",
		InputSchema: objectSchema([]string{"method", "path"}, map[string]any{
			"method":  enumProperty("HTTP method.", "GET", "POST", "PATCH", "PUT", "DELETE"),
			"path":    stringProperty("PayNow Management API path, for example /v1/stores or /v1/stores/{storeId}/products."),
			"query":   queryProperty(),
			"body":    bodyProperty("JSON request body. Use the exact fields expected by the PayNow Management API."),
			"confirm": boolProperty("Must be true for DELETE requests."),
		}),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			method, err := requiredString(args, "method")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			method = strings.ToUpper(method)
			if !allowedMethod(method) {
				return mcp.CallToolResult{}, fmt.Errorf("unsupported method %q", method)
			}
			if method == "DELETE" {
				if err := requireConfirm(args, "confirm", "send DELETE request"); err != nil {
					return mcp.CallToolResult{}, err
				}
			}

			path, err := requiredString(args, "path")
			if err != nil {
				return mcp.CallToolResult{}, err
			}

			query, err := optionalObject(args, "query")
			if err != nil {
				return mcp.CallToolResult{}, err
			}

			return b.call(ctx, method, path, query, args["body"])
		},
	}
}

func (b toolBuilder) listStores() mcp.Tool {
	return b.simpleListTool("paynow_list_stores", "List stores", "List PayNow stores available to the API key.", "/v1/stores")
}

func (b toolBuilder) getStore() mcp.Tool {
	return b.idTool("paynow_get_store", "Get store", "Get a PayNow store by ID.", "store_id", func(args map[string]any) (string, error) {
		storeID, err := requiredString(args, "store_id")
		return "/v1/stores/" + escape(storeID), err
	})
}

func (b toolBuilder) createStore() mcp.Tool {
	return b.bodyTool("paynow_create_store", "Create store", "Create a PayNow store. Body fields must match the PayNow CreateStoreDto.", "POST", "/v1/stores", nil)
}

func (b toolBuilder) updateStore() mcp.Tool {
	return b.storeBodyTool("paynow_update_store", "Update store", "Patch a PayNow store. Body fields must match the PayNow UpdateStoreDto.", "PATCH", func(storeID string) string {
		return "/v1/stores/" + escape(storeID)
	})
}

func (b toolBuilder) deleteStore() mcp.Tool {
	return b.deleteTool("paynow_delete_store", "Delete store", "Delete a PayNow store.", "store_id", func(storeID string) string {
		return "/v1/stores/" + escape(storeID)
	})
}

func (b toolBuilder) listProducts() mcp.Tool {
	return b.storeListTool("paynow_list_products", "List products", "List products for a store.", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/products"
	})
}

func (b toolBuilder) getProduct() mcp.Tool {
	return b.storeChildGetTool("paynow_get_product", "Get product", "Get a product by ID.", "product_id", func(storeID, productID string) string {
		return "/v1/stores/" + escape(storeID) + "/products/" + escape(productID)
	})
}

func (b toolBuilder) createProduct() mcp.Tool {
	return b.storeBodyTool("paynow_create_product", "Create product", "Create a product. Body fields must match the PayNow UpsertProductRequestDto.", "POST", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/products"
	})
}

func (b toolBuilder) updateProduct() mcp.Tool {
	return b.storeChildBodyTool("paynow_update_product", "Update product", "Patch a product. Body fields must match the PayNow UpsertProductRequestDto.", "PATCH", "product_id", func(storeID, productID string) string {
		return "/v1/stores/" + escape(storeID) + "/products/" + escape(productID)
	})
}

func (b toolBuilder) deleteProduct() mcp.Tool {
	return b.storeChildDeleteTool("paynow_delete_product", "Delete product", "Delete a product.", "product_id", func(storeID, productID string) string {
		return "/v1/stores/" + escape(storeID) + "/products/" + escape(productID)
	})
}

func (b toolBuilder) createCheckoutSession() mcp.Tool {
	return b.storeBodyTool("paynow_create_checkout_session", "Create checkout session", "Create a checkout session for a store. Body fields must match the PayNow CreateCheckoutSessionManagementDto.", "POST", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/checkouts"
	})
}

func (b toolBuilder) listCoupons() mcp.Tool {
	return b.storeListTool("paynow_list_coupons", "List coupons", "List coupons for a store.", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/coupons"
	})
}

func (b toolBuilder) getCoupon() mcp.Tool {
	return b.storeChildGetTool("paynow_get_coupon", "Get coupon", "Get a coupon by ID.", "coupon_id", func(storeID, couponID string) string {
		return "/v1/stores/" + escape(storeID) + "/coupons/" + escape(couponID)
	})
}

func (b toolBuilder) createCoupon() mcp.Tool {
	return b.storeBodyTool("paynow_create_coupon", "Create coupon", "Create a coupon. Body fields must match the PayNow CreateCouponDto.", "POST", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/coupons"
	})
}

func (b toolBuilder) updateCoupon() mcp.Tool {
	return b.storeChildBodyTool("paynow_update_coupon", "Update coupon", "Patch a coupon. Body fields must match the PayNow UpdateCouponDto.", "PATCH", "coupon_id", func(storeID, couponID string) string {
		return "/v1/stores/" + escape(storeID) + "/coupons/" + escape(couponID)
	})
}

func (b toolBuilder) deleteCoupon() mcp.Tool {
	return b.storeChildDeleteTool("paynow_delete_coupon", "Delete coupon", "Delete a coupon.", "coupon_id", func(storeID, couponID string) string {
		return "/v1/stores/" + escape(storeID) + "/coupons/" + escape(couponID)
	})
}

func (b toolBuilder) listCustomers() mcp.Tool {
	return b.storeListTool("paynow_list_customers", "List customers", "List customers for a store.", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/customers"
	})
}

func (b toolBuilder) getCustomer() mcp.Tool {
	return b.storeChildGetTool("paynow_get_customer", "Get customer", "Get a customer by ID.", "customer_id", func(storeID, customerID string) string {
		return "/v1/stores/" + escape(storeID) + "/customers/" + escape(customerID)
	})
}

func (b toolBuilder) lookupCustomer() mcp.Tool {
	return mcp.Tool{
		Name:        "paynow_lookup_customer",
		Title:       "Lookup customer",
		Description: "Look up a customer with PayNow query parameters such as email or external identifiers.",
		InputSchema: objectSchema([]string{"store_id", "query"}, map[string]any{
			"store_id": stringProperty("PayNow store ID."),
			"query":    queryProperty(),
		}),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			storeID, err := requiredString(args, "store_id")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			query, err := requiredObject(args, "query")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			return b.call(ctx, "GET", "/v1/stores/"+escape(storeID)+"/customers/lookup", query, nil)
		},
	}
}

func (b toolBuilder) createCustomer() mcp.Tool {
	return b.storeBodyTool("paynow_create_customer", "Create customer", "Create a customer. Body fields must match the PayNow UpsertCustomerRequestDto.", "POST", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/customers"
	})
}

func (b toolBuilder) updateCustomer() mcp.Tool {
	return b.storeChildBodyTool("paynow_update_customer", "Update customer", "Patch a customer. Body fields must match the PayNow UpsertCustomerRequestDto.", "PATCH", "customer_id", func(storeID, customerID string) string {
		return "/v1/stores/" + escape(storeID) + "/customers/" + escape(customerID)
	})
}

func (b toolBuilder) bulkCreateCustomers() mcp.Tool {
	return b.storeBodyTool("paynow_bulk_create_customers", "Bulk create customers", "Bulk create customers. Body must be an array of PayNow UpsertCustomerRequestDto objects.", "POST", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/customers/bulk"
	})
}

func (b toolBuilder) createCustomerToken() mcp.Tool {
	return b.storeChildPostNoBodyTool("paynow_create_customer_token", "Create customer token", "Create a customer token for Storefront API access.", "customer_id", false, func(storeID, customerID string) string {
		return "/v1/stores/" + escape(storeID) + "/customers/" + escape(customerID) + "/tokens"
	})
}

func (b toolBuilder) invalidateCustomerTokens() mcp.Tool {
	return b.storeChildDeleteTool("paynow_invalidate_customer_tokens", "Invalidate customer tokens", "Invalidate all tokens for a customer.", "customer_id", func(storeID, customerID string) string {
		return "/v1/stores/" + escape(storeID) + "/customers/" + escape(customerID) + "/tokens"
	})
}

func (b toolBuilder) listOrders() mcp.Tool {
	return b.storeListTool("paynow_list_orders", "List orders", "List orders for a store.", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/orders"
	})
}

func (b toolBuilder) getOrder() mcp.Tool {
	return b.storeChildGetTool("paynow_get_order", "Get order", "Get an order by ID.", "order_id", func(storeID, orderID string) string {
		return "/v1/stores/" + escape(storeID) + "/orders/" + escape(orderID)
	})
}

func (b toolBuilder) refundOrder() mcp.Tool {
	return mcp.Tool{
		Name:        "paynow_refund_order",
		Title:       "Refund order",
		Description: "Refund an order. Body fields must match the PayNow CreateRefundRequestDto. Requires confirm=true.",
		InputSchema: objectSchema([]string{"store_id", "order_id", "body", "confirm"}, map[string]any{
			"store_id": stringProperty("PayNow store ID."),
			"order_id": stringProperty("Order ID."),
			"body":     bodyProperty("Refund request body."),
			"confirm":  boolProperty("Must be true to refund the order."),
		}),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			if err := requireConfirm(args, "confirm", "refund order"); err != nil {
				return mcp.CallToolResult{}, err
			}
			storeID, orderID, err := requiredStoreChild(args, "order_id")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			body, err := requiredAny(args, "body")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			return b.call(ctx, "POST", "/v1/stores/"+escape(storeID)+"/orders/"+escape(orderID)+"/refund", nil, body)
		},
	}
}

func (b toolBuilder) listPayments() mcp.Tool {
	return b.storeListTool("paynow_list_payments", "List payments", "List payments for a store.", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/payments"
	})
}

func (b toolBuilder) getPayment() mcp.Tool {
	return b.storeChildGetTool("paynow_get_payment", "Get payment", "Get a payment by ID.", "payment_id", func(storeID, paymentID string) string {
		return "/v1/stores/" + escape(storeID) + "/payments/" + escape(paymentID)
	})
}

func (b toolBuilder) listSubscriptions() mcp.Tool {
	return b.storeListTool("paynow_list_subscriptions", "List subscriptions", "List subscriptions for a store.", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/subscriptions"
	})
}

func (b toolBuilder) getSubscription() mcp.Tool {
	return b.storeChildGetTool("paynow_get_subscription", "Get subscription", "Get a subscription by ID.", "subscription_id", func(storeID, subscriptionID string) string {
		return "/v1/stores/" + escape(storeID) + "/subscriptions/" + escape(subscriptionID)
	})
}

func (b toolBuilder) cancelSubscription() mcp.Tool {
	return b.storeChildPostNoBodyTool("paynow_cancel_subscription", "Cancel subscription", "Cancel a subscription. Requires confirm=true.", "subscription_id", true, func(storeID, subscriptionID string) string {
		return "/v1/stores/" + escape(storeID) + "/subscriptions/" + escape(subscriptionID) + "/cancel"
	})
}

func (b toolBuilder) previewSubscriptionChange() mcp.Tool {
	return b.storeChildBodyTool("paynow_preview_subscription_change", "Preview subscription change", "Preview a subscription item change. Body fields must match PayNow UpdateSubscriptionRequestDto.", "POST", "subscription_id", func(storeID, subscriptionID string) string {
		return "/v1/stores/" + escape(storeID) + "/subscriptions/" + escape(subscriptionID) + "/change/preview"
	})
}

func (b toolBuilder) changeSubscription() mcp.Tool {
	return b.storeChildBodyTool("paynow_change_subscription", "Change subscription", "Update items on a subscription. Body fields must match PayNow UpdateSubscriptionRequestDto.", "POST", "subscription_id", func(storeID, subscriptionID string) string {
		return "/v1/stores/" + escape(storeID) + "/subscriptions/" + escape(subscriptionID) + "/change"
	})
}

func (b toolBuilder) listWebhooks() mcp.Tool {
	return b.storeListTool("paynow_list_webhooks", "List webhooks", "List webhook subscriptions for a store.", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/webhooks"
	})
}

func (b toolBuilder) createWebhook() mcp.Tool {
	return b.storeBodyTool("paynow_create_webhook", "Create webhook", "Create a webhook subscription. Body fields must match PayNow CreateWebhookDto.", "POST", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/webhooks"
	})
}

func (b toolBuilder) updateWebhook() mcp.Tool {
	return b.storeChildBodyTool("paynow_update_webhook", "Update webhook", "Patch a webhook subscription. Body fields must match PayNow UpdateWebhookDto.", "PATCH", "webhook_id", func(storeID, webhookID string) string {
		return "/v1/stores/" + escape(storeID) + "/webhooks/" + escape(webhookID)
	})
}

func (b toolBuilder) deleteWebhook() mcp.Tool {
	return b.storeChildDeleteTool("paynow_delete_webhook", "Delete webhook", "Delete a webhook subscription.", "webhook_id", func(storeID, webhookID string) string {
		return "/v1/stores/" + escape(storeID) + "/webhooks/" + escape(webhookID)
	})
}

func (b toolBuilder) resendWebhook() mcp.Tool {
	return b.storeBodyTool("paynow_resend_webhook", "Resend webhook", "Resend a webhook. Body fields must match PayNow ResendWebhookDto.", "POST", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/webhooks/resend"
	})
}

func (b toolBuilder) listWebhookHistory() mcp.Tool {
	return b.storeChildListTool("paynow_list_webhook_history", "List webhook history", "List webhook delivery history.", "webhook_id", func(storeID, webhookID string) string {
		return "/v1/stores/" + escape(storeID) + "/webhooks/" + escape(webhookID) + "/history"
	})
}

func (b toolBuilder) listWebhookVariables() mcp.Tool {
	return b.storeListTool("paynow_list_webhook_variables", "List webhook variables", "List available webhook variables for a store.", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/webhooks/variables"
	})
}

func (b toolBuilder) listGameServers() mcp.Tool {
	return b.storeListTool("paynow_list_game_servers", "List game servers", "List game servers for a store.", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/gameservers"
	})
}

func (b toolBuilder) getGameServer() mcp.Tool {
	return b.storeChildGetTool("paynow_get_game_server", "Get game server", "Get a game server by ID.", "game_server_id", func(storeID, gameServerID string) string {
		return "/v1/stores/" + escape(storeID) + "/gameservers/" + escape(gameServerID)
	})
}

func (b toolBuilder) createGameServer() mcp.Tool {
	return b.storeBodyTool("paynow_create_game_server", "Create game server", "Create a game server. Body fields must match PayNow CreateGameServerDto.", "POST", func(storeID string) string {
		return "/v1/stores/" + escape(storeID) + "/gameservers"
	})
}

func (b toolBuilder) updateGameServer() mcp.Tool {
	return b.storeChildBodyTool("paynow_update_game_server", "Update game server", "Patch a game server. Body fields must match PayNow UpdateGameServerDto.", "PATCH", "game_server_id", func(storeID, gameServerID string) string {
		return "/v1/stores/" + escape(storeID) + "/gameservers/" + escape(gameServerID)
	})
}

func (b toolBuilder) deleteGameServer() mcp.Tool {
	return b.storeChildDeleteTool("paynow_delete_game_server", "Delete game server", "Delete a game server.", "game_server_id", func(storeID, gameServerID string) string {
		return "/v1/stores/" + escape(storeID) + "/gameservers/" + escape(gameServerID)
	})
}

func (b toolBuilder) resetGameServerToken() mcp.Tool {
	return b.storeChildPostNoBodyTool("paynow_reset_game_server_token", "Reset game server token", "Reset a game server API token. Requires confirm=true.", "game_server_id", true, func(storeID, gameServerID string) string {
		return "/v1/stores/" + escape(storeID) + "/gameservers/" + escape(gameServerID) + "/reset-token"
	})
}

func (b toolBuilder) simpleListTool(name, title, description, path string) mcp.Tool {
	return mcp.Tool{
		Name:        name,
		Title:       title,
		Description: description,
		InputSchema: objectSchema(nil, map[string]any{
			"query": queryProperty(),
		}),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			query, err := optionalObject(args, "query")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			return b.call(ctx, "GET", path, query, nil)
		},
	}
}

func (b toolBuilder) idTool(name, title, description, idField string, path func(map[string]any) (string, error)) mcp.Tool {
	return mcp.Tool{
		Name:        name,
		Title:       title,
		Description: description,
		InputSchema: objectSchema([]string{idField}, map[string]any{
			idField: stringProperty("PayNow " + strings.ReplaceAll(idField, "_", " ") + "."),
			"query": queryProperty(),
		}),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			resolvedPath, err := path(args)
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			query, err := optionalObject(args, "query")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			return b.call(ctx, "GET", resolvedPath, query, nil)
		},
	}
}

func (b toolBuilder) bodyTool(name, title, description, method, path string, extraProps map[string]any) mcp.Tool {
	props := map[string]any{
		"body": bodyProperty("JSON request body."),
	}
	for key, value := range extraProps {
		props[key] = value
	}

	return mcp.Tool{
		Name:        name,
		Title:       title,
		Description: description,
		InputSchema: objectSchema([]string{"body"}, props),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			body, err := requiredAny(args, "body")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			return b.call(ctx, method, path, nil, body)
		},
	}
}

func (b toolBuilder) storeListTool(name, title, description string, path func(string) string) mcp.Tool {
	return mcp.Tool{
		Name:        name,
		Title:       title,
		Description: description,
		InputSchema: objectSchema([]string{"store_id"}, map[string]any{
			"store_id": stringProperty("PayNow store ID."),
			"query":    queryProperty(),
		}),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			storeID, err := requiredString(args, "store_id")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			query, err := optionalObject(args, "query")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			return b.call(ctx, "GET", path(storeID), query, nil)
		},
	}
}

func (b toolBuilder) storeChildListTool(name, title, description, childField string, path func(string, string) string) mcp.Tool {
	return mcp.Tool{
		Name:        name,
		Title:       title,
		Description: description,
		InputSchema: objectSchema([]string{"store_id", childField}, map[string]any{
			"store_id": stringProperty("PayNow store ID."),
			childField: stringProperty("PayNow " + strings.ReplaceAll(childField, "_", " ") + "."),
			"query":    queryProperty(),
		}),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			storeID, childID, err := requiredStoreChild(args, childField)
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			query, err := optionalObject(args, "query")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			return b.call(ctx, "GET", path(storeID, childID), query, nil)
		},
	}
}

func (b toolBuilder) storeChildGetTool(name, title, description, childField string, path func(string, string) string) mcp.Tool {
	return b.storeChildListTool(name, title, description, childField, path)
}

func (b toolBuilder) storeBodyTool(name, title, description, method string, path func(string) string) mcp.Tool {
	return mcp.Tool{
		Name:        name,
		Title:       title,
		Description: description,
		InputSchema: objectSchema([]string{"store_id", "body"}, map[string]any{
			"store_id": stringProperty("PayNow store ID."),
			"body":     bodyProperty("JSON request body."),
		}),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			storeID, err := requiredString(args, "store_id")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			body, err := requiredAny(args, "body")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			return b.call(ctx, method, path(storeID), nil, body)
		},
	}
}

func (b toolBuilder) storeChildBodyTool(name, title, description, method, childField string, path func(string, string) string) mcp.Tool {
	return mcp.Tool{
		Name:        name,
		Title:       title,
		Description: description,
		InputSchema: objectSchema([]string{"store_id", childField, "body"}, map[string]any{
			"store_id": stringProperty("PayNow store ID."),
			childField: stringProperty("PayNow " + strings.ReplaceAll(childField, "_", " ") + "."),
			"body":     bodyProperty("JSON request body."),
		}),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			storeID, childID, err := requiredStoreChild(args, childField)
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			body, err := requiredAny(args, "body")
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			return b.call(ctx, method, path(storeID, childID), nil, body)
		},
	}
}

func (b toolBuilder) deleteTool(name, title, description, idField string, path func(string) string) mcp.Tool {
	return mcp.Tool{
		Name:        name,
		Title:       title,
		Description: description + " Requires confirm=true.",
		InputSchema: objectSchema([]string{idField, "confirm"}, map[string]any{
			idField:   stringProperty("PayNow " + strings.ReplaceAll(idField, "_", " ") + "."),
			"confirm": boolProperty("Must be true to delete."),
		}),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			if err := requireConfirm(args, "confirm", "delete"); err != nil {
				return mcp.CallToolResult{}, err
			}
			id, err := requiredString(args, idField)
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			return b.call(ctx, "DELETE", path(id), nil, nil)
		},
	}
}

func (b toolBuilder) storeChildDeleteTool(name, title, description, childField string, path func(string, string) string) mcp.Tool {
	return mcp.Tool{
		Name:        name,
		Title:       title,
		Description: description + " Requires confirm=true.",
		InputSchema: objectSchema([]string{"store_id", childField, "confirm"}, map[string]any{
			"store_id": stringProperty("PayNow store ID."),
			childField: stringProperty("PayNow " + strings.ReplaceAll(childField, "_", " ") + "."),
			"confirm":  boolProperty("Must be true to delete."),
		}),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			if err := requireConfirm(args, "confirm", "delete"); err != nil {
				return mcp.CallToolResult{}, err
			}
			storeID, childID, err := requiredStoreChild(args, childField)
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			return b.call(ctx, "DELETE", path(storeID, childID), nil, nil)
		},
	}
}

func (b toolBuilder) storeChildPostNoBodyTool(name, title, description, childField string, confirm bool, path func(string, string) string) mcp.Tool {
	required := []string{"store_id", childField}
	props := map[string]any{
		"store_id": stringProperty("PayNow store ID."),
		childField: stringProperty("PayNow " + strings.ReplaceAll(childField, "_", " ") + "."),
	}
	if confirm {
		required = append(required, "confirm")
		props["confirm"] = boolProperty("Must be true to perform this operation.")
	}

	return mcp.Tool{
		Name:        name,
		Title:       title,
		Description: description,
		InputSchema: objectSchema(required, props),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			if confirm {
				if err := requireConfirm(args, "confirm", title); err != nil {
					return mcp.CallToolResult{}, err
				}
			}
			storeID, childID, err := requiredStoreChild(args, childField)
			if err != nil {
				return mcp.CallToolResult{}, err
			}
			return b.call(ctx, "POST", path(storeID, childID), nil, nil)
		},
	}
}

func (b toolBuilder) call(ctx context.Context, method, path string, query map[string]any, body any) (mcp.CallToolResult, error) {
	result, err := b.client.Do(ctx, method, path, query, body)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			return mcp.NewToolError(apiErr.Response), nil
		}
		return mcp.NewToolError(map[string]any{"error": err.Error()}), nil
	}
	return mcp.NewToolResult(result), nil
}

func requiredStoreChild(args map[string]any, childField string) (string, string, error) {
	storeID, err := requiredString(args, "store_id")
	if err != nil {
		return "", "", err
	}
	childID, err := requiredString(args, childField)
	if err != nil {
		return "", "", err
	}
	return storeID, childID, nil
}

func requiredString(args map[string]any, name string) (string, error) {
	value, ok := args[name]
	if !ok {
		return "", fmt.Errorf("%s is required", name)
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", name)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return text, nil
}

func requiredAny(args map[string]any, name string) (any, error) {
	value, ok := args[name]
	if !ok || value == nil {
		return nil, fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func requiredObject(args map[string]any, name string) (map[string]any, error) {
	value, ok := args[name]
	if !ok || value == nil {
		return nil, fmt.Errorf("%s is required", name)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", name)
	}
	return object, nil
}

func optionalObject(args map[string]any, name string) (map[string]any, error) {
	value, ok := args[name]
	if !ok || value == nil {
		return nil, nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", name)
	}
	return object, nil
}

func requireConfirm(args map[string]any, name, action string) error {
	value, ok := args[name]
	if !ok {
		return fmt.Errorf("%s requires %s=true", action, name)
	}
	confirmed, ok := value.(bool)
	if !ok || !confirmed {
		return fmt.Errorf("%s requires %s=true", action, name)
	}
	return nil
}

func allowedMethod(method string) bool {
	switch method {
	case "GET", "POST", "PATCH", "PUT", "DELETE":
		return true
	default:
		return false
	}
}

func escape(value string) string {
	return url.PathEscape(value)
}
