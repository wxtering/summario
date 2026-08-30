package parser

import (
	"context"
	"fmt"
	"tldr/internal/config"
	"tldr/internal/models"
	"tldr/internal/parser/web"
)

// Parser routes a source to the concrete extractor by its type.
type Parser struct {
	webParser *web.WebParser
}

func NewParser(cfg *config.Config) *Parser {
	return &Parser{
		webParser: web.NewWebParser(cfg),
	}
}

func (p *Parser) Parse(ctx context.Context, sourceType models.SourceType, source string, lang string) (string, error) {
	switch sourceType {
	case models.SourceYouTube:
		return p.webParser.ParseYouTube(ctx, source, lang)
	case models.SourceGitHub:
		return p.webParser.ParseGitHub(ctx, source)
	case models.SourceWebpage:
		return p.webParser.ParseWebPage(ctx, source)
	case models.SourceRawText:
		return source, nil
	default:
		return "", fmt.Errorf("%w: %s", models.ErrUnsupportedSource, sourceType)
	}
}
