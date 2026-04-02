package main

import (
	"net/http"
	"runtime/debug"
)

func recoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				recordPanic(r, "http")
				trace := trackError(r, err)
				log.Error().
					Any("panic", err).
					Str("path", r.URL.Path).
					Bytes("stack", debug.Stack()).
					Msg("panic recovered in http middleware")
				writeInternalServerError(w, r, trace)
			}
		}()

		next.ServeHTTP(w, r)
	}
}

func writeInternalServerError(w http.ResponseWriter, r *http.Request, trace []string) {
	if rw, ok := w.(*timeoutResponseWriter); ok {
		rw.mu.Lock()
		defer rw.mu.Unlock()
		if rw.timedOut || rw.wroteHeader {
			return
		}
		rw.wroteHeader = true
		rw.ResponseWriter.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		rw.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.ResponseWriter.Write([]byte("internal server error"))
		return
	}

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.Error(w, "internal server error", http.StatusInternalServerError)
	_ = trace
}
