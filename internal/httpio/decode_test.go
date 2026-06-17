package httpio_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
)

// TestPayload is a dummy struct for testing the decoder
type TestPayload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestDecodeJSON(t *testing.T) {
	t.Run("Success - JSON Valid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name":"test","count":5}`)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.NoError(t, err)
		assert.Equal(t, "test", dst.Name)
		assert.Equal(t, 5, dst.Count)
	})

	t.Run("Failure - JSON Missing Content-Type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name":"test"}`)))
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErr.Code, apperr.CodeUnsupportedMediaType)
		assert.Contains(t, appErr.Message, httpio.UnsupportedMediaTypeMsg)
	})

	t.Run("Failure - JSON Unsupported or Malformed Media Types", func(t *testing.T) {
		tests := []struct {
			name        string
			contentType string
		}{
			{
				name:        "Valid format but incorrect type",
				contentType: "application/xml",
				// Triggers: mediaType != "application/json"
			},
			{
				name:        "Completely unrelated type",
				contentType: "text/html; charset=utf-8",
				// Triggers: mediaType != "application/json"
			},
			{
				name:        "Malformed parameter syntax causes parser failure",
				contentType: "text/html; =invalid-syntax",
				// Triggers: mime.ParseMediaType error (err != nil)
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body := bytes.NewReader([]byte(`{"name":"test"}`))
				req := httptest.NewRequest(http.MethodPost, "/", body)
				req.Header.Set("Content-Type", tt.contentType)
				recorder := httptest.NewRecorder()

				var dst TestPayload
				err := httpio.DecodeJSON(recorder, req, &dst)

				require.Error(t, err)
				var appErr *apperr.Error
				require.True(t, errors.As(err, &appErr))
				assert.Equal(t, appErr.Code, apperr.CodeUnsupportedMediaType)
				assert.Contains(t, appErr.Message, httpio.UnsupportedMediaTypeMsg)
			})
		}
	})

	t.Run("Failure - Context Lifecycle Errors", func(t *testing.T) {
		tests := []struct {
			name         string
			setupContext func() (context.Context, context.CancelFunc)
			expectedCode apperr.Code
			expectedMsg  string
		}{
			{
				name: "Client Cancelled Request Mid-Stream",
				setupContext: func() (context.Context, context.CancelFunc) {
					ctx, cancel := context.WithCancel(context.Background())
					cancel() // Intentionally cancel the context immediately
					return ctx, func() {}
				},
				expectedCode: apperr.CodeClientClosedRequest,
				expectedMsg:  httpio.ClientClosedConnectionMsg,
			},
			{
				name: "Request Timeout / Deadline Exceeded",
				setupContext: func() (context.Context, context.CancelFunc) {
					// Initialize a context that expired one hour ago
					pastTime := time.Now().Add(-1 * time.Hour)
					ctx, cancel := context.WithDeadline(context.Background(), pastTime)
					return ctx, cancel
				},
				expectedCode: apperr.CodeRequestTimeout,
				expectedMsg:  httpio.ReqTimeoutMsg,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx, cancel := tt.setupContext()
				defer cancel()

				req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name":"test"}`)))
				req = req.WithContext(ctx) // Inject the dead or expired context
				req.Header.Set("Content-Type", "application/json")

				recorder := httptest.NewRecorder()
				var dst TestPayload

				// Execute
				err := httpio.DecodeJSON(recorder, req, &dst)

				// Assert
				require.Error(t, err)

				var appErr *apperr.Error
				require.True(t, errors.As(err, &appErr))
				assert.Equal(t, tt.expectedCode, appErr.Code)
				assert.Contains(t, appErr.Message, tt.expectedMsg)
			})
		}
	})

	t.Run("Failure - JSON Payload Too Large", func(t *testing.T) {
		// Generate a payload slightly larger than 1MB (1048576 bytes)
		largeString := strings.Repeat("a", 1048577)
		payload := `{"name": "` + largeString + `"}`

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(payload)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErr.Code, apperr.CodePayloadTooLarge)
		assert.Contains(t, appErr.Message, httpio.PayloadTooLargeMsg)
	})

	t.Run("Failure - JSON Empty Body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte{}))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErr.Code, apperr.CodeInvalidInput)
		assert.Contains(t, appErr.Message, httpio.EmptyBodyMsg)
	})

	t.Run("Failure - JSON Malformed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name": "test", bad-json}`)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErr.Code, apperr.CodeInvalidInput)
		assert.Contains(t, appErr.Message, httpio.MalformedJSONMsg)
	})

	t.Run("Failure - JSON Truncated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name": "test"`)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErr.Code, apperr.CodeInvalidInput)
		assert.Contains(t, appErr.Message, httpio.TruncatedJSONMsg)
	})

	t.Run("Failure - JSON Type Mismatch (UnmarshalTypeError)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name": "test", "count": "five"}`)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))

		require.NotNil(t, appErr.Details)

		targetType := reflect.TypeOf(dst.Count).String()
		expectedDetail := fmt.Sprintf(targetType, httpio.FieldTypeExpectationFmt)

		assert.Equal(t, appErr.Details["count"], expectedDetail)
	})

	t.Run("Failure - Unknown Field", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name": "test", "count": 5, "admin": true}`)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErr.Code, apperr.CodeInvalidInput)
		assert.Equal(t, appErr.Message, fmt.Sprintf(httpio.UnknownFieldFmt, "admin"))
	})

	t.Run("Failure - Multiple JSON Values", func(t *testing.T) {
		// Sending two separate JSON objects in one payload
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name": "test"}{"name": "test2"}`)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErr.Code, apperr.CodeInvalidInput)
		assert.Contains(t, appErr.Message, httpio.DecodeErrMsg)
	})

	t.Run("Failure - Developer Error Non-Pointer Destination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name":"test"}`)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload // Intentionally missing the pointer '&' variable assignment
		err := httpio.DecodeJSON(recorder, req, dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		// Ensures your 'default' switch case catches this gracefully without panicking
		assert.Contains(t, appErr.Message, "Malformed or invalid request body")
	})
}
