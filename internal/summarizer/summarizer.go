package summarizer

import (
	"context"
	"log/slog"
	"time"
	"tldr/internal/config"
	llmgateway "tldr/internal/llm_gateway"
	"tldr/internal/models"
	"tldr/internal/parser"
	"tldr/internal/prompt"
	"tldr/internal/resolver"
)

type Summarizer struct {
	parser *parser.Parser
	llm    *llmgateway.LlmGateway
	cfg    *config.Config
}

func New(parser *parser.Parser, llm *llmgateway.LlmGateway, cfg *config.Config) *Summarizer {
	return &Summarizer{
		parser: parser,
		llm:    llm,
		cfg:    cfg,
	}
}

func (s *Summarizer) applyDefaults(r *models.SummarizeRequest) {
	if s.cfg == nil {
		return
	}
	if r.Language == "" {
		r.Language = s.cfg.Summarizer.DefaultLanguage
	}
	if r.Format == "" {
		r.Format = models.Format(s.cfg.Summarizer.DefaultFormat)
	}
	if r.Strategy == "" {
		r.Strategy = models.Strategy(s.cfg.Summarizer.DefaultStrategy)
	}

	if r.ProviderConfig == nil {
		r.ProviderConfig = &models.ProviderConfig{}
	}
	if r.ProviderConfig.Provider == "" {
		r.ProviderConfig.Provider = s.cfg.LLM.DefaultProvider
	}
	if r.ProviderConfig.ApiKey == "" {
		r.ProviderConfig.ApiKey = s.cfg.LLM.DefaultKey
	}
	if r.ProviderConfig.Url == "" {
		r.ProviderConfig.Url = s.cfg.LLM.DefaultURL
	}
	if r.ProviderConfig.Model == "" {
		r.ProviderConfig.Model = s.cfg.LLM.DefaultModel
	}
}

func (s *Summarizer) Summarize(ctx context.Context, data models.SummarizeRequest) (*models.SummarizeResponse, error) {
	totalStart := time.Now()
	s.applyDefaults(&data)

	sourceType, err := resolver.Resolve(data)
	if err != nil {
		return nil, err
	}

	parseStart := time.Now()
	text, err := s.parser.Parse(ctx, sourceType, data.Source, data.Language)
	parseDuration := time.Since(parseStart)
	if err != nil {
		slog.Error("parser failed",
			slog.String("source_type", string(sourceType)),
			slog.String("source", data.Source),
			slog.Duration("duration", parseDuration),
			slog.Any("error", err),
		)
		return nil, err
	}

	slog.Debug("parser completed",
		slog.String("source_type", string(sourceType)),
		slog.String("source", data.Source),
		slog.Int("chars_count", len(text)),
		slog.Duration("duration", parseDuration),
	)

	slog.Debug("raw parser content",
		slog.String("source_type", string(sourceType)),
		slog.String("source", data.Source),
		slog.String("content", text),
	)

	params := llmgateway.RequestParams{
		ProviderConfig:    data.ProviderConfig,
		FallbackProviders: data.FallbackProviders,
		SystemPrompt:      prompt.BuildPrompt(data.Format, data.Language, data.Strategy),
		Text:              text,
	}

	llmResp, err := s.llm.GenerateSummary(ctx, params)
	if err != nil {
		slog.Error("llm summary generation failed",
			slog.String("provider", data.ProviderConfig.Provider),
			slog.String("model", data.ProviderConfig.Model),
			slog.Any("error", err),
		)
		return nil, err
	}

	totalDuration := time.Since(totalStart)
	slog.Info("llm summary generated",
		slog.String("provider", llmResp.Provider),
		slog.String("model", llmResp.Model),
		slog.Int("input_tokens", llmResp.InputTokens),
		slog.Int("output_tokens", llmResp.OutputTokens),
		slog.Int("total_tokens", llmResp.TotalTokens),
		slog.Int64("llm_duration_ms", llmResp.DurationMs),
		slog.Int64("total_duration_ms", totalDuration.Milliseconds()),
	)

	return &models.SummarizeResponse{
		Summary:    llmResp.Summary,
		Title:      llmResp.Title,
		SourceType: sourceType,
		Source:     data.Source,
		Meta: models.ExecutionMeta{
			Language:     data.Language,
			Format:       data.Format,
			Strategy:     data.Strategy,
			Provider:     llmResp.Provider,
			Model:        llmResp.Model,
			InputTokens:  llmResp.InputTokens,
			OutputTokens: llmResp.OutputTokens,
			TotalTokens:  llmResp.TotalTokens,
			DurationMs:   totalDuration.Milliseconds(),
		},
	}, nil
}
