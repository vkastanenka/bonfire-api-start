package httpio

import (
	"bonfire-api/internal/apperr"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func MapErrorToResponse(err error) (int, ErrorResponse) {
	var appErr *apperr.Error

	statusCode := http.StatusInternalServerError
	resp := ErrorResponse{
		Error:   string(apperr.CodeInternal),
		Message: "An unexpected internal error occurred.",
	}

	if errors.As(err, &appErr) {
		resp.Error = string(appErr.Code)
		resp.Message = appErr.Message
		resp.Details = appErr.Details

		switch appErr.Code {
		case apperr.CodeInvalidInput:
			statusCode = http.StatusBadRequest
		case apperr.CodeNotFound:
			statusCode = http.StatusNotFound
		case apperr.CodePayloadTooLarge:
			statusCode = http.StatusRequestEntityTooLarge
		case apperr.CodeConflict:
			statusCode = http.StatusConflict
		case apperr.CodeUnauthenticated:
			statusCode = http.StatusUnauthorized
		case apperr.CodeMethodNotAllowed:
			statusCode = http.StatusMethodNotAllowed
		case apperr.CodeUnsupportedMediaType:
			statusCode = http.StatusUnsupportedMediaType
		case apperr.CodeInternal:
			statusCode = http.StatusInternalServerError
			resp.Message = "An unexpected internal error occurred."
			resp.Details = nil
		}
	}

	return statusCode, resp
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func ToHTTP(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			status, resp := MapErrorToResponse(err)
			ctx := r.Context()

			reqID := middleware.GetReqID(r.Context())
			if reqID == "" {
				reqID = "unknown"
			}

			var appErr *apperr.Error
			if errors.As(err, &appErr) {
				if appErr.Code == apperr.CodeInternal {
					slog.ErrorContext(ctx, "internal server error encountered",
						"code", appErr.Code,
						"message", appErr.Message,
						"details", appErr.Details,
						"error", appErr.Err,
					)
				} else {
					slog.InfoContext(ctx, "client request validation failed",
						"code", appErr.Code,
						"message", appErr.Message,
						"details", appErr.Details,
					)
				}
			} else {
				slog.ErrorContext(ctx, "unhandled application error", "error", err)
			}

			RespondJSON(w, status, resp)
		}
	}
}
