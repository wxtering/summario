package llmgateway_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	llmgateway "tldr/internal/llm_gateway"
	"tldr/internal/models"
)

func TestLlmGateway_OpenCode(t *testing.T) {
	var receivedHeaders http.Header
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		bodyBytes, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bodyBytes, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "{\"title\": \"OpenCode Title\", \"summary\": \"OpenCode Summary\"}"
					},
					"finish_reason": "stop"
				}
			],
			"usage": {
				"prompt_tokens": 12,
				"completion_tokens": 24,
				"total_tokens": 36
			}
		}`))
	}))
	defer server.Close()

	gw := llmgateway.New()

	t.Run("OpenCode with custom session and headers", func(t *testing.T) {
		params := llmgateway.RequestParams{
			ProviderConfig: &models.ProviderConfig{
				Provider: llmgateway.ProviderOpenCode,
				ApiKey:   "oc_sk_test_key",
				Url:      server.URL,
				Model:    "deepseek-v4.1-flash",
				Session:  "ses_test-conversation-001",
				Headers: map[string]string{
					"X-Custom-Test": "value-123",
				},
			},
			SystemPrompt: "test system prompt",
			Text:         "test user content",
		}

		resp, err := gw.GenerateSummary(context.Background(), params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Title != "OpenCode Title" {
			t.Errorf("expected title 'OpenCode Title', got %q", resp.Title)
		}
		if resp.Summary != "OpenCode Summary" {
			t.Errorf("expected summary 'OpenCode Summary', got %q", resp.Summary)
		}
		if resp.Provider != llmgateway.ProviderOpenCode {
			t.Errorf("expected provider %q, got %q", llmgateway.ProviderOpenCode, resp.Provider)
		}
		if resp.Model != "deepseek-v4.1-flash" {
			t.Errorf("expected model 'deepseek-v4.1-flash', got %q", resp.Model)
		}

		if auth := receivedHeaders.Get("Authorization"); auth != "Bearer oc_sk_test_key" {
			t.Errorf("expected Authorization 'Bearer oc_sk_test_key', got %q", auth)
		}
		if session := receivedHeaders.Get("x-opencode-session"); session != "ses_test-conversation-001" {
			t.Errorf("expected x-opencode-session 'ses_test-conversation-001', got %q", session)
		}
		if custom := receivedHeaders.Get("X-Custom-Test"); custom != "value-123" {
			t.Errorf("expected X-Custom-Test 'value-123', got %q", custom)
		}

		if _, hasRespFormat := receivedBody["response_format"]; hasRespFormat {
			t.Errorf("expected OpenCode request not to have response_format, but got %v", receivedBody["response_format"])
		}
	})

	t.Run("OpenCode generates session when empty", func(t *testing.T) {
		params := llmgateway.RequestParams{
			ProviderConfig: &models.ProviderConfig{
				Provider: llmgateway.ProviderOpenCode,
				ApiKey:   "oc_sk_test_key",
				Url:      server.URL,
			},
			SystemPrompt: "test prompt",
			Text:         "test content",
		}

		resp, err := gw.GenerateSummary(context.Background(), params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Model != llmgateway.DefaultOpenCodeModel {
			t.Errorf("expected default model %q, got %q", llmgateway.DefaultOpenCodeModel, resp.Model)
		}

		session := receivedHeaders.Get("x-opencode-session")
		if !strings.HasPrefix(session, "ses_") {
			t.Errorf("expected generated session starting with 'ses_', got %q", session)
		}
	})

	t.Run("Unsupported provider returns error", func(t *testing.T) {
		params := llmgateway.RequestParams{
			ProviderConfig: &models.ProviderConfig{
				Provider: "unsupported-provider",
				ApiKey:   "key",
			},
		}

		_, err := gw.GenerateSummary(context.Background(), params)
		if err == nil || !strings.Contains(err.Error(), "unsupported llm provider") {
			t.Fatalf("expected unsupported provider error, got %v", err)
		}
	})
}
