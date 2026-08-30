package github

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"tldr/internal/models"

	gh "github.com/google/go-github/v90/github"
)

// findAgentCandidateInTree checks if any candidate instruction file exists in tree entries.
func findAgentCandidateInTree(entries []*gh.TreeEntry, candidates []string) string {
	if len(entries) == 0 {
		return candidates[0]
	}

	existingPaths := make(map[string]bool, len(entries))
	for _, entry := range entries {
		existingPaths[entry.GetPath()] = true
	}

	for _, candidate := range candidates {
		if existingPaths[candidate] {
			return candidate
		}
	}
	return ""
}

// ExtractRepoInfo parses a raw URL or string to get owner and repository name.
func ExtractRepoInfo(source string) (owner string, repo string, err error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", "", models.ErrInvalidURL
	}

	// Add scheme if missing
	if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
		source = "https://" + source
	}

	u, err := url.Parse(source)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", models.ErrInvalidURL, err)
	}

	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")

	if host != "github.com" {
		return "", "", fmt.Errorf("%w: host %s is not github.com", models.ErrInvalidURL, host)
	}

	p := strings.Trim(u.Path, "/")
	p = strings.TrimSuffix(p, ".git")
	parts := strings.Split(p, "/")

	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("%w: invalid github repo path %s", models.ErrInvalidURL, p)
	}

	return parts[0], parts[1], nil
}

// BuildFileTreeText converts GitHub tree entries into a formatted tree representation up to maxDepth.
func BuildFileTreeText(entries []*gh.TreeEntry, maxDepth int) string {
	if len(entries) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, entry := range entries {
		p := entry.GetPath()
		if p == "" {
			continue
		}

		depth := strings.Count(p, "/") + 1
		if depth > maxDepth {
			continue
		}

		indent := strings.Repeat("  ", depth-1)
		name := path.Base(p)
		if entry.GetType() == "tree" {
			fmt.Fprintf(&sb, "%s%s/\n", indent, name)
		} else {
			fmt.Fprintf(&sb, "%s%s\n", indent, name)
		}
	}

	return strings.TrimSpace(sb.String())
}

// Stringify formats the GitHubResult into XML-tagged representation.
func (r *GitHubResult) Stringify() string {
	var sb strings.Builder
	sb.WriteString("<github_repository>\n")

	if r.Title != "" {
		fmt.Fprintf(&sb, "<name>%s</name>\n", r.Title)
	}
	if r.Description != "" {
		fmt.Fprintf(&sb, "<description>%s</description>\n", r.Description)
	}
	if r.PrimaryLanguage != "" {
		fmt.Fprintf(&sb, "<primary_language>%s</primary_language>\n", r.PrimaryLanguage)
	}
	if len(r.Topics) > 0 {
		fmt.Fprintf(&sb, "<topics>%s</topics>\n", strings.Join(r.Topics, ", "))
	}
	if r.FileTree != "" {
		fmt.Fprintf(&sb, "<file_tree>\n%s\n</file_tree>\n", r.FileTree)
	}
	if r.Agents != "" {
		fmt.Fprintf(&sb, "<agents_instructions>\n%s\n</agents_instructions>\n", r.Agents)
	}
	if r.Readme != "" {
		fmt.Fprintf(&sb, "<readme>%s\n</readme>\n", r.Readme)
	}

	sb.WriteString("</github_repository>")
	return sb.String()
}

func (r *GitHubResult) String() string {
	return r.Stringify()
}
