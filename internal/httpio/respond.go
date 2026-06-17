package httpio

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5/middleware"
)

// bufferPool reuses buffers to reduce memory allocations
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	// Always set JSON header
	w.Header().Set("Content-Type", "application/json")

	// Get a buffer from the pool
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	// Encode
	if err := json.NewEncoder(buf).Encode(data); err != nil {
		slog.Error("failed to encode json response", "error", err)

		// Fallback: Manual 500 status to ensure we stay JSON
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"INTERNAL","message":"An unexpected error occurred."}`))
		return
	}

	// Success: Commit status and write
	w.WriteHeader(status)
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
