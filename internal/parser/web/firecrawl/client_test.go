package firecrawl_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"tldr/internal/models"
	"tldr/internal/parser/web/firecrawl"
)

func TestClient_Scrape(t *testing.T) {
	t.Run("successful scrape", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v2/scrape" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Fatalf("unexpected auth header: %s", r.Header.Get("Authorization"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"success": true,
				"data": {
					"markdown": "# Hello World\nThis is scraped markdown.",
					"metadata": {
						"title": "Hello World",
						"statusCode": 200
					}
				}
			}`))
		}))
		defer ts.Close()

		client := firecrawl.NewClient(ts.URL, "test-key")
		content, err := client.Scrape(context.Background(), "https://example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "# Hello World\nThis is scraped markdown." {
			t.Fatalf("unexpected content: %q", content)
		}
	})

	t.Run("not found status maps to ErrNotFound", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		client := firecrawl.NewClient(ts.URL, "")
		_, err := client.Scrape(context.Background(), "https://example.com/notfound")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, models.ErrNotFound) {
			t.Fatalf("expected models.ErrNotFound, got %v", err)
		}
	})

	t.Run("rate limit status maps to ErrRateLimitExceeded", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		client := firecrawl.NewClient(ts.URL, "")
		_, err := client.Scrape(context.Background(), "https://example.com/ratelimit")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, models.ErrRateLimitExceeded) {
			t.Fatalf("expected models.ErrRateLimitExceeded, got %v", err)
		}
	})
}
