package httpio

import (
	"bonfire-api/internal/apperr"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
)

const (
	contentTypeJSON    = "application/json"
	contentTypeProblem = "application/problem+json"
)

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 2048))
	},
}

func RespondOK[T any](w http.ResponseWriter, r *http.Request, data T) {
	respondJSON(w, r, http.StatusOK, data)
}

func RespondCreated[T any](w http.ResponseWriter, r *http.Request, data T) {
	respondJSON(w, r, http.StatusCreated, data)
}

func RespondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func respondJSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	writeJSON(w, r, status, contentTypeJSON, data)
}

func ToHTTPErr(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			respondError(w, r, err)
		}
	}
}

func respondError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperr.Error

	if errors.As(err, &appErr) {
	} else if errors.Is(err, context.DeadlineExceeded) {
		appErr = &apperr.Error{
			Code:   apperr.CodeGatewayTimeout,
			Detail: apperr.CodeGatewayTimeout.Detail(),
			Err:    err,
		}
	} else {
		appErr = &apperr.Error{
			Code:   apperr.CodeInternal,
			Detail: apperr.CodeInternal.Detail(),
			Err:    err,
		}
	}

	status, resp := MapToProblemDetails(r, appErr)
	logError(r, appErr, resp, err)

	writeJSON(w, r, status, contentTypeProblem, resp)
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, contentType string, data any) {
	const maxPoolBufferCapacity = 64 * 1024
	ctx := r.Context()

	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	defer func() {
		if buf.Cap() <= maxPoolBufferCapacity {
			bufferPool.Put(buf)
		}
	}()

	if err := json.NewEncoder(buf).Encode(data); err != nil {
		slog.ErrorContext(ctx, "failed to encode json response payload",
			"error", err,
			"http.path", r.URL.Path,
		)

		w.Header().Set("Content-Type", contentTypeProblem)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"type":"https://api.bonfire.com/errors/internal","title":"Internal Server Error","status":500,"detail":"An unexpected error occurred during payload encoding."}`))
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func logError(r *http.Request, appErr *apperr.Error, resp ProblemDetails, originalErr error) {
	level := slog.LevelInfo
	if appErr.Code == apperr.CodeInternal {
		level = slog.LevelError
	}

	args := []any{
		"http.method", r.Method,
		"http.path", r.URL.Path,
		"http.status_code", resp.Status,
		slog.Group("error",
			"code", appErr.Code,
			"detail", appErr.Detail,
			"raw", originalErr.Error(),
		),
	}

	if len(appErr.InvalidParams) > 0 {
		args = append(args, "error.invalid_params", appErr.InvalidParams)
	}

	slog.Log(r.Context(), level, "http request execution failed", args...)
}
