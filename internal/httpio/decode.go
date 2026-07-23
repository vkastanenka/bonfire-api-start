package httpio

import (
	"bonfire-api/internal/errs"
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

const maxJSONBodyBytes = 1 * 1024 * 1024 // 1MB

var (
	formDecoder = form.NewDecoder()
	pathDecoder = func() *form.Decoder {
		d := form.NewDecoder()
		d.SetTagName("path")
		return d
	}()
)

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errs.Internal("DecodeJSON destination must be a pointer to a struct.").
			Reason("INVALID_CODE_CALL")
	}

	ct := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil || mediaType != "application/json" {
		return errs.InvalidArgument("Missing or invalid Content-Type header; must be application/json.").
			Reason("INVALID_CONTENT_TYPE").
			Wrap(err)
	}

	limitedBody := http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	defer limitedBody.Close()

	dec := json.NewDecoder(limitedBody)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if ctxErr := r.Context().Err(); ctxErr != nil {
			if errors.Is(ctxErr, context.DeadlineExceeded) {
				return errs.DeadlineExceeded("Request timed out.").
					Reason("REQUEST_TIMEOUT").
					Wrap(ctxErr)
			}
			return errs.Aborted("Client closed connection mid-request.").
				Reason("CLIENT_CLOSED").
				Wrap(ctxErr)
		}

		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &maxBytesErr):
			return errs.ResourceExhausted("Request body exceeds 1MB limit.").
				Reason("BODY_TOO_LARGE").
				Wrap(err)

		case errors.Is(err, io.EOF):
			return errs.InvalidArgument("Request body cannot be empty.").
				Reason("EMPTY_REQUEST_BODY").
				FieldViolation("body", "Request body cannot be empty.", "REQUIRED").
				Wrap(err)

		case errors.As(err, &syntaxErr):
			return errs.InvalidArgument("Malformed request body JSON syntax.").
				Reason("MALFORMED_JSON").
				Wrap(err)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errs.InvalidArgument("Truncated or malformed JSON structure received.").
				Reason("TRUNCATED_JSON").
				Wrap(err)

		case errors.As(err, &unmarshalTypeErr):
			fieldName := unmarshalTypeErr.Field
			if fieldName == "" {
				fieldName = "body"
			}
			msg := fmt.Sprintf("Invalid data type provided for field '%s'. Expected %s.", fieldName, unmarshalTypeErr.Type)
			return errs.InvalidArgument(msg).
				Reason("INVALID_FIELD_TYPE").
				FieldViolation(fieldName, msg, "TYPE_MISMATCH").
				Wrap(err)

		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			fieldName = strings.Trim(fieldName, `"`)
			msg := fmt.Sprintf("Unknown field '%s' present in request body.", fieldName)
			return errs.InvalidArgument(msg).
				Reason("UNKNOWN_FIELD").
				FieldViolation(fieldName, msg, "UNEXPECTED_FIELD").
				Wrap(err)

		default:
			return errs.Internal("Failed to decode JSON request body.").
				Reason("JSON_DECODE_FAILED").
				Wrap(err)
		}
	}

	if dec.More() {
		return errs.InvalidArgument("Request body must contain only a single JSON value.").
			Reason("MULTIPLE_JSON_VALUES")
	}

	return nil
}

func decodeQuery(r *http.Request, dst any) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errs.Internal("DecodeQuery destination must be a pointer to a struct.").
			Reason("INVALID_CODE_CALL")
	}

	queryParams := r.URL.Query()
	if len(queryParams) == 0 {
		return nil
	}

	if err := formDecoder.Decode(dst, queryParams); err != nil {
		var decodeErrors form.DecodeErrors
		if errors.As(err, &decodeErrors) {
			e := errs.InvalidArgument("Invalid data type provided for query parameter(s).").
				Reason("INVALID_QUERY_PARAMS").
				Wrap(err)

			for field, fe := range decodeErrors {
				e.FieldViolation(field, fe.Error(), "INVALID_FORMAT")
			}
			return e
		}
		return errs.InvalidArgument("Malformed query parameters.").
			Reason("MALFORMED_QUERY_PARAMS").
			Wrap(err)
	}

	return nil
}

func decodePath(r *http.Request, dst any) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errs.Internal("DecodePath destination must be a pointer to a struct.").
			Reason("INVALID_CODE_CALL")
	}

	pathValues := make(url.Values)
	compilePathValues(val.Elem(), r, pathValues)

	if len(pathValues) == 0 {
		return nil
	}

	if err := pathDecoder.Decode(dst, pathValues); err != nil {
		var decodeErrors form.DecodeErrors
		if errors.As(err, &decodeErrors) {
			e := errs.InvalidArgument("Invalid data type provided in URL path parameters.").
				Reason("INVALID_PATH_PARAMS").
				Wrap(err)

			for field, fe := range decodeErrors {
				e.FieldViolation(field, fe.Error(), "INVALID_FORMAT")
			}
			return e
		}
		return errs.InvalidArgument("Malformed path parameters.").
			Reason("MALFORMED_PATH_PARAMS").
			Wrap(err)
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
