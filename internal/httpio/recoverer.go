package httpio

import (
	"bonfire-api/internal/errs"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					panic(rvr)
				}

				stackTrace := debug.Stack()

				var err error
				if e, ok := rvr.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("%v", rvr)
				}

				appErr := &errs.Error{
					Code: errs.CodeInternal,
					Err:  err,
				}

				slog.ErrorContext(r.Context(), "catastrophic runtime panic recovered",
					"error.panic_message", err.Error(),
					"error.stack", string(stackTrace),
					"http.method", r.Method,
					"http.path", r.URL.Path,
				)

				respondError(w, r, appErr)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
