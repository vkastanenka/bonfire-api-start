package httpio

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	// 1. Prepare the payload buffer
	// Using a buffer prevents us from calling WriteHeader() before we know
	// the encoding succeeded.
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(data); err != nil {
		slog.Error("failed to encode json response", "error", err)
		// If we can't even encode, we must fallback to a hardcoded error
		http.Error(w, `{"error":"INTERNAL","message":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// 2. Set headers and status only after successful encoding
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// 3. Write the buffered payload
	_, _ = w.Write(buf.Bytes())
}

// TODO: Deprecate
func RespondText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// TODO: Deprecate
func RespondTextError(w http.ResponseWriter, r *http.Request, logMsg string, err error, status int, userMsg string) {
	reqID := middleware.GetReqID(r.Context())
	slog.ErrorContext(r.Context(), logMsg, "error", err, "reqID", reqID)
	RespondText(w, status, userMsg)
}
