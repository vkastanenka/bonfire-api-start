package httpio

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/validator"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

	"github.com/go-playground/form"
)

// --- REQUEST CONSTANTS ---

// Errors
const (
	// Header Errors
	ErrUnsupportedMediaType = "Missing or invalid Content-Type header; must be application/json."

	// Client/Stream Errors
	ErrClientClosedConnection = "Client closed connection mid-request."
	ErrPayloadTooLarge        = "Request body exceeds 1MB limit."
	ErrEmptyBody              = "Request body cannot be empty."

	// JSON Structural Errors
	ErrMalformedJSON       = "Malformed request body JSON syntax."
	ErrTruncatedJSON       = "Truncated or malformed JSON structure received."
	ErrInvalidPayload      = "Malformed or invalid request body JSON payload."
	ErrSingleValueRequired = "Request body must contain only a single JSON value."

	// Field Validation & Typing Errors
	ErrInvalidFieldType     = "Invalid data type provided for request body field(s)."
	ErrFieldTypeExpectation = "Must be of type %s"
	ErrUnknownField         = "Unknown field '%s' present in request body."

	// Internal Errors
	ErrDecode     = "An unexpected parsing error occurred."
	ErrReqTimeout = "Request timed out."
)

// --- REQUEST TYPES ---

type ClientMeta struct {
	IP        netip.Addr
	UserAgent string
}

type Sanitizable interface {
	Sanitize()
}

// --- REQUEST METADATA FUNCTIONS ---

// GetClientIP
func GetClientIP(r *http.Request, trustProxy bool) netip.Addr {
	var rawIP string

	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if idx := strings.Index(xff, ","); idx != -1 {
				rawIP = strings.TrimSpace(xff[:idx])
			} else {
				rawIP = strings.TrimSpace(xff)
			}
		}

		if rawIP == "" {
			if xri := r.Header.Get("X-Real-IP"); xri != "" {
				rawIP = strings.TrimSpace(xri)
			}
		}
	}

	if rawIP == "" {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			rawIP = r.RemoteAddr
		} else {
			rawIP = ip
		}
	}

	addr, err := netip.ParseAddr(rawIP)
	if err != nil {
		return netip.IPv4Unspecified()
	}

	return addr
}

// GetClientMeta
func GetClientMeta(r *http.Request, trustProxy bool) ClientMeta {
	return ClientMeta{
		IP:        GetClientIP(r, trustProxy),
		UserAgent: r.UserAgent(),
	}
}

// --- REQUEST QUERY FUNCTIONS ---

// GetQueryInt
func GetQueryInt(r *http.Request, key string, defaultValue int) int {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// GetQueryString
func GetQueryString(r *http.Request, key string, defaultValue string) string {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	return strings.TrimSpace(value)
}

// --- REQUEST BINDING FUNCTIONS ---

// BindJSON
func BindJSON[T any](w http.ResponseWriter, r *http.Request, validator *validator.Validator) (T, error) {
	var req T
	if err := DecodeJSON(w, r, &req); err != nil {
		return req, err
	}

	if s, ok := any(&req).(Sanitizable); ok {
		s.Sanitize()
	}

	if err := validator.ValidateStruct(&req); err != nil {
		return req, err
	}

	return req, nil
}

// DecodeJSON
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Define max JSON size
	const maxBodyBytes = 1 * 1024 * 1024

	// Check for "Content-Type" header
	ct := strings.TrimSpace(r.Header.Get("Content-Type"))
	if ct == "" {
		return apperr.New(apperr.CodeUnsupportedMediaType, ErrUnsupportedMediaType)
	}

	// Check for "application/json" header prefix, otherwise attempt parse
	if !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		mediaType, _, err := mime.ParseMediaType(ct)
		if err != nil || mediaType != "application/json" {
			return apperr.New(apperr.CodeUnsupportedMediaType, ErrUnsupportedMediaType, apperr.WithErr(err))
		}
	}

	// Init 1MB limit
	limitedBody := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer limitedBody.Close()

	// Init decoder - limit body and prevent unknown fields
	dec := json.NewDecoder(limitedBody)
	dec.DisallowUnknownFields()

	// Decode JSON into struct
	if err := dec.Decode(dst); err != nil {
		// Check if context is closed
		if r.Context().Err() != nil {
			switch {
			case errors.Is(r.Context().Err(), context.Canceled):
				return apperr.New(apperr.CodeClientClosedRequest, ErrClientClosedConnection, apperr.WithErr(r.Context().Err()))
			case errors.Is(r.Context().Err(), context.DeadlineExceeded):
				return apperr.New(apperr.CodeRequestTimeout, ErrReqTimeout, apperr.WithErr(r.Context().Err()))
			default:
				return apperr.New(apperr.CodeClientClosedRequest, ErrClientClosedConnection, apperr.WithErr(r.Context().Err()))
			}
		}

		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		// Check if payload is too large
		case errors.As(err, &maxBytesErr):
			return apperr.New(apperr.CodePayloadTooLarge, ErrPayloadTooLarge, apperr.WithErr(err))

		// Check if JSON body is empty
		case errors.Is(err, io.EOF):
			return apperr.New(apperr.CodeInvalidInput, ErrEmptyBody, apperr.WithErr(err))

		// Check if JSON is malformed
		case errors.As(err, &syntaxErr):
			return apperr.New(apperr.CodeInvalidInput, ErrMalformedJSON, apperr.WithErr(err))

		// Check if JSON is truncated
		case errors.Is(err, io.ErrUnexpectedEOF):
			return apperr.New(apperr.CodeInvalidInput, ErrTruncatedJSON, apperr.WithErr(err))

		// Check if JSON field types are valid
		case errors.As(err, &unmarshalTypeErr):
			fieldName := unmarshalTypeErr.Field
			if fieldName == "" { // Client sends a raw string
				fieldName = "field"
			}

			return apperr.New(
				apperr.CodeInvalidInput,
				ErrInvalidFieldType,
				apperr.WithErr(err),
				apperr.WithInvalidParam(fieldName, fmt.Sprintf(ErrFieldTypeExpectation, unmarshalTypeErr.Type)),
			)

		// Check if JSON has unknown fields
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			fieldName = strings.Trim(fieldName, `"`)
			return apperr.New(apperr.CodeInvalidInput, fmt.Sprintf(ErrUnknownField, fieldName), apperr.WithErr(err))

		// Handle other errors
		default:
			return apperr.New(apperr.CodeInternal, ErrDecode, apperr.WithErr(err))
		}
	}

	// Check for multiple JSON objects
	if dec.More() {
		return apperr.New(apperr.CodeInvalidInput, ErrSingleValueRequired)
	}

	return nil
}

// Initialize a single, thread-safe form decoder for the package
var formDecoder = form.NewDecoder()

// --- REQUEST BINDING FUNCTIONS ---

// BindQuery parses query strings into a struct, sanitizes, and validates it.
// Note: Unlike BindJSON, this does NOT need http.ResponseWriter because query parameters
// are already parsed into memory by the server and don't stream raw request bytes.
func BindQuery[T any](r *http.Request, validator *validator.Validator) (T, error) {
	var req T
	if err := DecodeQuery(r, &req); err != nil {
		return req, err
	}

	// Automagic sanitization hook if the DTO implements Sanitizable
	if s, ok := any(&req).(Sanitizable); ok {
		s.Sanitize()
	}

	// Validate the final populated struct
	if err := validator.ValidateStruct(&req); err != nil {
		return req, err
	}

	return req, nil
}

// DecodeQuery extracts URL query parameters and unmarshals them into the target destination.
func DecodeQuery(r *http.Request, dst any) error {
	// Ensure query parameters are fully parsed by the runtime
	if err := r.ParseForm(); err != nil {
		return apperr.New(apperr.CodeInvalidInput, "Failed to parse query parameters.", apperr.WithErr(err))
	}

	// Map the r.URL.Query() map[string][]string directly to the struct fields
	if err := formDecoder.Decode(dst, r.URL.Query()); err != nil {
		var decodeErrors form.DecodeErrors
		if errors.As(err, &decodeErrors) {
			// Extract the structural validation breakdown matching your UnmarshalTypeError pattern
			for field, fieldErr := range decodeErrors {
				return apperr.New(
					apperr.CodeInvalidInput,
					"Invalid data type provided for query parameter(s).",
					apperr.WithErr(fieldErr),
					apperr.WithInvalidParam(field, fmt.Sprintf("Must be a valid type: %v", fieldErr)),
				)
			}
		}
		return apperr.New(apperr.CodeInvalidInput, "Malformed query parameters.", apperr.WithErr(err))
	}

	return nil
}
