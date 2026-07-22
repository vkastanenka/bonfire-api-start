package httpio

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type StatusRecordingWriter struct {
	http.ResponseWriter
	Status       int
	BytesWritten int64
	Hijacked     bool
}

func NewStatusRecordingWriter(w http.ResponseWriter) *StatusRecordingWriter {
	return &StatusRecordingWriter{
		ResponseWriter: w,
		Status:         0,
	}
}

func (w *StatusRecordingWriter) WriteHeader(code int) {
	if w.Hijacked {
		return
	}
	w.Status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *StatusRecordingWriter) Write(b []byte) (int, error) {
	if w.Status == 0 {
		w.Status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.BytesWritten += int64(n)
	return n, err
}

func (w *StatusRecordingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
	}

	w.Status = http.StatusSwitchingProtocols
	w.Hijacked = true

	return hj.Hijack()
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recordingWriter := NewStatusRecordingWriter(w)

		next.ServeHTTP(recordingWriter, r)

		statusCode := recordingWriter.Status
		if statusCode == 0 {
			if recordingWriter.BytesWritten > 0 {
				statusCode = http.StatusOK
			} else {
				statusCode = http.StatusOK
			}
		}

		slog.Info("http request processed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", statusCode,
			"latency_ms", time.Since(start).Milliseconds(),
			"bytes_written", recordingWriter.BytesWritten,
		)
	})
}
