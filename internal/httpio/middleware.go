package httpio

import (
	"bonfire-api/internal/apperr"
	"errors"
	"log/slog"
	"net/http"
)

// ToHTTP wraps handlers that return an error to centralize response/logging logic
func ToHTTP(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}

		// Identify error type
		var appErr *apperr.Error
		if !errors.As(err, &appErr) {
			// If it's a generic error, wrap it as an internal one first
			err = apperr.New(apperr.CodeInternal, apperr.CodeInternal.Message(), apperr.WithErr(err))
			errors.As(err, &appErr)
		}

		// Add metadata
		appErr.Enrich(r)

		// Extract Response
		status, resp := appErr.HTTPResponse()

		// Set log level
		logLevel := slog.LevelInfo
		if appErr.IsCode(apperr.CodeInternal) {
			logLevel = slog.LevelError
		}

		slog.Log(r.Context(), logLevel, HTTPReqFailedMsg,
			"path", r.URL.Path,
			"code", appErr.Code,
			"status", status,
			"request_id", appErr.RequestID,
			"trace_id", appErr.TraceID,
			"error", err,
		)

		// Respond
		RespondJSON(w, status, resp)
	}
}
