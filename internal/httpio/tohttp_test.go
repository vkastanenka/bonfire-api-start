package httpio_test

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToHTTP(t *testing.T) {
	tests := []struct {
		name           string
		handlerFunc    func(w http.ResponseWriter, r *http.Request) error
		expectedStatus int
		expectedCode   string // Empty if we don't expect an apperr response
		expectedBody   string // Partial match for the message
	}{
		{
			name: "Success - Handler executes and writes response",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) error {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
				return nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Error - Domain Error (Known)",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) error {
				return apperr.New(apperr.CodeNotFound, "user not found")
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND", // Assumes CodeNotFound stringifies to this
			expectedBody:   "user not found",
		},
		{
			name: "Error - Unknown Error (Normalization)",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) error {
				return errors.New("database connection failed")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Setup Request with Context
			req := httptest.NewRequest("GET", "/test", nil)
			reqID := "test-request-id-123"
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, reqID)
			req = req.WithContext(ctx)

			// 2. Setup Recorder
			w := httptest.NewRecorder()

			// 3. Invoke Middleware
			httpio.ToHTTP(tt.handlerFunc).ServeHTTP(w, req)

			// 4. Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			// If we expect an error response (which is JSON)
			if tt.expectedCode != "" {
				var resp apperr.Error
				err := json.NewDecoder(w.Body).Decode(&resp)
				require.NoError(t, err)

				assert.Equal(t, tt.expectedCode, string(resp.Code))
				assert.Equal(t, reqID, resp.RequestID, "Request ID should match context")
			} else {
				// Assert body for the happy path
				assert.Equal(t, "ok", w.Body.String())
			}
		})
	}
}
