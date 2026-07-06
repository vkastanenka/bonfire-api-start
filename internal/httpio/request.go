package httpio

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/sanitize"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/go-playground/form"
)

const (
	errUnsupportedMediaType   = "Missing or invalid Content-Type header; must be application/json."
	errClientClosedConnection = "Client closed connection mid-request."
	errPayloadTooLarge        = "Request body exceeds 1MB limit."
	errEmptyBody              = "Request body cannot be empty."
	errMalformedJSON          = "Malformed request body JSON syntax."
	errTruncatedJSON          = "Truncated or malformed JSON structure received."
	errInvalidPayload         = "Malformed or invalid request body JSON payload."
	errSingleValueRequired    = "Request body must contain only a single JSON value."
	errInvalidFieldType       = "Invalid data type provided for request body field(s)."
	errFieldTypeExpectation   = "Must be of type %s"
	errUnknownField           = "Unknown field '%s' present in request body."
	errReqTimeout             = "Request timed out."
	errMissingAuthCtx         = "Missing authentication context."
)

func BindJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var req T
	if err := DecodeJSON(w, r, &req); err != nil {
		return req, err
	}

	return req, bindInput(&req)
}

func BindQuery[T any](r *http.Request) (T, error) {
	var req T
	if err := DecodeQuery(r, &req); err != nil {
		return req, err
	}

	return req, bindInput(&req)
}

func BindPath[T any](r *http.Request) (T, error) {
	var req T
	if err := DecodePath(r, &req); err != nil {
		return req, err
	}

	return req, bindInput(&req)
}

func bindInput[T any](req *T) error {
	sanitize.Normalize(req)
	return validate(req)
}

const maxJSONBodyBytes = 1 * 1024 * 1024

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {

	ct := strings.TrimSpace(r.Header.Get("Content-Type"))
	if ct == "" {
		return apperr.NewUnsupportedMediaType(nil, errUnsupportedMediaType)
	}

	if !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		mediaType, _, err := mime.ParseMediaType(ct)
		if err != nil || mediaType != "application/json" {
			return apperr.NewUnsupportedMediaType(err, errUnsupportedMediaType)
		}
	}

	limitedBody := http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	defer limitedBody.Close()

	dec := json.NewDecoder(limitedBody)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if r.Context().Err() != nil {
			switch {
			case errors.Is(r.Context().Err(), context.Canceled):
				return apperr.NewClientClosedRequest(r.Context().Err(), errClientClosedConnection)
			case errors.Is(r.Context().Err(), context.DeadlineExceeded):
				return apperr.NewRequestTimeout(r.Context().Err(), errReqTimeout)
			default:
				return apperr.NewClientClosedRequest(r.Context().Err(), errClientClosedConnection)
			}
		}

		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &maxBytesErr):
			return apperr.NewPayloadTooLarge(err, errPayloadTooLarge)

		case errors.Is(err, io.EOF):
			return apperr.NewInvalidInput(err, errEmptyBody)

		case errors.As(err, &syntaxErr):
			return apperr.NewInvalidInput(err, errMalformedJSON)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return apperr.NewInvalidInput(err, errTruncatedJSON)

		case errors.As(err, &unmarshalTypeErr):
			fieldName := unmarshalTypeErr.Field
			if fieldName == "" {
				fieldName = "field"
			}

			return apperr.NewInvalidInput(
				err,
				errInvalidFieldType,
				apperr.Param(fieldName, fmt.Sprintf(errFieldTypeExpectation, unmarshalTypeErr.Type)),
			)

		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			fieldName = strings.Trim(fieldName, `"`)
			return apperr.NewInvalidInput(err, fmt.Sprintf(errUnknownField, fieldName))

		default:
			return apperr.NewInternal(err, "")
		}
	}

	if dec.More() {
		return apperr.NewInvalidInput(nil, errSingleValueRequired)
	}

	return nil
}

func DecodeQuery(r *http.Request, dst any) error {
	if err := r.ParseForm(); err != nil {
		return apperr.NewInvalidInput(err, "Failed to parse query parameters.")
	}

	if err := formDecoder.Decode(dst, r.URL.Query()); err != nil {
		var decodeErrors form.DecodeErrors
		if errors.As(err, &decodeErrors) {
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

func DecodePath(r *http.Request, dst any) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return apperr.NewInternal(nil, "DecodePath destination must be a pointer to a struct.")
	}

	elem := val.Elem()
	t := elem.Type()
	pathValues := make(url.Values)

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
