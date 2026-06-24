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

const (
	ErrHTTPReqFailed = "http request failed"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 2048))
	},
}

// --- RESPONSE TYPES ---

type SuccessResponse[T any] struct {
	Message string `json:"message,omitempty"`
	Data    T      `json:"data"`
	Meta    any    `json:"meta,omitempty"`
}

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

		// 1. Identify/Normalize error (Only use reflection once)
		var appErr *apperr.Error
		if !errors.As(err, &appErr) {
			appErr = &apperr.Error{
				Code:   apperr.CodeInternal,
				Detail: apperr.CodeInternal.Title(),
				Err:    err,
			}
		}

		// 2. Map domain error onto a request-scoped payload (Thread-Safe!)
		status, resp := MapToProblemDetails(r, appErr)

		// 3. Log using the original context values and tracking IDs
		logError(r, appErr, resp, err)

		// 4. Respond
		RespondJSON(w, r, status, resp)
	}
}

// --- RESPONSE FUNCTIONS ---

func RespondJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	const maxPoolBufferCapacity = 64 * 1024
	ctx := r.Context()

	w.Header().Set("Content-Type", "application/json")

	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	defer func() {
		if buf.Cap() <= maxPoolBufferCapacity {
			bufferPool.Put(buf)
		}
	}()

	if err := json.NewEncoder(buf).Encode(data); err != nil {
		slog.ErrorContext(ctx, "failed to encode json response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"INTERNAL","message":"An unexpected error occurred."}`))
		return
	}

	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func RespondOK[T any](w http.ResponseWriter, r *http.Request, data T, message string) {
	RespondJSON(w, r, http.StatusOK, SuccessResponse[T]{
		Message: message,
		Data:    data,
	})
}

func RespondCreated[T any](w http.ResponseWriter, r *http.Request, data T, message string) {
	RespondJSON(w, r, http.StatusCreated, SuccessResponse[T]{
		Message: message,
		Data:    data,
	})
}

func RespondCursorList[T any](w http.ResponseWriter, r *http.Request, data T, message string, meta CursorPagination) {
	RespondJSON(w, r, http.StatusOK, SuccessResponse[T]{
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func RespondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// --- RESPONSE HELPERS ---

func logError(r *http.Request, appErr *apperr.Error, resp ProblemDetails, originalErr error) {
	level := slog.LevelInfo
	if appErr.Code == apperr.CodeInternal {
		level = slog.LevelError
	}

	args := []any{
		"path", r.URL.Path,
		"method", r.Method,
		"status", resp.Status,
		slog.Group("error_context",
			"code", appErr.Code,
			"detail", appErr.Detail,
			"req_id", resp.ReqID,
			"trace_id", resp.TraceID,
			"error", originalErr,
		),
	}

	if len(appErr.InvalidParams) > 0 {
		args = append(args, "invalid_params", appErr.InvalidParams)
	}

	slog.Log(r.Context(), level, ErrHTTPReqFailed, args...)
}
