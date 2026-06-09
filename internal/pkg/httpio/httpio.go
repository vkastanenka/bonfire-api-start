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

// DecodeJSON reads the request body and parses it into the target destination.
func DecodeJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	// 1. Enforce strict media type validation safely using non-deprecated parser
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return apperr.NewInvalidInput("Missing Content-Type header; must be application/json.")
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil || mediaType != "application/json" {
		return apperr.NewInvalidInput("Content-Type header must be application/json.")
	}

	// 2. Bound your reader to prevent memory DOS attacks
	limitedBody := http.MaxBytesReader(w, r.Body, 1048576) // 1MB

	dec := json.NewDecoder(limitedBody)
	dec.DisallowUnknownFields()

	if err := dec.Decode(data); err != nil {
		// Catch context timeouts/cancellations first
		if r.Context().Err() != nil {
			return apperr.NewInvalidInput("Client closed connection mid-request.", apperr.WithErr(r.Context().Err()))
		}

		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		// Catch standard payload ceiling breaks explicitly
		case errors.As(err, &maxBytesErr):
			return apperr.NewPayloadTooLarge("Request body exceeds 1MB limit.", apperr.WithErr(err))

		case errors.Is(err, io.EOF):
			return apperr.NewInvalidInput("Request body cannot be empty.", apperr.WithErr(err))

		// If it's a true JSON layout mistake
		case errors.As(err, &syntaxErr):
			return apperr.NewInvalidInput("Malformed request body JSON syntax.", apperr.WithErr(err))

		// Handle unexpected stream death as invalid data construction, NOT size warnings
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

		// Handle unknown field errors reliably using targeted sub-string checking
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return apperr.NewInvalidInput(fmt.Sprintf("Unknown field %s present in request body.", fieldName), apperr.WithErr(err))

		default:
			return apperr.NewInvalidInput("Malformed or invalid request body JSON payload.", apperr.WithErr(err))
		}
	}

	// Ensure there are no hanging extra expressions in the stream
	if dec.More() {
		return apperr.NewInvalidInput("Request body must contain only a single JSON value.")
	}

	return nil
}

// RespondJSON marshals data into memory first to guarantee successful encoding
// before sending a success status header to the client.
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

// MapErrorToResponse inspects the error and writes the appropriate JSON payload.
func MapErrorToResponse(err error) (int, ErrorResponse) {
	var appErr *apperr.AppError

	// Default values
	statusCode := http.StatusInternalServerError
	resp := ErrorResponse{
		Error:   string(apperr.TypeInternal),
		Message: "An unexpected internal error occurred.",
	}

	// Unpack if it's our structured domain error
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
		case apperr.TypeInternal:
			statusCode = http.StatusInternalServerError
			resp.Message = "An unexpected internal error occurred."
			resp.Details = nil
		}
	}

	return statusCode, resp
}

// HandlerFunc is an HTTP handler that returns a clean domain error.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// ToHTTP wraps our idiomatic clean handlers into standard Go router formats
func ToHTTP(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Ensure body is clean on function exit, maximizing keep-alive recycling
		defer func() {
			_, _ = io.CopyN(io.Discard, r.Body, 4096)
			r.Body.Close()
		}()

		if err := h(w, r); err != nil {
			status, resp := MapErrorToResponse(err)

			reqID := middleware.GetReqID(r.Context())
			if reqID == "" {
				reqID = "unknown"
			}

			var appErr *apperr.AppError
			if errors.As(err, &appErr) {
				log.Printf("[ERROR] ReqID: %s | Type: %s | Msg: %s | InternalErr: %v",
					reqID, appErr.Type, appErr.Message, appErr.Err)
			} else {
				log.Printf("[CRITICAL] ReqID: %s | Unhandled Error: %v", reqID, err)
			}

			RespondJSON(w, status, resp)
		}
	}
}
