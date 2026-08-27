package main

import (
	"bytes"
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/fiatjaf/njump/i18n"
)

// timeoutResponseWriter keeps a handler's response private until it completes.
// A timed-out handler may keep running briefly while it observes its cancelled
// context, so it must never retain access to the real ResponseWriter.
type timeoutResponseWriter struct {
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func (tw *timeoutResponseWriter) WriteHeader(code int) {
	if tw.wroteHeader {
		return
	}
	tw.wroteHeader = true
	tw.status = code
}

func (tw *timeoutResponseWriter) Write(b []byte) (int, error) {
	if !tw.wroteHeader {
		tw.WriteHeader(http.StatusOK)
	}
	return tw.body.Write(b)
}

func (tw *timeoutResponseWriter) Header() http.Header {
	return tw.header
}

func (tw *timeoutResponseWriter) writeTo(w http.ResponseWriter) {
	for key, values := range tw.header {
		w.Header()[key] = append([]string(nil), values...)
	}
	w.WriteHeader(tw.status)
	_, _ = w.Write(tw.body.Bytes())
}

func timeoutMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isMetricsPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), requestTimeoutDuration())
		defer cancel()

		tw := &timeoutResponseWriter{header: make(http.Header), status: http.StatusOK}
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() {
				if err := recover(); err != nil {
					recordPanic(r, "timeout")
					trace := trackError(r, err)
					log.Error().
						Any("panic", err).
						Str("path", r.URL.Path).
						Bytes("stack", debug.Stack()).
						Msg("panic recovered in timeout middleware")
					writeInternalServerError(tw, r, trace)
				}
			}()

			next.ServeHTTP(tw, r.WithContext(ctx))
		}()

		select {
		case <-done:
			tw.writeTo(w)
			return
		case <-ctx.Done():
			recordTimeout(r)
			if ctx.Err() == context.DeadlineExceeded {
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
				w.Header().Set("X-Robots-Tag", "noindex, nofollow")
				w.Header().Set("Retry-After", "5")
				w.WriteHeader(http.StatusServiceUnavailable)

				retryTemplate(RetryPageParams{
					HeadParams: HeadParams{
						Lang:   i18n.LanguageFromContext(r.Context()),
						Domain: domainFromCtx(r.Context()),
					},
				}).Render(r.Context(), w)
			}
		}
	}
}

func requestTimeoutDuration() time.Duration {
	timeout := time.Duration(s.RequestTimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return timeout
}
