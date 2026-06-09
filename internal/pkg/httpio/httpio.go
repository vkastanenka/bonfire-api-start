package httpio

import (
	"bonfire-api/internal/apperr"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// DecodeJSON reads the request body and parses it into the target destination.
func DecodeJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1048576) // 1MB
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(data); err != nil {
		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &maxBytesErr):
			return apperr.NewPayloadTooLarge("Request body exceeds 1MB limit.", err)

		case errors.Is(err, io.EOF):
			return apperr.NewInvalidInput("Request body cannot be empty.", err)

		case errors.As(err, &syntaxErr):
			return apperr.NewInvalidInput("Malformed request body JSON syntax.", err)

		case errors.As(err, &unmarshalTypeErr):
			return apperr.NewInvalidInput("Invalid data type provided for request body field(s).", err)

		case strings.HasPrefix(err.Error(), "json: unknown field"):
			return apperr.NewInvalidInput("Unknown field present in request body.", err)

		default:
			return apperr.NewInvalidInput("Malformed or invalid request body JSON payload.", err)
		}
	}

	if dec.More() {
		return apperr.NewInvalidInput("Request body must contain only a single JSON value.", nil)
	}

	return nil
}

// RespondJSON marshals data into memory first to guarantee successful encoding
// before sending a success status header to the client.
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		// Log this internal error safely out-of-band here
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"Internal server error processing response"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// MapErrorToResponse inspects the error and writes the appropriate JSON payload.
func MapErrorToResponse(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	var appErr *apperr.AppError

	// Default to internal server error if it's an unclassified error
	statusCode := http.StatusInternalServerError
	resp := ErrorResponse{
		Error: string(apperr.TypeInternal),
	}

	// Unpack if it's a known domain error
	if errors.As(err, &appErr) {
		resp.Error = string(appErr.Type)
		resp.Message = appErr.Message

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
		}
	} else {
		// This handles third-party or uncaught panicked errors safely
		resp.Message = "An unexpected error occurred."
	}

	// Log the full underlying error out-of-band for telemetry (Zap, Logrus, etc.)
	// logger.Error("HTTP Request Failed", "status", statusCode, "err", err)

	RespondJSON(w, statusCode, resp)
}

// HandlerFunc is an HTTP handler that returns a clean domain error.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// ToHTTP converts our custom domain-aware handler into a standard http.HandlerFunc.
func ToHTTP(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			MapErrorToResponse(w, err)
		}
	}
}
