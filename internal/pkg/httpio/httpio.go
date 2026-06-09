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
	ct := r.Header.Get("Content-Type")
	if ct != "" {
		mediaType, _, err := mime.ParseMediaType(ct)
		if err != nil || mediaType != "application/json" {
			return apperr.NewInvalidInput("Content-Type header must be application/json.")
		}
	} else {
		// Strictly enforce content-type if your API expects only JSON
		return apperr.NewInvalidInput("Missing Content-Type header; must be application/json.")
	}

	limitedBody := http.MaxBytesReader(w, r.Body, 1048576) // 1MB max body limit

	dec := json.NewDecoder(limitedBody)
	dec.DisallowUnknownFields()

	if err := dec.Decode(data); err != nil {
		if errors.Is(err, r.Context().Err()) {
			return apperr.NewInvalidInput("Client closed connection mid-request.", apperr.WithErr(err))
		}

		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &maxBytesErr):
			return apperr.NewPayloadTooLarge("Request body exceeds 1MB limit.", apperr.WithErr(err))

		case errors.Is(err, io.ErrUnexpectedEOF) && strings.Contains(err.Error(), "request body too large"):
			return apperr.NewPayloadTooLarge("Request body exceeds 1MB limit.", apperr.WithErr(err))

		case errors.Is(err, io.EOF):
			return apperr.NewInvalidInput("Request body cannot be empty.", apperr.WithErr(err))

		case errors.As(err, &syntaxErr):
			return apperr.NewInvalidInput("Malformed request body JSON syntax.", apperr.WithErr(err))

		case errors.As(err, &unmarshalTypeErr):
			msg := "Invalid data type provided for request body field(s)."
			fieldName := "field"
			if unmarshalTypeErr.Field != "" {
				fieldName = unmarshalTypeErr.Field
			}
			details := map[string]string{
				fieldName: fmt.Sprintf("Must be of type %s", unmarshalTypeErr.Type),
			}
			return apperr.NewInvalidInput(msg, apperr.WithDetails(details), apperr.WithErr(err))

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
			resp.Message = "An unexpected internal error occurred." // Mask sensitive internals
			resp.Details = nil                                      // Prevent leaking backend state
		}
	}

	return statusCode, resp
}

// HandlerFunc is an HTTP handler that returns a clean domain error.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func ToHTTP(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			// 1. Drain the body up to a limit to allow connection reuse on error
			_, _ = io.CopyN(io.Discard, r.Body, 4096)
			r.Body.Close()

			// 2. Map structural data
			status, resp := MapErrorToResponse(err)

			// 3. Centralized log placement inside the transport adapter
			reqID := middleware.GetReqID(r.Context())
			if reqID == "" {
				reqID = "unknown"
			}

			var appErr *apperr.AppError
			if errors.As(err, &appErr) {
				// Internal details printed to server logs safely, away from client eyes
				log.Printf("[ERROR] ReqID: %s | Type: %s | Msg: %s | InternalErr: %v",
					reqID, appErr.Type, appErr.Message, appErr.Err)
			} else {
				log.Printf("[CRITICAL] ReqID: %s | Unhandled Error: %v", reqID, err)
			}

			// 4. Render response
			RespondJSON(w, status, resp)
		}
	}
}
