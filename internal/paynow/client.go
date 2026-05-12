package paynow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.paynow.gg"

type Config struct {
	APIKey     string
	AuthPrefix string
	BaseURL    string
	Timeout    time.Duration
	UserAgent  string
}

type Client struct {
	apiKey     string
	authPrefix string
	baseURL    string
	userAgent  string
	httpClient *http.Client
}

type APIError struct {
	StatusCode int
	Response   map[string]any
}

func (e *APIError) Error() string {
	return fmt.Sprintf("PayNow API returned HTTP %d", e.StatusCode)
}

func ConfigFromEnv() Config {
	timeout := 30 * time.Second
	if raw := strings.TrimSpace(os.Getenv("PAYNOW_TIMEOUT_SECONDS")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	baseURL := strings.TrimSpace(os.Getenv("PAYNOW_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	authPrefix := strings.TrimSpace(os.Getenv("PAYNOW_AUTH_PREFIX"))
	if authPrefix == "" {
		authPrefix = "APIKey"
	}

	return Config{
		APIKey:     strings.TrimSpace(os.Getenv("PAYNOW_API_KEY")),
		AuthPrefix: authPrefix,
		BaseURL:    baseURL,
		Timeout:    timeout,
		UserAgent:  "paynow-mcp",
	}
}

func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}
	if config.AuthPrefix == "" {
		config.AuthPrefix = "APIKey"
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.UserAgent == "" {
		config.UserAgent = "paynow-mcp"
	}

	return &Client{
		apiKey:     config.APIKey,
		authPrefix: config.AuthPrefix,
		baseURL:    strings.TrimRight(config.BaseURL, "/"),
		userAgent:  config.UserAgent,
		httpClient: &http.Client{Timeout: config.Timeout},
	}
}

func (c *Client) Do(ctx context.Context, method, requestPath string, query map[string]any, body any) (map[string]any, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, errors.New("PAYNOW_API_KEY is not set")
	}

	endpoint, err := c.urlFor(requestPath, query)
	if err != nil {
		return nil, err
	}

	var requestBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		requestBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), endpoint, requestBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authorization())
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	payload, err := decodeResponse(resp)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"status": resp.StatusCode,
		"body":   payload,
	}
	if headers := responseHeaders(resp.Header); len(headers) > 0 {
		result["headers"] = headers
	}

	if resp.StatusCode >= 400 {
		return result, &APIError{StatusCode: resp.StatusCode, Response: result}
	}

	return result, nil
}

func (c *Client) authorization() string {
	apiKey := strings.TrimSpace(c.apiKey)
	if strings.Contains(apiKey, " ") {
		return apiKey
	}

	return c.authPrefix + " " + apiKey
}

func (c *Client) urlFor(requestPath string, query map[string]any) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid PAYNOW_BASE_URL: %w", err)
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return "", errors.New("PAYNOW_BASE_URL must use http or https")
	}

	normalizedPath, err := normalizeAPIPath(requestPath)
	if err != nil {
		return "", err
	}

	basePath := strings.TrimRight(base.EscapedPath(), "/")
	if strings.HasSuffix(basePath, "/v1") && strings.HasPrefix(normalizedPath, "/v1/") {
		normalizedPath = strings.TrimPrefix(normalizedPath, "/v1")
	}

	base.Path = joinURLPath(basePath, normalizedPath)
	base.RawQuery = encodeQuery(query)

	return base.String(), nil
}

func normalizeAPIPath(requestPath string) (string, error) {
	path := strings.TrimSpace(requestPath)
	if path == "" {
		return "", errors.New("path is required")
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "//") {
		return "", errors.New("path must be a PayNow API path, not a full URL")
	}
	if strings.Contains(path, "..") {
		return "", errors.New("path must not contain ..")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasPrefix(path, "/v") {
		path = "/v1" + path
	}

	return path, nil
}

func joinURLPath(basePath, requestPath string) string {
	basePath = strings.TrimRight(basePath, "/")
	requestPath = strings.TrimLeft(requestPath, "/")
	if basePath == "" {
		return "/" + requestPath
	}
	return basePath + "/" + requestPath
}

func encodeQuery(query map[string]any) string {
	if len(query) == 0 {
		return ""
	}

	values := url.Values{}
	for key, value := range query {
		addQueryValue(values, key, value)
	}

	return values.Encode()
}

func addQueryValue(values url.Values, key string, value any) {
	if value == nil {
		return
	}

	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			addQueryValue(values, key, item)
		}
	case []string:
		for _, item := range typed {
			values.Add(key, item)
		}
	case string:
		values.Add(key, typed)
	case bool:
		values.Add(key, strconv.FormatBool(typed))
	case float64:
		values.Add(key, strconv.FormatFloat(typed, 'f', -1, 64))
	case float32:
		values.Add(key, strconv.FormatFloat(float64(typed), 'f', -1, 32))
	case int:
		values.Add(key, strconv.Itoa(typed))
	case int64:
		values.Add(key, strconv.FormatInt(typed, 10))
	case json.Number:
		values.Add(key, typed.String())
	default:
		values.Add(key, fmt.Sprint(typed))
	}
}

func decodeResponse(resp *http.Response) (any, error) {
	data, err := io.ReadAll(io.LimitReader(resp.Body, 20*1024*1024))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	var parsed any
	if err := json.Unmarshal(data, &parsed); err == nil {
		return parsed, nil
	}

	return string(data), nil
}

func responseHeaders(headers http.Header) map[string]string {
	allowed := []string{
		"Content-Type",
		"Retry-After",
		"X-Request-Id",
		"X-Request-ID",
		"Request-Id",
		"Request-ID",
	}

	result := map[string]string{}
	for _, key := range allowed {
		if value := headers.Get(key); value != "" {
			result[key] = value
		}
	}

	return result
}
