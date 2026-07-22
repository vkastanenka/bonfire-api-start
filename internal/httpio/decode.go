package httpio

import (
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

	"bonfire-api/internal/apperr"

	"github.com/go-playground/form"
)

const maxJSONBodyBytes = 1 * 1024 * 1024 // 1MB

var (
	formDecoder = form.NewDecoder()
	pathDecoder = func() *form.Decoder {
		d := form.NewDecoder()
		d.SetTagName("path")
		return d
	}()
)

// decodeJSON reads and parses the JSON request body into dst.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return apperr.NewInternal(
			nil,
			apperr.WithMsg("DecodeJSON destination must be a pointer to a struct."),
		)
	}

	ct := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil || mediaType != "application/json" {
		return apperr.NewInvalidArgument(
			err,
			apperr.WithMsg("Missing or invalid Content-Type header; must be application/json."),
		)
	}

	limitedBody := http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	defer limitedBody.Close()

	dec := json.NewDecoder(limitedBody)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if ctxErr := r.Context().Err(); ctxErr != nil {
			if errors.Is(ctxErr, context.DeadlineExceeded) {
				return apperr.NewDeadlineExceeded(ctxErr, apperr.WithMsg("Request timed out."))
			}
			return apperr.NewAborted(ctxErr, apperr.WithMsg("Client closed connection mid-request."))
		}

		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &maxBytesErr):
			return apperr.NewResourceExhausted(err, apperr.WithMsg("Request body exceeds 1MB limit."))

		case errors.Is(err, io.EOF):
			return apperr.NewInvalidArgument(err, apperr.WithMsg("Request body cannot be empty."))

		case errors.As(err, &syntaxErr):
			return apperr.NewInvalidArgument(err, apperr.WithMsg("Malformed request body JSON syntax."))

		case errors.Is(err, io.ErrUnexpectedEOF):
			return apperr.NewInvalidArgument(err, apperr.WithMsg("Truncated or malformed JSON structure received."))

		case errors.As(err, &unmarshalTypeErr):
			fieldName := unmarshalTypeErr.Field
			if fieldName == "" {
				fieldName = "field"
			}
			return apperr.NewInvalidArgument(
				err,
				apperr.WithMsg(fmt.Sprintf("Invalid data type provided for field '%s'. Expected %s.", fieldName, unmarshalTypeErr.Type)),
			)

		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			fieldName = strings.Trim(fieldName, `"`)
			return apperr.NewInvalidArgument(
				err,
				apperr.WithMsg(fmt.Sprintf("Unknown field '%s' present in request body.", fieldName)),
			)

		default:
			return apperr.NewInternal(err, apperr.WithMsg("Failed to decode JSON request body."))
		}
	}

	if dec.More() {
		return apperr.NewInvalidArgument(
			nil,
			apperr.WithMsg("Request body must contain only a single JSON value."),
		)
	}

	return nil
}

// decodeQuery extracts URL query parameters into dst using the form decoder.
func decodeQuery(r *http.Request, dst any) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return apperr.NewInternal(
			nil,
			apperr.WithMsg("DecodeQuery destination must be a pointer to a struct."),
		)
	}

	queryParams := r.URL.Query()
	if len(queryParams) == 0 {
		return nil
	}

	if err := formDecoder.Decode(dst, queryParams); err != nil {
		var decodeErrors form.DecodeErrors
		if errors.As(err, &decodeErrors) {
			return apperr.NewInvalidArgument(
				err,
				apperr.WithMsg("Invalid data type provided for query parameter(s)."),
			)
		}
		return apperr.NewInvalidArgument(err, apperr.WithMsg("Malformed query parameters."))
	}

	return nil
}

// decodePath extracts URL path parameters into dst using Chi or standard Go 1.22+ mux path parameters.
func decodePath(r *http.Request, dst any) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return apperr.NewInternal(
			nil,
			apperr.WithMsg("DecodePath destination must be a pointer to a struct."),
		)
	}

	pathValues := make(url.Values)
	compilePathValues(val.Elem(), r, pathValues)

	if len(pathValues) == 0 {
		return nil
	}

	if err := pathDecoder.Decode(dst, pathValues); err != nil {
		var decodeErrors form.DecodeErrors
		if errors.As(err, &decodeErrors) {
			return apperr.NewInvalidArgument(
				err,
				apperr.WithMsg("Invalid data type provided in URL path parameters."),
			)
		}
		return apperr.NewInvalidArgument(err, apperr.WithMsg("Malformed path parameters."))
	}

	return nil
}

// compilePathValues recursively inspects struct fields for `path:"..."` tags and fetches values from Go 1.22+ `r.PathValue(...)`.
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
