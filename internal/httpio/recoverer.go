package httpio

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"bonfire-api/internal/pkg/errs"
)

// Recoverer returns a middleware that catches runtime panics occurring within downstream HTTP handlers.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rvr := recover()
			if rvr == nil {
				return
			}

			// 1. Allow standard HTTP handler aborts to propagate
			if rvr == http.ErrAbortHandler {
				panic(rvr)
			}

			// 2. Normalize panic payload into a standard error interface
			var err error
			if e, ok := rvr.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", rvr)
			}

			// 3. Handle broken pipe / socket disconnects gracefully without writing an error response
			if isBrokenPipe(err) {
				slog.WarnContext(r.Context(), "client connection closed abruptly",
					"error", err,
					"http.method", r.Method,
					"http.path", r.URL.Path,
				)
				return
			}

			// 4. Capture stack trace for internal telemetry
			stackTrace := string(debug.Stack())

			slog.ErrorContext(r.Context(), "catastrophic runtime panic recovered",
				"error.panic_message", err.Error(),
				"error.stack", stackTrace,
				"http.method", r.Method,
				"http.path", r.URL.Path,
			)

			// 5. Construct AIP-193 compliant internal error
			appErr := errs.Internal("").
				Wrap(err).
				ErrorInfoReason("PANIC_RECOVERED")

			respondError(w, r, appErr)
		}()

		next.ServeHTTP(w, r)
	})
}

// isBrokenPipe identifies socket resets or broken pipes where writing a response will fail.
func isBrokenPipe(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if sysErr, ok := netErr.Err.(*os.SyscallError); ok {
			errMsg := strings.ToLower(sysErr.Error())
			if strings.Contains(errMsg, "broken pipe") || strings.Contains(errMsg, "connection reset by peer") {
				return true
			}
		}
	}
	return false
}
