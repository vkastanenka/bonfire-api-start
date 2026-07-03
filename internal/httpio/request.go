package httpio

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/token"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-playground/form"
	goValidator "github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// --- PACKAGE ENGINE CORE ---

var (
	formDecoder = form.NewDecoder()
	validator   = goValidator.New()
	rgxUsername = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_.]?[a-zA-Z0-9])+$`)
)

// init configures our shared, thread-safe engines at application startup
func init() {
	// 1. Setup Tag Resolution to look for JSON names instead of Go Struct Field Names
	validator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		// Check JSON first (most common for request bodies)
		if tag := fld.Tag.Get("json"); tag != "" && tag != "-" {
			if idx := strings.IndexByte(tag, ','); idx != -1 {
				return tag[:idx]
			}
			return tag
		}

		// Fallback to form tags (for BindQuery)
		if tag := fld.Tag.Get("form"); tag != "" && tag != "-" {
			return tag
		}

		// Fallback to path tags (for BindPath)
		if tag := fld.Tag.Get("path"); tag != "" && tag != "-" {
			return tag
		}

		return ""
	})

	// 2. Register Custom Logic Rules
	validator.RegisterValidation("valid_username", func(fl goValidator.FieldLevel) bool {
		return rgxUsername.MatchString(fl.Field().String())
	})

	// 3. Register Domain Aliases
	validator.RegisterAlias("identity_id", "required,uuid,len=36")
	validator.RegisterAlias("identity_email", "required,email,max=255")
	validator.RegisterAlias("identity_username", "required,min=4,max=32,valid_username")
	validator.RegisterAlias("identity_password", "required,min=12,max=128")
	validator.RegisterAlias("security_password", "required,min=12,max=128")
	validator.RegisterAlias("profile_display_name", "omitempty,min=3,max=32")
}

// --- PRIVATIZED VALIDATION ERROR TEXTS ---

const (
	errValidationFailed       = "Validation failed for the request payload."
	errRequired               = "This field is required."
	errWhitespace             = "This field cannot consist entirely of whitespace."
	errEmail                  = "Invalid email format."
	errAlphanum               = "Must contain only letters and numbers."
	errUsername               = "Must contain only letters, numbers, underscores, or periods."
	errMinString              = "Must be at least %s characters long."
	errMinNumeric             = "Must be %s or greater."
	errMinCollection          = "Must contain at least %s items."
	errMaxString              = "Cannot be longer than %s characters."
	errMaxNumeric             = "Must be %s or less."
	errMaxCollection          = "Cannot contain more than %s items."
	errInvalidConstraintValue = "Invalid value for constraint: %s"
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

	// Auth Ctx
	errMissingAuthCtx = "Missing authentication context."
)

// --- REQUEST TYPES ---

type ClientMeta struct {
	IP        netip.Addr
	UserAgent string
}

type Sanitizable interface {
	Sanitize()
}

type ContextKey string

// --- REQUEST CONSTANTS ---

const ClaimsKey ContextKey = "user_claims"

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

// --- REQUEST AUTH FUNCTIONS ---

// GetCtxClaims extracts token claims from the request context.
func GetCtxClaims(ctx context.Context) (*token.Claims, error) {
	claims, ok := ctx.Value(ClaimsKey).(*token.Claims)
	if !ok {
		return nil, errors.New(errMissingAuthCtx)
	}
	return claims, nil
}

// GetCtxUserID
func GetCtxUserID(ctx context.Context) (uuid.UUID, error) {
	claims, err := GetCtxClaims(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
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
func BindJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var req T
	if err := DecodeJSON(w, r, &req); err != nil {
		return req, err
	}

	if s, ok := any(&req).(Sanitizable); ok {
		s.Sanitize()
	}

	if err := validate(&req); err != nil {
		return req, err
	}

	return req, nil
}

// validate validators a struct and maps failures to custom application errors.
func validate(s interface{}) error {
	// Validate struct with core engine and exit early when valid
	err := validator.Struct(s)
	if err == nil {
		return nil
	}

	// Handle invalid validator arg
	var invalidValidationError *goValidator.InvalidValidationError
	if errors.As(err, &invalidValidationError) {
		return apperr.NewInternal(err, "")
	}

	// Handle validation failures
	var validationErrors goValidator.ValidationErrors
	if errors.As(err, &validationErrors) {
		invalidParams := make([]apperr.InvalidParam, 0, len(validationErrors))

		// Loop through each validation failure
		for _, fieldErr := range validationErrors {
			ns := fieldErr.StructNamespace()
			var jsonPath string

			// Extract a clean field path by removing the root struct name (e.g., "User.Age" -> "Age")
			if idx := strings.Index(ns, "."); idx != -1 {
				jsonPath = ns[idx+1:]
			} else {
				jsonPath = fieldErr.Field()
			}

			// Append field and its error message to the list
			invalidParams = append(invalidParams, apperr.InvalidParam{
				Name:   jsonPath,
				Reason: msgForFieldError(fieldErr),
			})
		}

		// Return error with all validation errors
		return apperr.NewInvalidInput(
			err,
			errValidationFailed,
			apperr.Params(invalidParams),
		)
	}

	// Unexpected error fallback
	return apperr.NewInternal(err, "")
}

// msgForFieldError returns an error message for a failed validation tag.
func msgForFieldError(err goValidator.FieldError) string {
	// Custom handling for empty/whitespace string edge-cases caught by 'required'
	if err.Tag() == "required" {
		val := err.Value()

		// Guard against raw nil interface values before checking reflect.TypeOf(val)
		if val != nil && reflect.TypeOf(val).Kind() == reflect.Ptr {
			sv := reflect.ValueOf(val)
			if !sv.IsNil() {
				val = sv.Elem().Interface()
			}
		}

		// Check if the value is a string consisting only of whitespace characters
		if valStr, ok := val.(string); ok {
			if len(valStr) > 0 && strings.TrimSpace(valStr) == "" {
				return errWhitespace
			}
		}
		return errRequired
	}

	// Map validation tags to their error messages
	switch err.Tag() {
	case "email":
		return errEmail
	case "alphanum":
		return errAlphanum
	case "valid_username":
		return errUsername
	case "min":
		return formatRangeMessage(err, errMinString, errMinNumeric, errMinCollection)
	case "max":
		return formatRangeMessage(err, errMaxString, errMaxNumeric, errMaxCollection)
	default:
		return fmt.Sprintf(errInvalidConstraintValue, err.Tag())
	}
}

// formatRangeMessage formats range errors based on the field's data type.
func formatRangeMessage(err goValidator.FieldError, stringTmpl, numericTmpl, collectionTmpl string) string {
	switch err.Kind() {
	case reflect.String:
		return fmt.Sprintf(stringTmpl, err.Param())
	case reflect.Slice, reflect.Map, reflect.Array:
		return fmt.Sprintf(collectionTmpl, err.Param())
	default:
		return fmt.Sprintf(numericTmpl, err.Param())
	}
}

// DecodeJSON
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Define max JSON size
	const maxBodyBytes = 1 * 1024 * 1024

	// Check for "Content-Type" header
	ct := strings.TrimSpace(r.Header.Get("Content-Type"))
	if ct == "" {
		return apperr.NewUnsupportedMediaType(nil, ErrUnsupportedMediaType)
	}

	// Check for "application/json" header prefix, otherwise attempt parse
	if !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		mediaType, _, err := mime.ParseMediaType(ct)
		if err != nil || mediaType != "application/json" {
			return apperr.NewUnsupportedMediaType(err, ErrUnsupportedMediaType)
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
				return apperr.NewClientClosedRequest(r.Context().Err(), ErrClientClosedConnection)
			case errors.Is(r.Context().Err(), context.DeadlineExceeded):
				return apperr.NewRequestTimeout(r.Context().Err(), ErrReqTimeout)
			default:
				return apperr.NewClientClosedRequest(r.Context().Err(), ErrClientClosedConnection)
			}
		}

		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		// Check if payload is too large
		case errors.As(err, &maxBytesErr):
			return apperr.NewPayloadTooLarge(err, ErrPayloadTooLarge)

		// Check if JSON body is empty
		case errors.Is(err, io.EOF):
			return apperr.NewInvalidInput(err, ErrEmptyBody)

		// Check if JSON is malformed
		case errors.As(err, &syntaxErr):
			return apperr.NewInvalidInput(err, ErrMalformedJSON)

		// Check if JSON is truncated
		case errors.Is(err, io.ErrUnexpectedEOF):
			return apperr.NewInvalidInput(err, ErrTruncatedJSON)

		// Check if JSON field types are valid
		case errors.As(err, &unmarshalTypeErr):
			fieldName := unmarshalTypeErr.Field
			if fieldName == "" { // Client sends a raw string
				fieldName = "field"
			}

			return apperr.NewInvalidInput(
				err,
				ErrInvalidFieldType,
				apperr.Param(fieldName, fmt.Sprintf(ErrFieldTypeExpectation, unmarshalTypeErr.Type)),
			)

		// Check if JSON has unknown fields
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			fieldName = strings.Trim(fieldName, `"`)
			return apperr.NewInvalidInput(err, fmt.Sprintf(ErrUnknownField, fieldName))

		// Handle other errors
		default:
			return apperr.NewInternal(err, "")
		}
	}

	// Check for multiple JSON objects
	if dec.More() {
		return apperr.NewInvalidInput(nil, ErrSingleValueRequired)
	}

	return nil
}

// --- REQUEST BINDING FUNCTIONS ---

// BindQuery parses query strings into a struct, sanitizes, and validators it.
// Note: Unlike BindJSON, this does NOT need http.ResponseWriter because query parameters
// are already parsed into memory by the server and don't stream raw request bytes.
func BindQuery[T any](r *http.Request) (T, error) {
	var req T
	if err := DecodeQuery(r, &req); err != nil {
		return req, err
	}

	// Automagic sanitization hook if the DTO implements Sanitizable
	if s, ok := any(&req).(Sanitizable); ok {
		s.Sanitize()
	}

	// Validate the final populated struct
	if err := validate(&req); err != nil {
		return req, err
	}

	return req, nil
}

// DecodeQuery extracts URL query parameters and unmarshals them into the target destination.
func DecodeQuery(r *http.Request, dst any) error {
	// Ensure query parameters are fully parsed by the runtime
	if err := r.ParseForm(); err != nil {
		return apperr.NewInvalidInput(err, "Failed to parse query parameters.")
	}

	// Map the r.URL.Query() map[string][]string directly to the struct fields
	if err := formDecoder.Decode(dst, r.URL.Query()); err != nil {
		var decodeErrors form.DecodeErrors
		if errors.As(err, &decodeErrors) {
			// Extract the structural validation breakdown matching your UnmarshalTypeError pattern
			for field, fieldErr := range decodeErrors {
				return apperr.NewInvalidInput(
					err,
					"Invalid data type provided for query parameter(s).",
					apperr.Param(field, fmt.Sprintf("Must be a valid type: %v", fieldErr)),
				)
			}
		}
		return apperr.NewInvalidInput(err, "Malformed query parameters.")
	}

	return nil
}

// BindPath extracts URL path variables into a struct, sanitizes, and validators it.
func BindPath[T any](r *http.Request) (T, error) {
	var req T
	if err := DecodePath(r, &req); err != nil {
		return req, err
	}

	if s, ok := any(&req).(Sanitizable); ok {
		s.Sanitize()
	}

	if err := validate(&req); err != nil {
		return req, err
	}

	return req, nil
}

// DecodePath maps r.PathValue parameters to struct fields via the "path" tag.
func DecodePath(r *http.Request, dst any) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return apperr.NewInternal(nil, "DecodePath destination must be a pointer to a struct.")
	}

	elem := val.Elem()
	t := elem.Type()
	pathValues := make(url.Values)

	// Inspect struct fields for 'path' tags and populate standard url.Values
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("path")
		if tag == "" || tag == "-" {
			continue
		}

		if pathVal := r.PathValue(tag); pathVal != "" {
			pathValues.Set(tag, pathVal)
		}
	}

	if len(pathValues) == 0 {
		return nil
	}

	// Hand off to your optimized go-playground/form decoder for safe parsing/type conversion
	if err := formDecoder.Decode(dst, pathValues); err != nil {
		var decodeErrors form.DecodeErrors
		if errors.As(err, &decodeErrors) {
			for field, fieldErr := range decodeErrors {
				return apperr.NewInvalidInput(
					fieldErr,
					"Invalid data type provided in URL path parameters.",
					apperr.Param(field, fmt.Sprintf("Must be a valid type: %v", fieldErr)),
				)
			}
		}
		return apperr.NewInvalidInput(err, "Malformed path parameters.")
	}

	return nil
}
