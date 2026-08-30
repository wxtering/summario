package config

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Server     ServerConfig
	LLM        LLMConfig
	Parser     ParserConfig
	Summarizer SummarizerConfig
}

type ServerConfig struct {
	Port     string `env:"PORT" env-default:"8080"`
	LogLevel string `env:"LOG_LEVEL" env-default:"info"`
}

type LLMConfig struct {
	DefaultKey      string `env:"DEFAULT_API_KEY"`
	DefaultURL      string `env:"DEFAULT_URL"`
	DefaultModel    string `env:"DEFAULT_MODEL" env-default:"gpt-4o-mini"`
	DefaultProvider string `env:"DEFAULT_PROVIDER" env-default:"openai"`
}

type SummarizerConfig struct {
	DefaultLanguage string `env:"DEFAULT_LANGUAGE" env-default:"en"`
	DefaultFormat   string `env:"DEFAULT_FORMAT" env-default:"markdown"`
	DefaultStrategy string `env:"DEFAULT_STRATEGY" env-default:"tldr"`
}

type ParserConfig struct {
	Firecrawl   FirecrawlConfig
	GitHubToken string `env:"GITHUB_TOKEN"`
}

type FirecrawlConfig struct {
	APIKey   string `env:"FIRECRAWL_API_KEY"`
	BaseURL  string `env:"FIRECRAWL_BASE_URL" env-default:"https://api.firecrawl.dev"`
	Fallback bool   `env:"FIRECRAWL_FALLBACK" env-default:"true"`
	Prefer   bool   `env:"PREFER_FIRECRAWL" env-default:"false"`
}

func LoadConfig() (*Config, error) {
	// .env only fills in variables missing from the real environment:
	// env vars must take precedence over the file.
	if err := godotenv.Load(".env"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("failed to read .env: %w", err)
	}

	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &cfg, nil
}
