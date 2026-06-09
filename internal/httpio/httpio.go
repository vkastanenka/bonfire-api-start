package httpio

import (
	"bonfire-api/internal/apperr"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return apperr.NewInvalidInput("Missing Content-Type header; must be application/json.")
	}

	ctLower := strings.ToLower(ct)
	if !strings.HasPrefix(ctLower, "application/json") {
		mediaType, _, err := mime.ParseMediaType(ct)
		if err != nil || mediaType != "application/json" {
			return apperr.NewInvalidInput("Content-Type header must be application/json.")
		}
	}

	// 1MB standard buffer ceiling
	limitedBody := http.MaxBytesReader(w, r.Body, 1048576)
	defer limitedBody.Close()

	dec := json.NewDecoder(limitedBody)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if r.Context().Err() != nil {
			return apperr.NewInvalidInput("Client closed connection mid-request.", apperr.WithErr(r.Context().Err()))
		}

		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &maxBytesErr):
			return apperr.NewPayloadTooLarge("Request body exceeds 1MB limit.", apperr.WithErr(err))

		case errors.Is(err, io.EOF):
			return apperr.NewInvalidInput("Request body cannot be empty.", apperr.WithErr(err))

		case errors.As(err, &syntaxErr):
			return apperr.NewInvalidInput("Malformed request body JSON syntax.", apperr.WithErr(err))

		case errors.Is(err, io.ErrUnexpectedEOF):
			return apperr.NewInvalidInput("Truncated or malformed JSON structure received.", apperr.WithErr(err))

		case errors.As(err, &unmarshalTypeErr):
			fieldName := unmarshalTypeErr.Field
			if fieldName == "" {
				fieldName = "field"
			}
			details := map[string]string{
				fieldName: fmt.Sprintf("Must be of type %s", unmarshalTypeErr.Type),
			}
			return apperr.NewInvalidInput("Invalid data type provided for request body field(s).", apperr.WithDetails(details), apperr.WithErr(err))

		// Safe string checking fallback for unknown JSON fields
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return apperr.NewInvalidInput(fmt.Sprintf("Unknown field %s present in request body.", fieldName), apperr.WithErr(err))

		default:
			return apperr.NewInvalidInput("Malformed or invalid request body JSON payload.", apperr.WithErr(err))
		}
	}

	if dec.More() {
		return apperr.NewInvalidInput("Request body must contain only a single JSON value.")
	}

	return nil
}

func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"INTERNAL","message":"An unexpected internal error occurred."}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

func MapErrorToResponse(err error) (int, ErrorResponse) {
	var appErr *apperr.Error

	statusCode := http.StatusInternalServerError
	resp := ErrorResponse{
		Error:   string(apperr.TypeInternal),
		Message: "An unexpected internal error occurred.",
	}

	if errors.As(err, &appErr) {
		resp.Error = string(appErr.Type)
		resp.Message = appErr.Message
		resp.Details = appErr.Details

		switch appErr.Type {
		case apperr.TypeInvalidInput:
			statusCode = http.StatusBadRequest
		case apperr.TypeNotFound:
			statusCode = http.StatusNotFound
		case apperr.TypePayloadTooLarge:
			statusCode = http.StatusRequestEntityTooLarge
		case apperr.TypeConflict:
			statusCode = http.StatusConflict
		case apperr.TypeUnauthenticated:
			statusCode = http.StatusUnauthorized
		case apperr.TypeMethodNotAllowed:
			statusCode = http.StatusMethodNotAllowed
		case apperr.TypeInternal:
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
		// Clean up is handled gracefully via standard library context execution
		// inside standard HTTP multiplexers; manual deep draining omitted
		// here to avoid swallowing MaxBytes errors prematurely.
		if err := h(w, r); err != nil {
			status, resp := MapErrorToResponse(err)

			reqID := middleware.GetReqID(r.Context())
			if reqID == "" {
				reqID = "unknown"
			}

			var appErr *apperr.Error
			if errors.As(err, &appErr) {
				// Don't log full internal details to standard error output if it's just a user syntax error
				if appErr.Type == apperr.TypeInternal {
					log.Printf("[CRITICAL] ReqID: %s | Msg: %s | Details: %v | InternalErr: %v",
						reqID, appErr.Message, appErr.Details, appErr.Err)
				} else {
					log.Printf("[INFO] ReqID: %s | UserError: %s | Msg: %s", reqID, appErr.Type, appErr.Message)
				}
			} else {
				log.Printf("[CRITICAL] ReqID: %s | Unhandled Error: %v", reqID, err)
			}

			RespondJSON(w, status, resp)
		}
	}
}
