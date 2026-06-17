package httpio_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	t.Run("Success - Valid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name":"test","count":5}`)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.NoError(t, err)
		assert.Equal(t, "test", dst.Name)
		assert.Equal(t, 5, dst.Count)
	})

	t.Run("Failure - Missing Content-Type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name":"test"}`)))
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.CodeUnsupportedMediaType, appErr.Code)
		assert.Contains(t, appErr.Message, "Missing Content-Type header")
	})

	t.Run("Failure - Empty Body (EOF)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte{}))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Contains(t, appErr.Message, "Request body cannot be empty")
	})

	t.Run("Failure - Malformed JSON Syntax", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name": "test", bad-json}`)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Contains(t, appErr.Message, "Malformed request body JSON syntax")
	})

	t.Run("Failure - Type Mismatch (UnmarshalTypeError)", func(t *testing.T) {
		// Passing a string to "count" which expects an int
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name": "test", "count": "five"}`)))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.CodeInvalidInput, appErr.Code)
		assert.Contains(t, appErr.Message, "Invalid data type")

		// Verify your custom details map captured the field name
		require.NotNil(t, appErr.Details)
		assert.Contains(t, appErr.Details["count"], "Must be of type int")
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
		assert.Contains(t, appErr.Message, "Unknown field 'admin'")
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
		assert.Contains(t, appErr.Message, "contain only a single JSON value")
	})

	t.Run("Failure - Payload Too Large", func(t *testing.T) {
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
		assert.Equal(t, apperr.CodePayloadTooLarge, appErr.Code)
		assert.Contains(t, appErr.Message, "Request body exceeds 1MB limit")
	})

	t.Run("Failure - Client Cancelled Request Mid-Stream", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Intentionally cancel the context immediately

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name":"test"}`)))
		req = req.WithContext(ctx) // Inject the dead context
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		var dst TestPayload
		err := httpio.DecodeJSON(recorder, req, &dst)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.CodeInvalidInput, appErr.Code)
		assert.Contains(t, appErr.Message, "Client closed connection mid-request")
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
