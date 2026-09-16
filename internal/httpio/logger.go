package httpio

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"time"

	"bonfire-api/internal/pkg/errs"
)

// StatusRecordingWriter intercepts HTTP response metrics like status code and byte count.
type StatusRecordingWriter struct {
	http.ResponseWriter
	Status       int
	BytesWritten int64
	Hijacked     bool
}

// NewStatusRecordingWriter wraps an existing http.ResponseWriter.
func NewStatusRecordingWriter(w http.ResponseWriter) *StatusRecordingWriter {
	return &StatusRecordingWriter{
		ResponseWriter: w,
		Status:         0,
	}
}

// WriteHeader records the status code before writing headers to the underlying writer.
func (w *StatusRecordingWriter) WriteHeader(code int) {
	if w.Hijacked {
		return
	}
	w.Status = code
	w.ResponseWriter.WriteHeader(code)
}

// Write captures bytes written and defaults status code to 200 OK if unwritten.
func (w *StatusRecordingWriter) Write(b []byte) (int, error) {
	if w.Status == 0 {
		w.Status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.BytesWritten += int64(n)
	return n, err
}

// Flush proxies response flushing if supported by the underlying writer (e.g., SSE/streaming).
func (w *StatusRecordingWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack upgrades the connection protocol (e.g., WebSockets) while updating recording state.
func (w *StatusRecordingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errs.Internal("underlying ResponseWriter does not support hijacking")
	}

	w.Status = http.StatusSwitchingProtocols
	w.Hijacked = true

	return hj.Hijack()
}

// Logger records request duration, status code, byte throughput, and context tracing data.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recordingWriter := NewStatusRecordingWriter(w)

		next.ServeHTTP(recordingWriter, r)

		statusCode := recordingWriter.Status
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		duration := time.Since(start)

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", statusCode,
			"latency_ms", duration.Milliseconds(),
			"bytes_written", recordingWriter.BytesWritten,
		}

		ctx := r.Context()

		// Dynamically adjust log level based on response status
		switch {
		case statusCode >= 500:
			slog.ErrorContext(ctx, "http request failed", attrs...)
		case statusCode >= 400:
			slog.WarnContext(ctx, "http request client error", attrs...)
		default:
			slog.InfoContext(ctx, "http request processed", attrs...)
		}
	})
}
