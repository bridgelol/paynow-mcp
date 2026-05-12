package paynow

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/bridgelol/paynow-mcp/internal/mcp"
)

//go:embed openapi/*.json
var openAPIFS embed.FS

type openAPIDoc struct {
	Paths map[string]map[string]openAPIOperation `json:"paths"`
}

type openAPIOperation struct {
	OperationID string              `json:"operationId"`
	Summary     string              `json:"summary"`
	Description string              `json:"description"`
	Deprecated  bool                `json:"deprecated"`
	Tags        []string            `json:"tags"`
	Parameters  []openAPIParameter  `json:"parameters"`
	RequestBody *openAPIRequestBody `json:"requestBody"`
}

type openAPIParameter struct {
	Name        string         `json:"name"`
	In          string         `json:"in"`
	Required    bool           `json:"required"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema"`
}

type openAPIRequestBody struct {
	Required    bool                          `json:"required"`
	Description string                        `json:"description"`
	Content     map[string]openAPIContentType `json:"content"`
}

type openAPIContentType struct {
	Schema map[string]any `json:"schema"`
}

type openAPIBundle struct {
	Label  string
	File   string
	Prefix string
}

var openAPIBundles = map[string]openAPIBundle{
	"management": {Label: "management", File: "management-api.json", Prefix: "paynow"},
	"storefront": {Label: "storefront", File: "storefront-api.json", Prefix: "paynow_storefront"},
	"gameserver": {Label: "gameserver", File: "gameserver-api.json", Prefix: "paynow_gameserver"},
}

func OpenAPITools(registry *Registry) []mcp.Tool {
	selected, err := selectedOpenAPIBundles()
	if err != nil {
		return []mcp.Tool{openAPIConfigErrorTool(err)}
	}

	var tools []mcp.Tool
	for _, bundle := range selected {
		doc, err := loadOpenAPIDoc(bundle.File)
		if err != nil {
			return []mcp.Tool{openAPIConfigErrorTool(err)}
		}

		tools = append(tools, buildOpenAPITools(registry, bundle, doc)...)
	}

	return tools
}

func selectedOpenAPIBundles() ([]openAPIBundle, error) {
	if disabledEnv("PAYNOW_OPENAPI_TOOLS") || enabledEnv("PAYNOW_DISABLE_OPENAPI_TOOLS") {
		return nil, nil
	}

	raw := strings.TrimSpace(os.Getenv("PAYNOW_INCLUDE_APIS"))
	if raw == "" {
		raw = "management"
	}

	seen := map[string]bool{}
	var selected []openAPIBundle
	for _, part := range strings.Split(raw, ",") {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		if name == "none" || name == "curated" {
			return nil, nil
		}

		bundle, ok := openAPIBundles[name]
		if !ok {
			return nil, fmt.Errorf("unknown PAYNOW_INCLUDE_APIS value %q; valid values are management, storefront, gameserver", name)
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		selected = append(selected, bundle)
	}

	return selected, nil
}

func disabledEnv(name string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return value == "0" || value == "false" || value == "off" || value == "no"
}

func enabledEnv(name string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return value == "1" || value == "true" || value == "on" || value == "yes"
}

func loadOpenAPIDoc(file string) (openAPIDoc, error) {
	data, err := openAPIFS.ReadFile("openapi/" + file)
	if err != nil {
		return openAPIDoc{}, err
	}

	var doc openAPIDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return openAPIDoc{}, fmt.Errorf("parse %s: %w", file, err)
	}

	return doc, nil
}

func buildOpenAPITools(registry *Registry, bundle openAPIBundle, doc openAPIDoc) []mcp.Tool {
	var tools []mcp.Tool
	for _, operation := range openAPIOperations(doc) {
		if operation.Operation.OperationID == "" {
			continue
		}
		tools = append(tools, buildOpenAPITool(registry, bundle, operation))
	}
	return tools
}

type openAPIOperationEntry struct {
	Path      string
	Method    string
	Operation openAPIOperation
}

func openAPIOperations(doc openAPIDoc) []openAPIOperationEntry {
	var entries []openAPIOperationEntry
	for path, methods := range doc.Paths {
		for method, operation := range methods {
			method = strings.ToUpper(method)
			if !allowedMethod(method) {
				continue
			}
			entries = append(entries, openAPIOperationEntry{
				Path:      path,
				Method:    method,
				Operation: operation,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Path == entries[j].Path {
			return entries[i].Method < entries[j].Method
		}
		return entries[i].Path < entries[j].Path
	})

	return entries
}

func buildOpenAPITool(registry *Registry, bundle openAPIBundle, entry openAPIOperationEntry) mcp.Tool {
	operation := entry.Operation
	name := bundle.Prefix + "_" + snakeCase(operation.OperationID)
	bodySchema, hasBody := requestBodySchema(operation.RequestBody)

	pathParams := filterOpenAPIParams(operation.Parameters, "path")
	queryParams := filterOpenAPIParams(operation.Parameters, "query")

	required := make([]string, 0, len(pathParams)+len(queryParams)+2)
	properties := map[string]any{
		"query": queryProperty(),
	}

	for _, param := range pathParams {
		property := schemaProperty(param.Schema, param.Description)
		if param.Name == "storeId" {
			property["description"] = descriptionWithFallback(param.Description, "Store ID. Optional if the selected profile has store_id.")
		} else {
			required = append(required, param.Name)
		}
		properties[param.Name] = property
	}

	for _, param := range queryParams {
		properties[param.Name] = schemaProperty(param.Schema, param.Description)
		if param.Required {
			required = append(required, param.Name)
		}
	}

	if hasBody {
		properties["body"] = bodyProperty(descriptionWithFallback(operation.RequestBody.Description, "JSON request body."))
		if operation.RequestBody.Required || len(bodySchema) > 0 {
			required = append(required, "body")
		}
	}

	if entry.Method == "DELETE" {
		properties["confirm"] = boolProperty("Must be true for DELETE requests.")
		required = append(required, "confirm")
	}

	description := openAPIToolDescription(bundle, entry)

	return mcp.Tool{
		Name:        name,
		Title:       operationTitle(operation, entry.Method, entry.Path),
		Description: description,
		InputSchema: objectSchema(required, properties),
		Handler: func(ctx context.Context, args map[string]any) (mcp.CallToolResult, error) {
			if entry.Method == "DELETE" {
				if err := requireConfirm(args, "confirm", "send DELETE request"); err != nil {
					return mcp.CallToolResult{}, err
				}
			}

			profile, err := profileFromRegistry(registry, args)
			if err != nil {
				return mcp.CallToolResult{}, err
			}

			path, err := resolveOpenAPIPath(entry.Path, pathParams, args, profile)
			if err != nil {
				return mcp.CallToolResult{}, err
			}

			query, err := openAPIQuery(queryParams, args)
			if err != nil {
				return mcp.CallToolResult{}, err
			}

			var body any
			if hasBody {
				body = args["body"]
			}

			result, err := profile.Client.Do(ctx, entry.Method, path, query, body)
			if err != nil {
				if apiErr, ok := err.(*APIError); ok {
					return mcp.NewToolError(apiErr.Response), nil
				}
				return mcp.NewToolError(map[string]any{"error": err.Error()}), nil
			}

			return mcp.NewToolResult(result), nil
		},
	}
}

func openAPIConfigErrorTool(err error) mcp.Tool {
	return mcp.Tool{
		Name:        "paynow_openapi_configuration_error",
		Title:       "OpenAPI configuration error",
		Description: "Reports a PayNow OpenAPI tool configuration error.",
		InputSchema: emptySchema(),
		Handler: func(context.Context, map[string]any) (mcp.CallToolResult, error) {
			return mcp.NewToolError(map[string]any{"error": err.Error()}), nil
		},
	}
}

func filterOpenAPIParams(params []openAPIParameter, location string) []openAPIParameter {
	var filtered []openAPIParameter
	for _, param := range params {
		if param.In == location && param.Name != "" {
			filtered = append(filtered, param)
		}
	}
	return filtered
}

func requestBodySchema(body *openAPIRequestBody) (map[string]any, bool) {
	if body == nil {
		return nil, false
	}
	for _, contentType := range []string{"application/json", "text/json"} {
		if content, ok := body.Content[contentType]; ok {
			return content.Schema, true
		}
	}
	return nil, false
}

func schemaProperty(schema map[string]any, description string) map[string]any {
	property := map[string]any{}
	if description != "" {
		property["description"] = description
	}

	if len(schema) == 0 {
		property["type"] = "string"
		return property
	}

	if enumValues, ok := schema["enum"].([]any); ok && len(enumValues) > 0 {
		property["enum"] = enumValues
	}

	if schemaType, ok := schema["type"].(string); ok && schemaType != "" {
		property["type"] = schemaType
		if schemaType == "array" {
			if items, ok := schema["items"].(map[string]any); ok {
				property["items"] = schemaProperty(items, "")
			}
		}
		return property
	}

	if _, ok := schema["$ref"].(string); ok {
		property["type"] = "string"
		return property
	}

	property["type"] = "string"
	return property
}

func descriptionWithFallback(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func openAPIToolDescription(bundle openAPIBundle, entry openAPIOperationEntry) string {
	operation := entry.Operation
	parts := []string{}
	if operation.Summary != "" {
		parts = append(parts, operation.Summary)
	}
	if operation.Description != "" && operation.Description != operation.Summary {
		parts = append(parts, operation.Description)
	}
	if operation.Deprecated {
		parts = append(parts, "Deprecated operation.")
	}
	parts = append(parts, fmt.Sprintf("%s %s", entry.Method, entry.Path))
	if bundle.Label != "management" {
		parts = append(parts, bundle.Label+" API")
	}
	parts = append(parts, "Generated from PayNow's bundled OpenAPI specification. Pass profile to target a named PAYNOW_PROFILES entry.")
	return strings.Join(parts, "\n\n")
}

func operationTitle(operation openAPIOperation, method, path string) string {
	if operation.Summary != "" {
		return operation.Summary
	}
	if len(operation.Tags) > 0 {
		return operation.Tags[0] + " " + method + " " + path
	}
	return method + " " + path
}

func profileFromRegistry(registry *Registry, args map[string]any) (Profile, error) {
	name := ""
	if value, ok := args["profile"]; ok && value != nil {
		text, ok := value.(string)
		if !ok {
			return Profile{}, fmt.Errorf("profile must be a string")
		}
		name = strings.TrimSpace(text)
	}
	return registry.Get(name)
}

func resolveOpenAPIPath(pathTemplate string, pathParams []openAPIParameter, args map[string]any, profile Profile) (string, error) {
	path := pathTemplate
	for _, param := range pathParams {
		value, ok := args[param.Name]
		if (!ok || value == nil || strings.TrimSpace(fmt.Sprint(value)) == "") && param.Name == "storeId" && profile.StoreID != "" {
			value = profile.StoreID
			ok = true
		}
		if !ok || value == nil || strings.TrimSpace(fmt.Sprint(value)) == "" {
			return "", fmt.Errorf("%s is required", param.Name)
		}
		path = strings.ReplaceAll(path, "{"+param.Name+"}", url.PathEscape(fmt.Sprint(value)))
	}
	return path, nil
}

func openAPIQuery(queryParams []openAPIParameter, args map[string]any) (map[string]any, error) {
	query := map[string]any{}
	if raw, ok := args["query"]; ok && raw != nil {
		object, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("query must be an object")
		}
		for key, value := range object {
			query[key] = value
		}
	}

	for _, param := range queryParams {
		if value, ok := args[param.Name]; ok && value != nil {
			query[param.Name] = value
		}
	}

	return query, nil
}

var snakeBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)
var snakeWord = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func snakeCase(input string) string {
	value := snakeBoundary.ReplaceAllString(input, `${1}_${2}`)
	value = snakeWord.ReplaceAllString(value, "_")
	return strings.Trim(strings.ToLower(value), "_")
}
