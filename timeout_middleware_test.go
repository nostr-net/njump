package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeoutMiddlewareDoesNotExposeLateHandlerHeaders(t *testing.T) {
	previousTimeout := s.RequestTimeoutMs
	s.RequestTimeoutMs = 1
	t.Cleanup(func() { s.RequestTimeoutMs = previousTimeout })

	release := make(chan struct{})
	finished := make(chan struct{})
	handler := timeoutMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		w.Header().Set("X-Late-Handler", "must-not-reach-client")
		close(finished)
	})

	response := httptest.NewRecorder()
	handler(response, httptest.NewRequest(http.MethodGet, "http://example.com/note", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}

	close(release)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("timed-out handler did not finish")
	}
	if got := response.Header().Get("X-Late-Handler"); got != "" {
		t.Fatalf("late handler header reached client: %q", got)
	}
}
