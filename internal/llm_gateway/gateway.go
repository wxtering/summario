package llmgateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"tldr/internal/models"
	"tldr/internal/utils"

	"github.com/sashabaranov/go-openai"
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

type LlmGateway struct{}

func New() *LlmGateway {
	return &LlmGateway{}
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

func (g *LlmGateway) GenerateSummary(ctx context.Context, p RequestParams) (LLMResponse, error) {
	if p.ProviderConfig == nil {
		p.ProviderConfig = &models.ProviderConfig{}
	}

	start := time.Now()

	var resp LLMResponse
	var err error

	switch p.ProviderConfig.Provider {
	case "openai":
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
	cfg := openai.DefaultConfig(p.ProviderConfig.ApiKey)
	cfg.BaseURL = p.ProviderConfig.Url
	client := openai.NewClientWithConfig(cfg)

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: p.ProviderConfig.Model,
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: p.SystemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: p.Text,
				},
			},
		},
	)
	if err != nil {
		return LLMResponse{}, mapOpenAIError(err)
	}

	if len(resp.Choices) == 0 {
		return LLMResponse{}, ErrEmptyResponse
	}

	content := resp.Choices[0].Message.Content
	cleanedContent := cleanJSONContent(content)
	var raw rawJSONResponse
	if err := json.Unmarshal([]byte(cleanedContent), &raw); err != nil || raw.Summary == "" {
		raw.Summary = content
	}

	return LLMResponse{
		Title:        raw.Title,
		Summary:      raw.Summary,
		Provider:     p.ProviderConfig.Provider,
		Model:        p.ProviderConfig.Model,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		TotalTokens:  resp.Usage.TotalTokens,
	}, nil
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

func mapOpenAIError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	// 1. OpenAI application errors
	if apiErr, ok := utils.AsType[*openai.APIError](err); ok {
		switch apiErr.HTTPStatusCode {
		case 401, 403:
			return ErrInvalidAPIKey
		case 404:
			return ErrModelNotFound
		case 429:
			if apiErr.Type == "insufficient_quota" || fmt.Sprint(apiErr.Code) == "insufficient_quota" {
				return ErrQuotaExceeded
			}
			return ErrRateLimit
		case 500, 502, 503, 504:
			return ErrProviderDown
		}
	}

	// 2. Network / infrastructure errors
	if reqErr, ok := utils.AsType[*openai.RequestError](err); ok {
		if reqErr.HTTPStatusCode >= 500 || reqErr.HTTPStatusCode == 429 {
			return ErrProviderDown
		}
	}

	return fmt.Errorf("%w: %w", ErrUnexpected, err)
}
