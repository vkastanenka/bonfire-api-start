package httpio

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5/middleware"
)

// MaxPoolBufferCapacity prevents oversized buffers from polluting memory (64 KB)
const maxPoolBufferCapacity = 64 * 1024

var bufferPool = sync.Pool{
	New: func() interface{} {
		// Pre-allocate a reasonable starting size (e.g., 2 KB) to avoid early allocations
		return bytes.NewBuffer(make([]byte, 0, 2048))
	},
}

// RespondJSON marshals data and writes it securely to the wire.
// Pass r.Context() to ensure structured logs maintain distributed trace IDs.
func RespondJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	ctx := r.Context()

	w.Header().Set("Content-Type", "application/json")

	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	// Conditionally return the buffer to the pool to prevent memory bloat
	defer func() {
		if buf.Cap() <= maxPoolBufferCapacity {
			bufferPool.Put(buf)
		}
	}()

	if err := json.NewEncoder(buf).Encode(data); err != nil {
		// Uses ErrorContext to tie logs directly to your request span/ID
		slog.ErrorContext(ctx, "failed to encode json response", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"INTERNAL","message":"An unexpected error occurred."}`))
		return
	}

	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// ==========================================
// STANDARD ENVELOPES
// ==========================================

func RespondOK[T any](w http.ResponseWriter, r *http.Request, data T, message string) {
	RespondJSON(w, r, http.StatusOK, SuccessResponse[T]{
		Message: message,
		Data:    data,
	})
}

func RespondCreated[T any](w http.ResponseWriter, r *http.Request, data T, message string) {
	RespondJSON(w, r, http.StatusCreated, SuccessResponse[T]{
		Message: message,
		Data:    data,
	})
}

func RespondCursorList[T any](w http.ResponseWriter, r *http.Request, data T, meta CursorPagination) {
	RespondJSON(w, r, http.StatusOK, SuccessResponse[T]{
		Data: data,
		Meta: meta,
	})
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
