package apperr

import (
	"bonfire-api/internal/repository"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgconn"
)

// BaseDocURL defines the namespace hosting documentation detailing specific errors
var BaseDocURL = "https://api.bonfire.com/errors"

// InvalidParam represents precise parameter-level validation failures compliant with RFC extensions
type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Error models an RFC 7807 structured problem detail payload
type Error struct {
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail"`
	Instance      string         `json:"instance"`
	Code          Code           `json:"code"`
	InvalidParams []InvalidParam `json:"invalid_params,omitempty"`
	Timestamp     string         `json:"timestamp"`
	ReqID         string         `json:"req_id"`
	TraceID       string         `json:"trace_id"`
	Err           error          `json:"-"` // Omit the raw underlying root error string from JSON serialization
}

// Assert compile-time correctness for the error interface assignment
var _ error = (*Error)(nil)

// New initializes an application domain error model
func New(code Code, detail string, opts ...ErrorOption) error {
	e := &Error{
		Code:   code,
		Detail: detail,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// NewDBError translates raw database anomalies and row-omission exceptions (404s)
// into unified, structured RFC 7807 application domain errors.
func NewDBError(err error, entityName ...string) error {
	if err == nil {
		return nil
	}

	// 1. Resolve a friendly noun for the entity context (defaults to "Resource")
	entity := "Resource"
	if len(entityName) > 0 && entityName[0] != "" {
		entity = entityName[0]
	}

	// 2. Intercept "Not Found" exceptions natively (e.g., pgx.ErrNoRows / sqlc targets)
	if repository.IsNotFoundError(err) {
		return New(
			CodeNotFound,
			fmt.Sprintf("%s could not be found.", entity),
			WithErr(err),
		)
	}

	// 3. Inspect specific structural PostgreSQL constraints
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {

		// --- Integrity & Relationship Violations ---
		case "23505": // unique_violation
			return New(
				CodeConflict,
				fmt.Sprintf("A conflict occurred. This %s already exists.", entity),
				WithErr(err),
			)
		case "23503": // foreign_key_violation
			return New(CodeInvalidInput, "A referenced record does not exist.", WithErr(err))
		case "23502": // not_null_violation
			return New(CodeInvalidInput, "A required field is missing.", WithErr(err))
		case "23514": // check_violation
			return New(CodeInvalidInput, "The provided data failed validation rules.", WithErr(err))

		// --- Data Formatting Exceptions ---
		case "22001": // string_data_right_truncation
			return New(CodeInvalidInput, "A provided text field exceeds the maximum allowed length.", WithErr(err))
		case "22003": // numeric_value_out_of_range
			return New(CodeInvalidInput, "A provided number is out of the acceptable range.", WithErr(err))
		case "22P02": // invalid_text_representation (e.g., bad UUIDs)
			return New(CodeInvalidInput, "The data format is invalid or malformed.", WithErr(err))

		// --- Concurrency & Locks ---
		case "40001": // serialization_failure
			return New(CodeConflict, "The system is busy. Please retry your request.", WithErr(err))
		case "40P01": // deadlock_detected
			return New(CodeConflict, "A resource conflict occurred. Please retry your request.", WithErr(err))

		// --- Timeouts ---
		case "57014": // query_canceled
			return New(CodeRequestTimeout, "The database operation timed out or was canceled.", WithErr(err))
		}
	}

	// Default fallback for unmapped systemic or network connection infrastructure faults
	return New(CodeInternal, CodeInternal.Title(), WithErr(err))
}

// Error converts the internal model values to an explicit debugging line string for console logs
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s (%s): %v", e.Code, e.Title, e.Detail, e.Err)
	}
	return fmt.Sprintf("[%s] %s (%s)", e.Code, e.Title, e.Detail)
}

// Unwrap handles nested native standard error unbundling chains
func (e *Error) Unwrap() error {
	return e.Err
}

// Option configures targeted attributes on an instantiated Error object instance
type ErrorOption func(*Error)

// WithErr couples lower-level execution context or database failures cleanly
func WithErr(err error) ErrorOption {
	return func(e *Error) {
		e.Err = err
	}
}

// WithInvalidParam appends a single distinct input parameters error to the parameter slice tracking validation bugs
func WithInvalidParam(name, reason string) ErrorOption {
	return func(e *Error) {
		e.InvalidParams = append(e.InvalidParams, InvalidParam{
			Name:   name,
			Reason: reason,
		})
	}
}

// WithInvalidParams chains whole structural groups of batch evaluation results directly
func WithInvalidParams(params []InvalidParam) ErrorOption {
	return func(e *Error) {
		e.InvalidParams = append(e.InvalidParams, params...)
	}
}

// ErrorCode checks any basic generic error wrapper chain to isolate the app specific error identity token
func ErrorCode(err error) Code {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return CodeInternal
}

// ToResponse processes the error model state right before JSON parsing to protect core debugging logs
func (e *Error) ToResponse() *Error {
	// Sanitize 500 error variations so internal implementation traces don't escape to client layers
	if e.Code == CodeInternal {
		return &Error{
			Type:      fmt.Sprintf("%s/%s", BaseDocURL, CodeInternal.Slug()),
			Title:     CodeInternal.Title(),
			Status:    CodeInternal.HTTPStatus(),
			Detail:    "An unexpected error occurred on our end.",
			Instance:  e.Instance,
			Code:      CodeInternal,
			Timestamp: e.Timestamp,
			ReqID:     e.ReqID,
			TraceID:   e.TraceID,
		}
	}
	return e
}

// HTTPResponse extracts metadata tracking targets required to pipe values accurately into transport layer pipelines
func (e *Error) HTTPResponse() (int, *Error) {
	return e.Code.HTTPStatus(), e.ToResponse()
}

// Enrich binds HTTP contextual state directly to structural attributes right before writing response streams
func (e *Error) Enrich(r *http.Request) {
	e.Instance = r.URL.Path
	e.Timestamp = time.Now().UTC().Format(time.RFC3339)

	// Get req id
	reqID := middleware.GetReqID(r.Context())
	if reqID == "" {
		reqID = "unknown"
	}
	e.ReqID = reqID

	// Pull an incoming distributed trace identity string or safely default to context middleware state indicators
	traceID := r.Header.Get("X-Trace-ID")
	if traceID == "" {
		traceID = generateW3CTraceID()
	}
	e.TraceID = traceID

	// Fallback hydration routines if base attributes were left unconfigured during standard init instantiation sequences
	if e.Type == "" {
		e.Type = fmt.Sprintf("%s/%s", BaseDocURL, e.Code.Slug())
	}
	if e.Title == "" {
		e.Title = e.Code.Title()
	}
	if e.Status == 0 {
		e.Status = e.Code.HTTPStatus()
	}
}

// generateW3CTraceID produces 16 random hex bytes ensuring compliance with distributed monitoring systems
func generateW3CTraceID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "00000000000000000000000000000000" // Fallback trace zero boundary indicator if entropy fails
	}
	return hex.EncodeToString(b)
}
