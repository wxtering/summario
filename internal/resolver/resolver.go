package resolver

import (
	"net/url"
	"strings"
	"tldr/internal/models"
)

var youtubeDomains = map[string]struct{}{
	"youtube.com":   {},
	"m.youtube.com": {},
	"youtu.be":      {},
}

var githubDomains = map[string]struct{}{
	"github.com": {},
}

func Resolve(request models.SummarizeRequest) (models.SourceType, error) {
	source := strings.TrimSpace(request.Source)
	if source == "" {
		return "", models.ErrEmptySource
	}

	// source_type overrides auto-detection
	if request.SourceType != "" && request.SourceType != models.SourceAuto {
		return request.SourceType, nil
	}

	return resolveType(source), nil
}

func resolveType(source string) models.SourceType {
	u, err := url.Parse(source)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return models.SourceRawText
	}

	host := getHost(u)
	switch {
	case isYoutubeDomain(host):
		return models.SourceYouTube
	case isGithubDomain(host):
		return models.SourceGitHub
	default:
		return models.SourceWebpage
	}
}

func getHost(source *url.URL) string {
	host := strings.ToLower(source.Hostname())
	return strings.TrimPrefix(host, "www.")
}

func isYoutubeDomain(host string) bool {
	_, ok := youtubeDomains[host]
	return ok
}

func isGithubDomain(host string) bool {
	_, ok := githubDomains[host]
	return ok
}
