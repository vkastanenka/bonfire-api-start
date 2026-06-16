package httpio

import (
	"bonfire-api/internal/apperr"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5/middleware"
)

type ErrorResponse struct {
	Error   string         `json:"error"`
	Message string         `json:"message,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

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

func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"INTERNAL","message":"An unexpected internal error occurred."}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

func RespondText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func RespondTextError(w http.ResponseWriter, r *http.Request, logMsg string, err error, status int, userMsg string) {
	reqID := middleware.GetReqID(r.Context())
	slog.ErrorContext(r.Context(), logMsg, "error", err, "reqID", reqID)
	RespondText(w, status, userMsg)
}

func MapErrorToResponse(err error) (int, ErrorResponse) {
	var appErr *apperr.Error

	statusCode := http.StatusInternalServerError
	resp := ErrorResponse{
		Error:   string(apperr.CodeInternal),
		Message: "An unexpected internal error occurred.",
	}

	if errors.As(err, &appErr) {
		resp.Error = string(appErr.Code)
		resp.Message = appErr.Message
		resp.Details = appErr.Details

		switch appErr.Code {
		case apperr.CodeInvalidInput:
			statusCode = http.StatusBadRequest
		case apperr.CodeNotFound:
			statusCode = http.StatusNotFound
		case apperr.CodePayloadTooLarge:
			statusCode = http.StatusRequestEntityTooLarge
		case apperr.CodeConflict:
			statusCode = http.StatusConflict
		case apperr.CodeUnauthenticated:
			statusCode = http.StatusUnauthorized
		case apperr.CodeMethodNotAllowed:
			statusCode = http.StatusMethodNotAllowed
		case apperr.CodeInternal:
			statusCode = http.StatusInternalServerError
			resp.Message = "An unexpected internal error occurred."
			resp.Details = nil
		}
	}

	return statusCode, resp
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func ToHTTP(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			status, resp := MapErrorToResponse(err)

			reqID := middleware.GetReqID(r.Context())
			if reqID == "" {
				reqID = "unknown"
			}

			var appErr *apperr.Error
			if errors.As(err, &appErr) {
				if appErr.Code == apperr.CodeInternal {
					log.Printf("[CRITICAL] ReqID: %s | Msg: %s | Details: %v | InternalErr: %v",
						reqID, appErr.Message, appErr.Details, appErr.Err)
				} else {
					log.Printf("[INFO] ReqID: %s | UserError: %s | Msg: %s", reqID, appErr.Code, appErr.Message)
				}
			} else {
				log.Printf("[CRITICAL] ReqID: %s | Unhandled Error: %v", reqID, err)
			}

			RespondJSON(w, status, resp)
		}
	}
}

// resolveUnmarshalPath maps internal Go sub-struct names to their external
// camelCase JSON pathways to avoid leaking raw Go struct types to the client.
func resolveUnmarshalPath(structName, fieldName string) string {
	if structName == "" {
		return fieldName
	}

	// Map your internal Go struct types to their parent JSON keys.
	// As Bonfire grows, add your nested sub-structs here.
	switch structName {
	case "ProfileInfo", "ProfileData":
		return "profileInfo." + fieldName
	case "UserSettings", "Settings":
		return "settings." + fieldName
	case "ChannelPermissions":
		return "permissions." + fieldName
	default:
		// Fallback: lowercase the internal struct name as a sensible default
		// if you forget to add a explicit mapping later.
		if len(structName) > 0 {
			runes := []rune(structName)
			runes[0] = unicode.ToLower(runes[0]) // Native, clean rune translation
			return string(runes) + "." + fieldName
		}
		return fieldName
	}
}
