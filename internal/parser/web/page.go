package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"tldr/internal/models"

	"github.com/markusmobius/go-trafilatura"
)

const (
	minMeaningfulContentLength = 150
)

func (w *WebParser) ParseWebPage(ctx context.Context, source string) (string, error) {
	// 1. If user explicitly configured PREFER_FIRECRAWL, run Firecrawl directly without fallback.
	if w.preferFirecrawl {
		slog.Debug("prefer firecrawl enabled, scraping directly", slog.String("source", source))
		return w.firecrawlClient.Scrape(ctx, source)
	}

	// 2. Primary fast path: Local Trafilatura extraction
	content, err := w.extractWithTrafilatura(ctx, source)
	if err == nil && len(strings.TrimSpace(content)) >= minMeaningfulContentLength {
		slog.Debug("trafilatura extraction succeeded",
			slog.String("source", source),
			slog.Int("chars_count", len(strings.TrimSpace(content))),
		)
		return content, nil
	}

	// 3. Fallback path: If Trafilatura failed or returned insufficient content (< 150 chars), fallback to Firecrawl
	if w.fallbackEnabled {
		slog.Debug("trafilatura insufficient or failed, attempting firecrawl fallback",
			slog.String("source", source),
			slog.Int("trafilatura_chars", len(strings.TrimSpace(content))),
			slog.Any("trafilatura_err", err),
		)

		fcContent, fcErr := w.firecrawlClient.Scrape(ctx, source)
		if fcErr == nil && len(strings.TrimSpace(fcContent)) > 0 {
			slog.Debug("firecrawl fallback succeeded",
				slog.String("source", source),
				slog.Int("chars_count", len(strings.TrimSpace(fcContent))),
			)
			return fcContent, nil
		}
		// If both Trafilatura and Firecrawl failed during fallback, wrap in ErrUpstreamFailed
		if fcErr != nil && err != nil {
			return "", fmt.Errorf("%w: trafilatura (%v), firecrawl fallback (%v)", models.ErrUpstreamFailed, err, fcErr)
		}
		if fcErr != nil && len(strings.TrimSpace(content)) < minMeaningfulContentLength {
			return "", fmt.Errorf("%w: content too short and firecrawl fallback failed: %w", models.ErrNoContent, fcErr)
		}
	}

	if err != nil {
		return "", fmt.Errorf("%w: %w", models.ErrUpstreamFailed, err)
	}
	if len(strings.TrimSpace(content)) < minMeaningfulContentLength {
		return "", fmt.Errorf("%w: extracted webpage content is too short (%d chars)", models.ErrNoContent, len(strings.TrimSpace(content)))
	}
	return content, nil
}

func (w *WebParser) extractWithTrafilatura(ctx context.Context, source string) (string, error) {
	resp, err := w.client.R().
		SetContext(ctx).
		Get(source)
	if err != nil {
		return "", fmt.Errorf("%w: %w", models.ErrUpstreamFailed, err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return "", fmt.Errorf("%w: page not found (%s)", models.ErrNotFound, source)
	}
	if resp.StatusCode() == http.StatusForbidden || resp.StatusCode() == http.StatusUnauthorized {
		return "", fmt.Errorf("%w: access forbidden (status %d)", models.ErrRestricted, resp.StatusCode())
	}
	if resp.StatusCode() == http.StatusTooManyRequests {
		return "", fmt.Errorf("%w: rate limited by remote server", models.ErrRateLimitExceeded)
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("%w: unexpected HTTP status %d", models.ErrUpstreamFailed, resp.StatusCode())
	}

	opts := trafilatura.Options{}
	res, err := trafilatura.Extract(resp.Body, opts)
	if err != nil {
		return "", fmt.Errorf("%w: extraction failed: %w", models.ErrNoContent, err)
	}
	content := strings.TrimSpace(res.ContentText)
	if content == "" {
		return "", fmt.Errorf("%w: webpage has no extractable text", models.ErrNoContent)
	}
	return content, nil
}
