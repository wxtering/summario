package github

import (
	"context"
	"fmt"
	"net/http"
	"tldr/internal/models"

	gh "github.com/google/go-github/v90/github"
)

// Priority order for instruction files in repository
var agentsCandidates = []string{
	"AGENTS.md",
	"CONTEXT.md",
	"CLAUDE.md",
}

type GitHubParser struct {
	ghClient *gh.Client
}

func NewGitHubParser(token string) (*GitHubParser, error) {
	var opts []gh.ClientOptionsFunc

	if token != "" {
		httpClient := &http.Client{
			Transport: &transportWithToken{
				token:     token,
				transport: http.DefaultTransport,
			},
		}
		opts = append(opts, gh.WithHTTPClient(httpClient))
	}

	client, err := gh.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create github client: %w", err)
	}

	return &GitHubParser{
		ghClient: client,
	}, nil
}

// transportWithToken is an http.RoundTripper decorator that binds the GitHub Bearer token into outgoing client requests.
type transportWithToken struct {
	token     string
	transport http.RoundTripper
}

func (t *transportWithToken) RoundTrip(req *http.Request) (*http.Response, error) {
	reqCopy := req.Clone(req.Context())
	reqCopy.Header.Set("Authorization", "Bearer "+t.token)
	return t.transport.RoundTrip(reqCopy)
}

// GitHubResult holds extracted repo info.
type GitHubResult struct {
	Title           string
	Description     string
	PrimaryLanguage string
	Topics          []string
	FileTree        string
	Readme          string
	Agents          string
}

// FetchRepo fetches repository metadata, README, AGENTS.md/CONTEXT.md, and file tree.
func (p *GitHubParser) FetchRepo(ctx context.Context, source string) (*GitHubResult, error) {
	owner, repo, err := ExtractRepoInfo(source)
	if err != nil {
		return nil, err
	}

	// 1. Fetch Repository Metadata
	repoData, resp, err := p.ghClient.Repositories.Get(ctx, owner, repo)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: github repository %s/%s", models.ErrNotFound, owner, repo)
		}
		if resp != nil && (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests) {
			return nil, fmt.Errorf("%w: github api rate limit", models.ErrRateLimitExceeded)
		}
		return nil, fmt.Errorf("%w: failed to fetch repo metadata: %w", models.ErrUpstreamFailed, err)
	}

	res := &GitHubResult{
		Title:           fmt.Sprintf("%s/%s", owner, repo),
		Description:     repoData.GetDescription(),
		PrimaryLanguage: repoData.GetLanguage(),
		Topics:          repoData.Topics,
	}

	// 2. Fetch README
	readme, _, err := p.ghClient.Repositories.GetReadme(ctx, owner, repo, nil)
	if err == nil && readme != nil {
		content, err := readme.GetContent()
		if err == nil {
			res.Readme = content
		}
	}

	// 3. Fetch Root File Tree (non-recursive, light on RAM)
	defaultBranch := repoData.GetDefaultBranch()
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	var treeEntries []*gh.TreeEntry
	tree, _, err := p.ghClient.Git.GetTree(ctx, owner, repo, defaultBranch, false)
	if err == nil && tree != nil {
		treeEntries = tree.Entries
		res.FileTree = BuildFileTreeText(treeEntries, 2)
	}

	// 4. Search for AGENTS.md / CONTEXT.md instructions
	targetAgentPath := findAgentCandidateInTree(treeEntries, agentsCandidates)
	if targetAgentPath != "" {
		fileContent, _, _, err := p.ghClient.Repositories.GetContents(ctx, owner, repo, targetAgentPath, nil)
		if err == nil && fileContent != nil {
			content, err := fileContent.GetContent()
			if err == nil && content != "" {
				res.Agents = content
			}
		}
	}

	return res, nil
}
