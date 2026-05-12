package paynow

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDoBuildsPayNowRequest(t *testing.T) {
	var gotAuthorization string
	var gotPath string
	var gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:    "test-token",
		BaseURL:   server.URL,
		Timeout:   time.Second,
		UserAgent: "test-agent",
	})

	result, err := client.Do(context.Background(), "GET", "stores", map[string]any{
		"limit": float64(25),
		"tag":   []any{"a", "b"},
	}, nil)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}

	if gotAuthorization != "APIKey test-token" {
		t.Fatalf("authorization = %q", gotAuthorization)
	}
	if gotPath != "/v1/stores" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotQuery != "limit=25&tag=a&tag=b" {
		t.Fatalf("query = %q", gotQuery)
	}
	if result["status"] != http.StatusOK {
		t.Fatalf("status = %#v", result["status"])
	}
}

func TestDoPreservesFullAuthorizationHeader(t *testing.T) {
	var gotAuthorization string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:  "APIKey already-prefixed",
		BaseURL: server.URL,
		Timeout: time.Second,
	})

	if _, err := client.Do(context.Background(), "GET", "/v1/stores", nil, nil); err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if gotAuthorization != "APIKey already-prefixed" {
		t.Fatalf("authorization = %q", gotAuthorization)
	}
}

func TestDoRejectsFullRequestURL(t *testing.T) {
	client := NewClient(Config{
		APIKey:  "test-token",
		BaseURL: "https://api.paynow.gg",
		Timeout: time.Second,
	})

	if _, err := client.Do(context.Background(), "GET", "https://example.com/v1/stores", nil, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDoReturnsAPIErrorWithResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"bad request"}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:  "test-token",
		BaseURL: server.URL,
		Timeout: time.Second,
	})

	result, err := client.Do(context.Background(), "GET", "/v1/stores", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", apiErr.StatusCode)
	}
	if result["status"] != http.StatusBadRequest {
		t.Fatalf("result status = %#v", result["status"])
	}
}
