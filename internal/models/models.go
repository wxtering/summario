package models

type SourceType string

const (
	SourceAuto    SourceType = "auto"
	SourceYouTube SourceType = "youtube"
	SourceWebpage SourceType = "webpage"
	SourceGitHub  SourceType = "github"
	SourceRawText SourceType = "raw_text"
)

type Format string

const (
	FormatMarkdown  Format = "markdown"
	FormatPlainText Format = "plain_text"
	FormatHTML      Format = "html"
)

type Strategy string

const (
	StrategyTLDR         Strategy = "tldr"
	StrategyDetailed     Strategy = "detailed"
	StrategyBulletPoints Strategy = "bullet_points"
)

type SummarizeRequest struct {
	Source            string            `json:"source"`
	SourceType        SourceType        `json:"source_type"`
	Format            Format            `json:"format"`
	Strategy          Strategy          `json:"strategy"`
	Language          string            `json:"language"`
	ProviderConfig    *ProviderConfig   `json:"provider_config"`
	FallbackProviders []*ProviderConfig `json:"fallback_providers"`
}

type ProviderConfig struct {
	Provider string `json:"provider"`
	ApiKey   string `json:"api_key"`
	Url      string `json:"url"`
	Model    string `json:"model"`
}

type SummarizeResponse struct {
	Summary    string        `json:"summary"`
	Title      string        `json:"title"`
	SourceType SourceType    `json:"source_type"`
	Source     string        `json:"source"`
	Meta       ExecutionMeta `json:"meta"`
}

type ExecutionMeta struct {
	Language     string   `json:"language"`
	Format       Format   `json:"format"`
	Strategy     Strategy `json:"strategy"`
	Provider     string   `json:"provider"`
	Model        string   `json:"model"`
	InputTokens  int      `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
	TotalTokens  int      `json:"total_tokens"`
	DurationMs   int64    `json:"duration_ms"`
}
