package httpio

import (
	"bonfire-api/internal/apperr"
	"errors"
	"net/http"
)

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
			errors.As(err, &appErr)
		}

		// Enrich and Prepare
		appErr.Enrich(r)
		status, resp := appErr.HTTPResponse()

		// Log
		logError(r, appErr, err, status)

		// Respond
		RespondJSON(w, status, resp)
	}
}

// logError logs app errors
func logError(r *http.Request, appErr *apperr.Error, originalErr error, status int) {
	// level := slog.LevelInfo
	// if appErr.IsCode(apperr.CodeInternal) {
	// 	level = slog.LevelError
	// }

	// args := []any{
	// 	"path", r.URL.Path,
	// 	"method", r.Method,
	// 	"status", status,
	// 	slog.Group("error_context",
	// 		"code", appErr.Code,
	// 		"request_id", appErr.RequestID,
	// 		"trace_id", appErr.TraceID,
	// 		"error", originalErr,
	// 	),
	// }

	// if len(appErr.Details) > 0 {
	// 	args = append(args, "details", appErr.Details)
	// }
	// if len(appErr.ValidationErrors) > 0 {
	// 	args = append(args, "validation_errors", appErr.ValidationErrors)
	// }

	// slog.Log(r.Context(), level, HTTPReqFailedMsg, args...)
}
