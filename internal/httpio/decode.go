package httpio

import (
	"bonfire-api/internal/apperr"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
)

// Define max JSON size
const maxBodyBytes = 1 * 1024 * 1024

// DecodeJSON reads an incoming HTTP request body, enforces secure body sizes,
// strictly validates its format and schema properties, and unpacks it into the destination.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Check for "Content-Type" header
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return apperr.New(apperr.CodeUnsupportedMediaType, UnsupportedMediaTypeMsg)
	}

	// Check for "application/json" header prefix, otherwise attempt parse
	if !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		mediaType, _, err := mime.ParseMediaType(ct)
		if err != nil || mediaType != "application/json" {
			return apperr.New(apperr.CodeUnsupportedMediaType, UnsupportedMediaTypeMsg, apperr.WithErr(err))
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
				return apperr.New(apperr.CodeClientClosedRequest, ClientClosedConnectionMsg, apperr.WithErr(r.Context().Err()))
			case errors.Is(r.Context().Err(), context.DeadlineExceeded):
				return apperr.New(apperr.CodeRequestTimeout, ReqTimeoutMsg, apperr.WithErr(r.Context().Err()))
			default:
				return apperr.New(apperr.CodeClientClosedRequest, ClientClosedConnectionMsg, apperr.WithErr(r.Context().Err()))
			}
		}

		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		// Check if payload is too large
		case errors.As(err, &maxBytesErr):
			return apperr.New(apperr.CodePayloadTooLarge, PayloadTooLargeMsg, apperr.WithErr(err))

		// Check if JSON body is empty
		case errors.Is(err, io.EOF):
			return apperr.New(apperr.CodeInvalidInput, EmptyBodyMsg, apperr.WithErr(err))

		// Check if JSON is malformed
		case errors.As(err, &syntaxErr):
			return apperr.New(apperr.CodeInvalidInput, MalformedJSONMsg, apperr.WithErr(err))

		// Check if JSON is truncated
		case errors.Is(err, io.ErrUnexpectedEOF):
			return apperr.New(apperr.CodeInvalidInput, TruncatedJSONMsg, apperr.WithErr(err))

		// Check if JSON field types are valid
		case errors.As(err, &unmarshalTypeErr):
			fieldName := unmarshalTypeErr.Field
			if fieldName == "" { // Client sends a raw string
				fieldName = "field"
			}

			return apperr.New(
				apperr.CodeInvalidInput,
				InvalidFieldTypeMsg,
				apperr.WithDetails(fieldName, fmt.Sprintf(FieldTypeExpectationFmt, unmarshalTypeErr.Type)),
				apperr.WithErr(err),
			)

		// Check if JSON has unknown fields
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			fieldName = strings.Trim(fieldName, `"`)
			return apperr.New(apperr.CodeInvalidInput, fmt.Sprintf(UnknownFieldFmt, fieldName), apperr.WithErr(err))

		// Handle other errors
		default:
			return apperr.New(apperr.CodeInternal, DecodeErrMsg, apperr.WithErr(err))
		}
	}

	// Check for multiple JSON objects
	if dec.More() {
		return apperr.New(apperr.CodeInvalidInput, SingleValueRequiredMsg)
	}

	return nil
}
