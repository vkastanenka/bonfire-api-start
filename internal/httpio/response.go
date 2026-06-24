package httpio

import (
	"bonfire-api/internal/apperr"
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
)

// --- RESPONSE CONSTANTS ---

// Errors
const (
	ErrHTTPReqFailed = "http request failed"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		// Pre-allocate a reasonable starting size (e.g., 2 KB) to avoid early allocations
		return bytes.NewBuffer(make([]byte, 0, 2048))
	},
}

// --- RESPONSE TYPES ---

// SuccessResponse
type SuccessResponse[T any] struct {
	Message string `json:"message,omitempty"`
	Data    T      `json:"data"`
	Meta    any    `json:"meta,omitempty"`
}

// CursorPagination
type CursorPagination struct {
	NextCursor *string `json:"next_cursor,omitempty"`
	PageSize   int32   `json:"page_size"`
}

// --- RESPONSE ADAPTERS ---

// ToHTTP wraps handlers that return an error to centralize response/logging logic
func ToHTTP(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}

		// Identify/Normalize error
		var appErr *apperr.Error
		if !errors.As(err, &appErr) {
			err = apperr.New(apperr.CodeInternal, apperr.CodeInternal.Title(), apperr.WithErr(err))
			// Extract the newly created concrete *apperr.Error pointer safely
			_ = errors.As(err, &appErr)
		}

		// Enrich and Prepare
		appErr.Enrich(r)
		status, resp := appErr.HTTPResponse()

		// Log
		logError(r, appErr, err, status)

		// Respond
		RespondJSON(w, r, status, resp)
	}
}

// --- RESPONSE FUNCTIONS ---

// RespondJSON
func RespondJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	// MaxPoolBufferCapacity prevents oversized buffers from polluting memory (64 KB)
	const maxPoolBufferCapacity = 64 * 1024

	ctx := r.Context()

	w.Header().Set("Content-Type", "application/json")

	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	// Conditionally return the buffer to the pool to prevent memory bloat
	defer func() {
		if buf.Cap() <= maxPoolBufferCapacity {
			bufferPool.Put(buf)
		}
	}()

	if err := json.NewEncoder(buf).Encode(data); err != nil {
		// Uses ErrorContext to tie logs directly to your request span/ID
		slog.ErrorContext(ctx, "failed to encode json response", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"INTERNAL","message":"An unexpected error occurred."}`))
		return
	}

	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// RespondOK
func RespondOK[T any](w http.ResponseWriter, r *http.Request, data T, message string) {
	RespondJSON(w, r, http.StatusOK, SuccessResponse[T]{
		Message: message,
		Data:    data,
	})
}

// RespondCreated
func RespondCreated[T any](w http.ResponseWriter, r *http.Request, data T, message string) {
	RespondJSON(w, r, http.StatusCreated, SuccessResponse[T]{
		Message: message,
		Data:    data,
	})
}

// RespondCursorList
func RespondCursorList[T any](w http.ResponseWriter, r *http.Request, data T, message string, meta CursorPagination) {
	RespondJSON(w, r, http.StatusOK, SuccessResponse[T]{
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// RespondNoContent
func RespondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// --- RESPONSE HELPERS ---

// logError
func logError(r *http.Request, appErr *apperr.Error, originalErr error, status int) {
	level := slog.LevelInfo
	if appErr.Code == apperr.CodeInternal {
		level = slog.LevelError
	}

	args := []any{
		"path", r.URL.Path,
		"method", r.Method,
		"status", status,
		slog.Group("error_context",
			"code", appErr.Code,
			"detail", appErr.Detail,
			"req_id", appErr.ReqID,
			"trace_id", appErr.TraceID,
			"error", originalErr,
		),
	}

	// Dynamic evaluation of RFC 7807 validation parameter slices
	if len(appErr.InvalidParams) > 0 {
		args = append(args, "invalid_params", appErr.InvalidParams)
	}

	slog.Log(r.Context(), level, ErrHTTPReqFailed, args...)
}
