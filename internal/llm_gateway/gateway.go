package llmgateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"tldr/internal/models"

	"resty.dev/v3"
)

const (
	ProviderOpenAI   = "openai"
	ProviderOpenCode = "opencode"
)

const (
	DefaultOpenAIBaseURL   = "https://api.openai.com/v1"
	DefaultOpenCodeBaseURL = "https://opencode.ai/inference/go/openai/v1"
	DefaultOpenCodeModel   = "deepseek-v4.1-flash"
)

var (
	ErrAPIKeyRequired      = errors.New("api key is required")
	ErrInvalidAPIKey       = errors.New("invalid or unauthorized api key")
	ErrQuotaExceeded       = errors.New("quota or balance exceeded")
	ErrRateLimit           = errors.New("rate limit exceeded")
	ErrModelNotFound       = errors.New("requested model not found")
	ErrProviderDown        = errors.New("llm provider is unavailable")
	ErrEmptyResponse       = errors.New("empty response from llm")
	ErrUnexpected          = errors.New("unexpected llm error")
	ErrUnsupportedProvider = errors.New("unsupported llm provider")
)

type LlmGateway struct {
	client *resty.Client
}

func New() *LlmGateway {
	client := resty.New().
		SetTimeout(120 * time.Second).
		SetRetryCount(1).
		SetRetryWaitTime(500 * time.Millisecond)

	return &LlmGateway{
		client: client,
	}
}

type RequestParams struct {
	ProviderConfig    *models.ProviderConfig
	FallbackProviders []*models.ProviderConfig
	SystemPrompt      string
	Text              string
}

type LLMResponse struct {
	Title        string
	Summary      string
	Provider     string
	Model        string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	DurationMs   int64
}

type rawJSONResponse struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

type chatCompletionRequest struct {
	Model          string                  `json:"model"`
	ResponseFormat *chatResponseFormatType `json:"response_format,omitempty"`
	Messages       []chatCompletionMessage `json:"messages"`
}

type chatResponseFormatType struct {
	Type string `json:"type"`
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []Choice           `json:"choices"`
	Usage   Usage              `json:"usage"`
	Error   *openAIErrorDetail `json:"error,omitempty"`
}

type Choice struct {
	Index        int                   `json:"index"`
	Message      chatCompletionMessage `json:"message"`
	FinishReason string                `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code"`
}

func (g *LlmGateway) GenerateSummary(ctx context.Context, p RequestParams) (LLMResponse, error) {
	if p.ProviderConfig == nil {
		p.ProviderConfig = &models.ProviderConfig{}
	}

	start := time.Now()

	var resp LLMResponse
	var err error

	switch p.ProviderConfig.Provider {
	case ProviderOpenAI, ProviderOpenCode:
		resp, err = g.openAIRequest(ctx, p)
	default:
		return LLMResponse{}, fmt.Errorf("%w: %s", ErrUnsupportedProvider, p.ProviderConfig.Provider)
	}

	if err != nil {
		return LLMResponse{}, err
	}

	resp.DurationMs = time.Since(start).Milliseconds()
	return resp, nil
}

func (g *LlmGateway) openAIRequest(ctx context.Context, p RequestParams) (LLMResponse, error) {
	if p.ProviderConfig.ApiKey == "" {
		return LLMResponse{}, ErrAPIKeyRequired
	}

	model := p.ProviderConfig.Model
	if model == "" && p.ProviderConfig.Provider == ProviderOpenCode {
		model = DefaultOpenCodeModel
	}

	baseURL := strings.TrimRight(p.ProviderConfig.Url, "/")
	if baseURL == "" {
		if p.ProviderConfig.Provider == ProviderOpenCode {
			baseURL = DefaultOpenCodeBaseURL
		} else {
			baseURL = DefaultOpenAIBaseURL
		}
	}
	endpointURL := baseURL + "/chat/completions"

	reqBody := chatCompletionRequest{
		Model: model,
		Messages: []chatCompletionMessage{
			{
				Role:    "system",
				Content: p.SystemPrompt,
			},
			{
				Role:    "user",
				Content: p.Text,
			},
		},
	}
	if p.ProviderConfig.Provider != ProviderOpenCode {
		reqBody.ResponseFormat = &chatResponseFormatType{
			Type: "json_object",
		}
	}

	req := g.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+p.ProviderConfig.ApiKey)

	if p.ProviderConfig.Provider == ProviderOpenCode {
		sessionID := p.ProviderConfig.Session
		if sessionID == "" {
			sessionID = generateSessionID()
		}
		req.SetHeader("x-opencode-session", sessionID)
	} else if p.ProviderConfig.Session != "" {
		req.SetHeader("x-opencode-session", p.ProviderConfig.Session)
	}

	for k, v := range p.ProviderConfig.Headers {
		req.SetHeader(k, v)
	}

	var chatResp chatCompletionResponse
	resp, err := req.
		SetBody(reqBody).
		SetResult(&chatResp).
		Post(endpointURL)

	if err != nil {
		return LLMResponse{}, fmt.Errorf("%w: %w", ErrProviderDown, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return LLMResponse{}, mapHTTPError(resp.StatusCode(), chatResp.Error)
	}

	if len(chatResp.Choices) == 0 {
		return LLMResponse{}, ErrEmptyResponse
	}

	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if content == "" {
		return LLMResponse{}, fmt.Errorf("%w: llm generated empty content", ErrEmptyResponse)
	}

	cleanedContent := cleanJSONContent(content)
	var raw rawJSONResponse
	if err := json.Unmarshal([]byte(cleanedContent), &raw); err != nil || strings.TrimSpace(raw.Summary) == "" {
		raw.Summary = content
	}

	return LLMResponse{
		Title:        raw.Title,
		Summary:      raw.Summary,
		Provider:     p.ProviderConfig.Provider,
		Model:        model,
		InputTokens:  chatResp.Usage.PromptTokens,
		OutputTokens: chatResp.Usage.CompletionTokens,
		TotalTokens:  chatResp.Usage.TotalTokens,
	}, nil
}

func generateSessionID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("ses_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("ses_%s", hex.EncodeToString(b))
}

// cleanJSONContent strips markdown code block fences (e.g. ```json ... ```)
// from the raw LLM output before parsing it with json.Unmarshal.
func cleanJSONContent(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		if idx := strings.Index(content, "\n"); idx != -1 {
			content = content[idx+1:]
		}
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	return content
}

func mapHTTPError(statusCode int, errBody *openAIErrorDetail) error {
	var errType, errCode, errMsg string
	if errBody != nil {
		errType = errBody.Type
		errMsg = errBody.Message
		if errBody.Code != nil {
			errCode = fmt.Sprint(errBody.Code)
		}
	}

	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrInvalidAPIKey
	case http.StatusNotFound:
		return ErrModelNotFound
	case http.StatusTooManyRequests:
		if errType == "insufficient_quota" || errCode == "insufficient_quota" || strings.Contains(errMsg, "quota") {
			return ErrQuotaExceeded
		}
		return ErrRateLimit
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return ErrProviderDown
	default:
		return fmt.Errorf("%w: status %d: %s", ErrUnexpected, statusCode, errMsg)
	}
}
