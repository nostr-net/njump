package main

import (
	"context"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/fiatjaf/njump/i18n"
)

// timeoutResponseWriter wraps http.ResponseWriter to prevent writes after timeout
type timeoutResponseWriter struct {
	http.ResponseWriter
	mu          sync.Mutex
	timedOut    bool
	wroteHeader bool
}

func (tw *timeoutResponseWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut || tw.wroteHeader {
		return
	}
	tw.wroteHeader = true
	tw.ResponseWriter.WriteHeader(code)
}

func (tw *timeoutResponseWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	if !tw.wroteHeader {
		tw.wroteHeader = true
	}
	return tw.ResponseWriter.Write(b)
}

func (tw *timeoutResponseWriter) markTimedOut() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.timedOut = true
}

func timeoutMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isMetricsPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), requestTimeoutDuration())
		defer cancel()

		tw := &timeoutResponseWriter{ResponseWriter: w}
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
			// Request completed successfully
			return
		case <-ctx.Done():
			// Timeout reached
			tw.markTimedOut()
			recordTimeout(r)
			if ctx.Err() == context.DeadlineExceeded {
				// Only write retry page if handler hasn't written anything yet
				tw.mu.Lock()
				hasWritten := tw.wroteHeader
				tw.mu.Unlock()

				if !hasWritten {
					w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
					w.Header().Set("Pragma", "no-cache")
					w.Header().Set("Expires", "0")
					w.Header().Set("X-Robots-Tag", "noindex, nofollow")
					w.WriteHeader(http.StatusOK)

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
}

func requestTimeoutDuration() time.Duration {
	timeout := time.Duration(s.RequestTimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return timeout
}
