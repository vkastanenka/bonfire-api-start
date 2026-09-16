package httpio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"

	"bonfire-api/internal/pkg/errs"

	"github.com/go-playground/form"
)

const maxJSONBodyBytes = 1 * 1024 * 1024

var (
	queryDecoder = func() *form.Decoder {
		d := form.NewDecoder()
		d.SetTagName("query")
		return d
	}()

	pathDecoder = func() *form.Decoder {
		d := form.NewDecoder()
		d.SetTagName("path")
		return d
	}()

	pathTagCache sync.Map
)

// validateDestination checks that dst is a non-nil pointer to a struct.
func validateDestination(dst any, funcName string) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr || val.IsNil() || val.Elem().Kind() != reflect.Struct {
		return errs.Internal("Destination must be a non-nil pointer to a struct.").
			ErrorInfoReason("INVALID_CODE_CALL").
			ErrorInfoMeta("function_name", funcName)
	}
	return nil
}

// decodeJSON reads and decodes a JSON request body into dst, enforcing payload limits,
// standard content types, single-value restrictions, and AIP-compliant error metadata.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if err := validateDestination(dst, "DecodeJSON"); err != nil {
		return err
	}

	ct := r.Header.Get("Content-Type")
	mediaType, _, _ := strings.Cut(ct, ";")
	if strings.TrimSpace(strings.ToLower(mediaType)) != "application/json" {
		return errs.InvalidArgument("Missing or invalid Content-Type header; must be application/json.").
			ErrorInfoReason("INVALID_CONTENT_TYPE").
			ErrorInfoMeta("expected_content_type", "application/json")
	}

	limitedBody := http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	defer limitedBody.Close()

	dec := json.NewDecoder(limitedBody)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		if ctxErr := r.Context().Err(); ctxErr != nil {
			if errors.Is(ctxErr, context.DeadlineExceeded) {
				return errs.DeadlineExceeded("Request timed out.").
					ErrorInfoReason("REQUEST_TIMEOUT").
					Wrap(ctxErr)
			}
			return errs.Cancelled("Client cancelled the request.").
				ErrorInfoReason("CLIENT_CANCELLED").
				Wrap(ctxErr)
		}

		var maxBytesErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &maxBytesErr):
			return errs.ResourceExhausted("Request body exceeds size limit.").
				ErrorInfoReason("BODY_TOO_LARGE").
				ErrorInfoMeta("max_bytes", fmt.Sprintf("%d", maxJSONBodyBytes)).
				Wrap(err)

		case errors.Is(err, io.EOF):
			return errs.InvalidArgument("Request body cannot be empty.").
				ErrorInfoReason("EMPTY_REQUEST_BODY").
				FieldViolation("body", "Request body cannot be empty.", "REQUIRED").
				Wrap(err)

		case errors.As(err, &syntaxErr):
			return errs.InvalidArgument("Malformed request body JSON syntax.").
				ErrorInfoReason("MALFORMED_JSON").
				Wrap(err)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errs.InvalidArgument("Truncated or malformed JSON structure received.").
				ErrorInfoReason("TRUNCATED_JSON").
				Wrap(err)

		case errors.As(err, &unmarshalTypeErr):
			fieldName := unmarshalTypeErr.Field
			if fieldName == "" {
				fieldName = "body"
			}

			expectedType := "unknown"
			if unmarshalTypeErr.Type != nil {
				expectedType = unmarshalTypeErr.Type.String()
			}

			return errs.InvalidArgument("Invalid data type provided for field.").
				ErrorInfoReason("INVALID_FIELD_TYPE").
				ErrorInfoMeta("field", fieldName).
				ErrorInfoMeta("expected_type", expectedType).
				FieldViolation(fieldName, "Invalid data type provided for field.", "TYPE_MISMATCH").
				Wrap(err)

		case strings.HasPrefix(err.Error(), "json: unknown field"):
			rawField := strings.TrimPrefix(err.Error(), "json: unknown field ")
			fieldName := strings.Trim(rawField, `"`)

			return errs.InvalidArgument("Request payload contains unrecognized fields.").
				ErrorInfoReason("UNEXPECTED_FIELD").
				ErrorInfoMeta("field", fieldName).
				FieldViolation(fieldName, "Unknown field present in request body.", "UNEXPECTED_FIELD").
				Wrap(err)

		default:
			return errs.Internal("Failed to decode JSON request body.").
				ErrorInfoReason("JSON_DECODE_FAILED").
				Wrap(err)
		}
	}

	if dec.More() {
		return errs.InvalidArgument("Request body must contain only a single JSON value.").
			ErrorInfoReason("MULTIPLE_JSON_VALUES")
	}

	return nil
}

// decodeQuery decodes URL query parameters into fields tagged with "query" on dst.
func decodeQuery(r *http.Request, dst any) error {
	if err := validateDestination(dst, "DecodeQuery"); err != nil {
		return err
	}

	queryParams := r.URL.Query()
	if len(queryParams) == 0 {
		return nil
	}

	if err := queryDecoder.Decode(dst, queryParams); err != nil {
		return mapFormDecodeError(err, "INVALID_QUERY_PARAMS", "MALFORMED_QUERY_PARAMS", "Invalid data type provided for query parameter(s).", "Malformed query parameters.")
	}

	return nil
}

// decodePath extracts standard HTTP path parameters matching "path" tags on dst and decodes them into the struct.
func decodePath(r *http.Request, dst any) error {
	if err := validateDestination(dst, "DecodePath"); err != nil {
		return err
	}

	structType := reflect.TypeOf(dst).Elem()
	tags := getOrExtractPathTags(structType)

	if len(tags) == 0 {
		return nil
	}

	pathValues := make(url.Values, len(tags))
	for _, tag := range tags {
		if pathVal := r.PathValue(tag); pathVal != "" {
			pathValues.Set(tag, pathVal)
		}
	}

	if len(pathValues) == 0 {
		return nil
	}

	if err := pathDecoder.Decode(dst, pathValues); err != nil {
		return mapFormDecodeError(err, "INVALID_PATH_PARAMS", "MALFORMED_PATH_PARAMS", "Invalid data type provided in URL path parameters.", "Malformed path parameters.")
	}

	return nil
}

// mapFormDecodeError converts a form decoding error into an InvalidArgument error with itemized field violations.
func mapFormDecodeError(err error, invalidReason, malformedReason, invalidMsg, malformedMsg string) error {
	var decodeErrors form.DecodeErrors
	if errors.As(err, &decodeErrors) {
		e := errs.InvalidArgument(invalidMsg).
			ErrorInfoReason(invalidReason).
			Wrap(err)

		for field, fe := range decodeErrors {
			e.FieldViolation(field, fe.Error(), "INVALID_FORMAT")
		}
		return e
	}

	return errs.InvalidArgument(malformedMsg).
		ErrorInfoReason(malformedReason).
		Wrap(err)
}

// getOrExtractPathTags parses "path" struct tags recursively and caches them by reflection type.
func getOrExtractPathTags(t reflect.Type) []string {
	if cached, ok := pathTagCache.Load(t); ok {
		return cached.([]string)
	}

	var tags []string
	var extract func(reflect.Type)
	extract = func(typ reflect.Type) {
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct {
			return
		}

		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if field.Anonymous {
				extract(field.Type)
				continue
			}

			tag := field.Tag.Get("path")
			if tag != "" && tag != "-" {
				tagName := strings.Split(tag, ",")[0]
				if tagName != "" {
					tags = append(tags, tagName)
				}
			}
		}
	}

	extract(t)
	actual, _ := pathTagCache.LoadOrStore(t, tags)
	return actual.([]string)
}
