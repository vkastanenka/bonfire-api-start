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

func BindJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var req T
	if err := decodeJSON(w, r, &req); err != nil {
		return req, err
	}

	return req, bindInput(&req)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return apperr.NewInternal(nil, "DecodeJSON destination must be a pointer to a struct.")
	}

	const maxJSONBodyBytes = 1 * 1024 * 1024

	ct := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil || mediaType != "application/json" {
		return apperr.NewUnsupportedMediaType(err, "Missing or invalid Content-Type header; must be application/json.")
	}

	limitedBody := http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	defer limitedBody.Close()

	dec := json.NewDecoder(limitedBody)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if ctxErr := r.Context().Err(); ctxErr != nil {
			if errors.Is(ctxErr, context.DeadlineExceeded) {
				return apperr.NewRequestTimeout(ctxErr, "Request timed out.")
			}

			return apperr.NewClientClosedRequest(ctxErr, "Client closed connection mid-request.")
		}

		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &maxBytesErr):
			return apperr.NewPayloadTooLarge(err, "Request body exceeds 1MB limit.")

		case errors.Is(err, io.EOF):
			return apperr.NewInvalidInput(err, "Request body cannot be empty.")

		case errors.As(err, &syntaxErr):
			return apperr.NewInvalidInput(err, "Malformed request body JSON syntax.")

		case errors.Is(err, io.ErrUnexpectedEOF):
			return apperr.NewInvalidInput(err, "Truncated or malformed JSON structure received.")

		case errors.As(err, &unmarshalTypeErr):
			fieldName := unmarshalTypeErr.Field
			if fieldName == "" {
				fieldName = "field"
			}

			return apperr.NewInvalidInput(
				err,
				"Invalid data type provided for request body field(s).",
				apperr.Param(fieldName, fmt.Sprintf("Must be of type %s", unmarshalTypeErr.Type)),
			)

		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			fieldName = strings.Trim(fieldName, `"`)
			return apperr.NewInvalidInput(err, fmt.Sprintf("Unknown field '%s' present in request body.", fieldName))

		default:
			return apperr.NewInternal(err, "")
		}
	}

	if dec.More() {
		return apperr.NewInvalidInput(nil, "Request body must contain only a single JSON value.")
	}

	return nil
}

func BindQuery[T any](r *http.Request) (T, error) {
	var req T
	if err := decodeQuery(r, &req); err != nil {
		return req, err
	}

	return req, bindInput(&req)
}

func decodeQuery(r *http.Request, dst any) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return apperr.NewInternal(nil, "DecodeQuery destination must be a pointer to a struct.")
	}

	queryParams := r.URL.Query()

	if err := formDecoder.Decode(dst, queryParams); err != nil {
		var decodeErrors form.DecodeErrors
		if errors.As(err, &decodeErrors) {
			params := make([]apperr.ErrorOption, 0, len(decodeErrors))
			for field, fieldErr := range decodeErrors {
				params = append(params, apperr.Param(field, fmt.Sprintf("Must be a valid type: %v", fieldErr)))
			}
			return apperr.NewInvalidInput(err, "Invalid data type provided for query parameter(s).", params...)
		}
		return apperr.NewInvalidInput(err, "Malformed query parameters.")
	}

	return nil
}

func BindPath[T any](r *http.Request) (T, error) {
	var req T
	if err := decodePath(r, &req); err != nil {
		return req, err
	}

	return req, bindInput(&req)
}

func decodePath(r *http.Request, dst any) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return apperr.NewInternal(nil, "DecodePath destination must be a pointer to a struct.")
	}

	pathValues := make(url.Values)
	compilePathValues(val.Elem(), r, pathValues)

	if len(pathValues) == 0 {
		return nil
	}

	if err := pathDecoder.Decode(dst, pathValues); err != nil {
		var decodeErrors form.DecodeErrors
		if errors.As(err, &decodeErrors) {
			params := make([]apperr.ErrorOption, 0, len(decodeErrors))
			for field, fieldErr := range decodeErrors {
				params = append(params, apperr.Param(field, fmt.Sprintf("Must be a valid type: %v", fieldErr)))
			}
			return apperr.NewInvalidInput(err, "Invalid data type provided in URL path parameters.", params...)
		}
		return apperr.NewInvalidInput(err, "Malformed path parameters.")
	}

	return nil
}

func compilePathValues(structVal reflect.Value, r *http.Request, output url.Values) {
	t := structVal.Type()

	for i := 0; i < t.NumField(); i++ {
		fieldType := t.Field(i)
		fieldVal := structVal.Field(i)

		if fieldType.Anonymous && fieldVal.Kind() == reflect.Struct {
			compilePathValues(fieldVal, r, output)
			continue
		}

		tag := fieldType.Tag.Get("path")
		if tag == "" || tag == "-" {
			continue
		}

		if pathVal := r.PathValue(tag); pathVal != "" {
			output.Set(tag, pathVal)
		}
	}
}

func bindInput[T any](req *T) error {
	sanitize.Normalize(req)
	return validate(req)
}
