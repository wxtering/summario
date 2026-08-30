package web

import (
	"context"
	"time"

	"tldr/internal/config"
	"tldr/internal/parser/web/firecrawl"
	ghparser "tldr/internal/parser/web/github"
	ytparser "tldr/internal/parser/web/youtube"

	"resty.dev/v3"
)

type WebParser struct {
	client          *resty.Client
	ytParser        *ytparser.YtParser
	ghParser        *ghparser.GitHubParser
	firecrawlClient *firecrawl.Client
	preferFirecrawl bool
	fallbackEnabled bool
}

func NewWebParser(cfg *config.Config) *WebParser {
	client := resty.New().
		SetTimeout(30 * time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(500 * time.Millisecond)

	var githubToken string
	var fcClient *firecrawl.Client
	var fallbackEnabled = true
	var preferFirecrawl = false

	if cfg != nil {
		githubToken = cfg.Parser.GitHubToken
		fcClient = firecrawl.NewClient(cfg.Parser.Firecrawl.BaseURL, cfg.Parser.Firecrawl.APIKey)
		fallbackEnabled = cfg.Parser.Firecrawl.Fallback
		preferFirecrawl = cfg.Parser.Firecrawl.Prefer
	} else {
		fcClient = firecrawl.NewClient("", "")
	}

	ghP, _ := ghparser.NewGitHubParser(githubToken)

	return &WebParser{
		client:          client,
		ytParser:        ytparser.NewYtParser(),
		ghParser:        ghP,
		firecrawlClient: fcClient,
		preferFirecrawl: preferFirecrawl,
		fallbackEnabled: fallbackEnabled,
	}
}

func (w *WebParser) ParseYouTube(ctx context.Context, source string, lang string) (string, error) {
	res, err := w.ytParser.FetchTranscript(ctx, source, lang)
	if err != nil {
		return "", err
	}
	return res.Transcript, nil
}

func (w *WebParser) ParseGitHub(ctx context.Context, source string) (string, error) {
	res, err := w.ghParser.FetchRepo(ctx, source)
	if err != nil {
		return "", err
	}
	return res.Stringify(), nil
}
