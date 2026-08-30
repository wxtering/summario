package resolver_test

import (
	"errors"
	"testing"
	"tldr/internal/models"
	"tldr/internal/resolver"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		request     models.SummarizeRequest
		expected    models.SourceType
		expectError bool
	}{
		// Happy Path - YouTube
		{
			name:        "YouTube short URL (youtu.be)",
			request:     models.SummarizeRequest{Source: "https://youtu.be/dQw4w9WgXcQ"},
			expected:    models.SourceYouTube,
			expectError: false,
		},
		{
			name:        "YouTube full URL (www.youtube.com)",
			request:     models.SummarizeRequest{Source: "https://www.youtube.com/watch?v=dQw4w9WgXcQ"},
			expected:    models.SourceYouTube,
			expectError: false,
		},
		{
			name:        "YouTube mobile URL (m.youtube.com)",
			request:     models.SummarizeRequest{Source: "https://m.youtube.com/watch?v=dQw4w9WgXcQ"},
			expected:    models.SourceYouTube,
			expectError: false,
		},

		// Happy Path - GitHub
		{
			name:        "GitHub repository URL",
			request:     models.SummarizeRequest{Source: "https://github.com/torvalds/linux"},
			expected:    models.SourceGitHub,
			expectError: false,
		},

		// Happy Path - Web (including Reddit as webpage)
		{
			name:        "Reddit post URL (resolved as Webpage)",
			request:     models.SummarizeRequest{Source: "https://www.reddit.com/r/NixOS/comments/1w050g9/switched_from_nixos_to_arch_but/"},
			expected:    models.SourceWebpage,
			expectError: false,
		},
		{
			name:        "Regular web article URL",
			request:     models.SummarizeRequest{Source: "https://habr.com/ru/articles/123456/"},
			expected:    models.SourceWebpage,
			expectError: false,
		},

		// Happy Path - Raw Text
		{
			name:        "Raw text input without URL scheme",
			request:     models.SummarizeRequest{Source: "Привет! Сделай мне краткую выжимку этого текста."},
			expected:    models.SourceRawText,
			expectError: false,
		},

		// Explicit Override
		{
			name: "Explicit override takes precedence over URL detection",
			request: models.SummarizeRequest{
				Source:     "https://youtu.be/dQw4w9WgXcQ",
				SourceType: models.SourceRawText,
			},
			expected:    models.SourceRawText,
			expectError: false,
		},

		// Bad Path - Empty / Whitespace
		{
			name:        "Empty string returns error",
			request:     models.SummarizeRequest{Source: ""},
			expected:    "",
			expectError: true,
		},
		{
			name:        "Only whitespaces return error",
			request:     models.SummarizeRequest{Source: "    \t\n   "},
			expected:    "",
			expectError: true,
		},

		// Edge Cases - Phishing / Similar domains
		{
			name:        "Phishing domain (notyoutube.com) detected as Webpage",
			request:     models.SummarizeRequest{Source: "https://notyoutube.com/watch?v=123"},
			expected:    models.SourceWebpage,
			expectError: false,
		},
		{
			name:        "Fake GitHub domain (fake-github.com) detected as Webpage",
			request:     models.SummarizeRequest{Source: "https://fake-github.com/torvalds/linux"},
			expected:    models.SourceWebpage,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.Resolve(tt.request)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, models.ErrEmptySource) {
					t.Fatalf("expected models.ErrEmptySource, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Fatalf("Resolve() = %v, expected %v", got, tt.expected)
			}
		})
	}
}
