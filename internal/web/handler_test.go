package web_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"tldr/internal/config"
	"tldr/internal/web"
)

func TestHandler_HandleUrl_Errors(t *testing.T) {
	cfg := &config.Config{}
	container := web.NewContainer(cfg)
	handler := container.Handler

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "Malformed JSON payload returns 400 INVALID_URL",
			body:           `{"source": `,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_URL",
		},
		{
			name:           "Empty source field returns 400 EMPTY_SOURCE",
			body:           `{"source": "   "}`,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "EMPTY_SOURCE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tldr", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.HandleUrl(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("status code = %d, expected %d. Body: %s", w.Code, tt.expectedStatus, w.Body.String())
			}

			var errResp web.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("failed to unmarshal JSON error response: %v. Raw body: %s", err, w.Body.String())
			}

			if errResp.Error.Code != tt.expectedCode {
				t.Errorf("error.code = %q, expected %q", errResp.Error.Code, tt.expectedCode)
			}
			if errResp.Error.Message == "" {
				t.Errorf("error.message is empty, expected meaningful message")
			}
		})
	}
}
