package httpio

import (
	"bonfire-api/internal/apperr"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
)

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return apperr.New(apperr.CodeInvalidInput, "Missing Content-Type header; must be application/json.")
	}

	ctLower := strings.ToLower(ct)
	if !strings.HasPrefix(ctLower, "application/json") {
		mediaType, _, err := mime.ParseMediaType(ct)
		if err != nil || mediaType != "application/json" {
			return apperr.New(apperr.CodeInvalidInput, "Content-Type header must be application/json.")
		}
	}

	// 1MB standard buffer ceiling
	limitedBody := http.MaxBytesReader(w, r.Body, 1048576)
	defer limitedBody.Close()

	dec := json.NewDecoder(limitedBody)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if r.Context().Err() != nil {
			return apperr.New(apperr.CodeInvalidInput, "Client closed connection mid-request.", apperr.WithErr(r.Context().Err()))
		}

		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &maxBytesErr):
			return apperr.New(apperr.CodePayloadTooLarge, "Request body exceeds 1MB limit.", apperr.WithErr(err))

		case errors.Is(err, io.EOF):
			return apperr.New(apperr.CodeInvalidInput, "Request body cannot be empty.", apperr.WithErr(err))

		case errors.As(err, &syntaxErr):
			return apperr.New(apperr.CodeInvalidInput, "Malformed request body JSON syntax.", apperr.WithErr(err))

		case errors.Is(err, io.ErrUnexpectedEOF):
			return apperr.New(apperr.CodeInvalidInput, "Truncated or malformed JSON structure received.", apperr.WithErr(err))

		case errors.As(err, &unmarshalTypeErr):
			fieldName := unmarshalTypeErr.Field
			if fieldName == "" {
				fieldName = "field"
			} else {
				// Resolve the full structural path if the type mismatch occurred inside a nested struct
				fieldName = resolveUnmarshalPath(unmarshalTypeErr.Struct, fieldName)
			}

			return apperr.New(
				apperr.CodeInvalidInput,
				"Invalid data type provided for request body field(s).",
				apperr.WithDetails(fieldName, fmt.Sprintf("Must be of type %s", unmarshalTypeErr.Type)),
				apperr.WithErr(err),
			)

		// Safe string checking fallback for unknown JSON fields
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			fieldName = strings.Trim(fieldName, `"`)
			return apperr.New(apperr.CodeInvalidInput, fmt.Sprintf("Unknown field '%s' present in request body.", fieldName), apperr.WithErr(err))

		default:
			return apperr.New(apperr.CodeInvalidInput, "Malformed or invalid request body JSON payload.", apperr.WithErr(err))
		}
	}

	if dec.More() {
		return apperr.New(apperr.CodeInvalidInput, "Request body must contain only a single JSON value.")
	}

	return nil
}
