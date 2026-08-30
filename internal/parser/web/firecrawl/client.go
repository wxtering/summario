package firecrawl

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"tldr/internal/models"

	"resty.dev/v3"
)

type Client struct {
	client  *resty.Client
	baseURL string
	apiKey  string
}

func NewClient(baseURL, apiKey string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.firecrawl.dev"
	}

	client := resty.New().
		SetTimeout(45 * time.Second).
		SetRetryCount(1).
		SetRetryWaitTime(500 * time.Millisecond)

	return &Client{
		client:  client,
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

type ScrapeRequest struct {
	URL             string   `json:"url"`
	Formats         []string `json:"formats"`
	OnlyMainContent bool     `json:"onlyMainContent,omitempty"`
}

type ScrapeResponse struct {
	Success bool       `json:"success"`
	Data    ScrapeData `json:"data"`
	Error   string     `json:"error,omitempty"`
}

type ScrapeData struct {
	Markdown string         `json:"markdown"`
	Metadata ScrapeMetadata `json:"metadata"`
}

type ScrapeMetadata struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StatusCode  int    `json:"statusCode"`
}

// Scrape calls the Firecrawl /v2/scrape endpoint to extract clean markdown from targetURL.
func (c *Client) Scrape(ctx context.Context, targetURL string) (string, error) {
	reqBody := ScrapeRequest{
		URL:             targetURL,
		Formats:         []string{"markdown"},
		OnlyMainContent: true,
	}

	req := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+c.apiKey).
		SetBody(reqBody)

	var scrapeResp ScrapeResponse

	resp, err := req.
		SetResult(&scrapeResp).
		Post(c.baseURL + "/v2/scrape")

	if err != nil {
		return "", fmt.Errorf("%w: firecrawl scrape request failed: %w", models.ErrUpstreamFailed, err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return "", fmt.Errorf("%w: firecrawl reported target page not found", models.ErrNotFound)
	}
	if resp.StatusCode() == http.StatusForbidden || resp.StatusCode() == http.StatusUnauthorized {
		return "", fmt.Errorf("%w: firecrawl access forbidden or unauthorized (status %d)", models.ErrRestricted, resp.StatusCode())
	}
	if resp.StatusCode() == http.StatusTooManyRequests {
		return "", fmt.Errorf("%w: firecrawl rate limit exceeded", models.ErrRateLimitExceeded)
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("%w: firecrawl returned status %d: %s", models.ErrUpstreamFailed, resp.StatusCode(), scrapeResp.Error)
	}

	if !scrapeResp.Success && scrapeResp.Error != "" {
		return "", fmt.Errorf("%w: firecrawl error: %s", models.ErrUpstreamFailed, scrapeResp.Error)
	}

	content := strings.TrimSpace(scrapeResp.Data.Markdown)
	if content == "" {
		return "", fmt.Errorf("%w: firecrawl returned empty markdown", models.ErrNoContent)
	}

	return content, nil
}
