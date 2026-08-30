package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"tldr/internal/config"
	llmgateway "tldr/internal/llm_gateway"
	"tldr/internal/models"
	"tldr/internal/summarizer"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Handler struct {
	summarizer *summarizer.Summarizer
	cfg        *config.Config
}

func NewHandler(s *summarizer.Summarizer, cfg *config.Config) *Handler {
	return &Handler{
		summarizer: s,
		cfg:        cfg,
	}
}

func (h *Handler) HandleUrl(w http.ResponseWriter, r *http.Request) {
	var data models.SummarizeRequest
	if err := decodeJSON(r, &data); err != nil {
		writeError(w, fmt.Errorf("%w: %w", models.ErrInvalidURL, err))
		return
	}

	result, err := h.summarizer.Summarize(r.Context(), data)
	if err != nil {
		writeError(w, err)
		return
	}

	writeResponse(w, http.StatusOK, result)
}

func (h *Handler) HandleText(w http.ResponseWriter, r *http.Request) {
	// TODO: Handle incoming request
}

func (h *Handler) HandleFile(w http.ResponseWriter, r *http.Request) {
	// TODO: Handle incoming request
}

func writeResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("encode response failed", slog.Any("error", err))
	}
}

func writeError(w http.ResponseWriter, err error) {
	var statusCode int
	var code, message string

	switch {
	// 400 Bad Request
	case errors.Is(err, models.ErrEmptySource):
		statusCode = http.StatusBadRequest
		code = "EMPTY_SOURCE"
		message = "Source cannot be empty"

	case errors.Is(err, models.ErrInvalidURL):
		statusCode = http.StatusBadRequest
		code = "INVALID_URL"
		message = "Provided URL is invalid or malformed"

	case errors.Is(err, models.ErrUnsupportedSource):
		statusCode = http.StatusBadRequest
		code = "UNSUPPORTED_SOURCE"
		message = "Source type is not supported"

	case errors.Is(err, llmgateway.ErrUnsupportedProvider):
		statusCode = http.StatusBadRequest
		code = "UNSUPPORTED_PROVIDER"
		message = "Requested LLM provider is not supported"

	// 401 Unauthorized
	case errors.Is(err, llmgateway.ErrAPIKeyRequired), errors.Is(err, llmgateway.ErrInvalidAPIKey):
		statusCode = http.StatusUnauthorized
		code = "UNAUTHORIZED"
		message = "LLM provider API key is missing or invalid"

	// 403 Forbidden
	case errors.Is(err, models.ErrRestricted):
		statusCode = http.StatusForbidden
		code = "RESTRICTED"
		message = "Requested resource is private, restricted or unplayable"

	// 404 Not Found
	case errors.Is(err, models.ErrNotFound), errors.Is(err, llmgateway.ErrModelNotFound):
		statusCode = http.StatusNotFound
		code = "NOT_FOUND"
		message = "Requested resource or model was not found"

	// 422 Unprocessable Entity
	case errors.Is(err, models.ErrNoContent), errors.Is(err, llmgateway.ErrEmptyResponse):
		statusCode = http.StatusUnprocessableEntity
		code = "NO_CONTENT"
		message = "No extractable text or content found in source"

	// 429 Too Many Requests
	case errors.Is(err, models.ErrRateLimitExceeded), errors.Is(err, llmgateway.ErrRateLimit):
		statusCode = http.StatusTooManyRequests
		code = "RATE_LIMIT_EXCEEDED"
		message = "Rate limit exceeded, please retry later"

	case errors.Is(err, llmgateway.ErrQuotaExceeded):
		statusCode = http.StatusTooManyRequests
		code = "QUOTA_EXCEEDED"
		message = "LLM provider quota or balance exceeded"

	// 502 Bad Gateway
	case errors.Is(err, models.ErrUpstreamFailed), errors.Is(err, llmgateway.ErrProviderDown):
		statusCode = http.StatusBadGateway
		code = "UPSTREAM_FAILED"
		message = "Failed to communicate with remote website or LLM provider"

	// 500 Internal Server Error (Fallback)
	default:
		statusCode = http.StatusInternalServerError
		code = "INTERNAL_SERVER_ERROR"
		message = "An unexpected internal server error occurred"
	}

	slog.Error("http request error",
		slog.Int("status_code", statusCode),
		slog.String("code", code),
		slog.Any("err", err),
	)

	writeResponse(w, statusCode, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
