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

const (
	errHTTPReqFailed = "http request failed"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 2048))
	},
}

type SuccessResponse[T any] struct {
	Message string `json:"message,omitempty"`
	Data    T      `json:"data"`
	Meta    any    `json:"meta,omitempty"`
}

type CursorPagination struct {
	NextCursor *string `json:"next_cursor,omitempty"`
	PageSize   int32   `json:"page_size"`
}

func ToHTTP(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			RespondError(w, r, err)
		}
	}
}

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

func RespondError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		appErr = &apperr.Error{
			Code:   apperr.CodeInternal,
			Detail: apperr.CodeInternal.Title(),
			Err:    err,
		}
	}

	// // Crucial RFC 7807 requirement:
	// w.Header().Set("Content-Type", "application/problem+json")
	// w.WriteHeader(status)

	status, resp := MapToProblemDetails(r, appErr)
	logError(r, appErr, resp, err)
	RespondJSON(w, r, status, resp)
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

func logError(r *http.Request, appErr *apperr.Error, resp ProblemDetails, originalErr error) {
	level := slog.LevelInfo
	if appErr.Code == apperr.CodeInternal {
		level = slog.LevelError
	}

	args := []any{
		"method", r.Method,
		"path", r.URL.Path,
		"status", resp.Status,
		slog.Group("error_context",
			"code", appErr.Code,
			"detail", appErr.Detail,
			"error", originalErr,
		),
	}

	if len(appErr.InvalidParams) > 0 {
		args = append(args, "invalid_params", appErr.InvalidParams)
	}

	slog.Log(r.Context(), level, errHTTPReqFailed, args...)
}
