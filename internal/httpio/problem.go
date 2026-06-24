package httpio

import (
	"bonfire-api/internal/apperr"
	customMiddleware "bonfire-api/internal/middleware"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

var BaseDocURL = "https://api.bonfire.com/errors"

// ProblemDetails represents the transient, JSON-serialized RFC 7807 response
type ProblemDetails struct {
	Type          string                `json:"type"`
	Title         string                `json:"title"`
	Status        int                   `json:"status"`
	Detail        string                `json:"detail"`
	Instance      string                `json:"instance"`
	Code          apperr.Code           `json:"code"`
	InvalidParams []apperr.InvalidParam `json:"invalid_params,omitempty"`
	Timestamp     string                `json:"timestamp"`
	ReqID         string                `json:"req_id"`
	TraceID       string                `json:"trace_id"`
}

// MapToProblemDetails extracts core error data and hydrates request contextual fields safely
func MapToProblemDetails(r *http.Request, err *apperr.Error) (int, ProblemDetails) {
	status := err.Code.HTTPStatus()

	detail := err.Detail
	if err.Code == apperr.CodeInternal {
		detail = "An unexpected error occurred on our end."
	}

	reqID := middleware.GetReqID(r.Context())
	if reqID == "" {
		reqID = "unknown"
	}

	payload := ProblemDetails{
		Type:          fmt.Sprintf("%s/%s", BaseDocURL, err.Code.Slug()),
		Title:         err.Code.Title(),
		Status:        status,
		Detail:        detail,
		Instance:      r.URL.Path,
		Code:          err.Code,
		InvalidParams: err.InvalidParams,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		ReqID:         reqID,
		TraceID:       customMiddleware.GetTraceID(r.Context()),
	}

	return status, payload
}
