package httpio

import (
	"bonfire-api/internal/apperr"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func ToHTTP(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			ctx := r.Context()
			reqID := middleware.GetReqID(ctx)

			var appErr *apperr.Error
			if errors.As(err, &appErr) {
				appErr.TraceID = reqID
				appErr.RequestID = reqID
				appErr.Timestamp = time.Now().UTC().Format(time.RFC3339)
			}

			// Map to status and response DTO
			status, resp := mapErrorToResponse(err)

			// Determine logging level
			logLevel := slog.LevelInfo
			if apperr.ErrorCode(err) == apperr.CodeInternal {
				logLevel = slog.LevelError
			}

			// Log the error
			slog.Log(ctx, logLevel, "http request failed",
				"path", r.URL.Path,
				"code", apperr.ErrorCode(err),
				"status", status,
				"error", err,
			)

			// Send JSON response
			RespondJSON(w, status, resp)
		}
	}
}

// mapErrorToResponse translates domain errors into HTTP status and JSON responses
func mapErrorToResponse(err error) (int, apperr.ErrorResponse) {
	var appErr *apperr.Error

	if !errors.As(err, &appErr) || appErr.IsCode(apperr.CodeInternal) {
		return http.StatusInternalServerError, apperr.ErrorResponse{
			Code:    string(apperr.CodeInternal),
			Message: "An unexpected internal error occurred.",
		}
	}

	return appErr.Code.HTTPStatus(), appErr.ToResponse()
}
