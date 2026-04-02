package main

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestMetricsDomainLabelUsesContextDomain(t *testing.T) {
	s.Domain = "fallback.example"

	req := httptest.NewRequest("GET", "http://127.0.0.1/resource", nil)
	req = req.WithContext(context.WithValue(req.Context(), "domain", "nostr.ae"))

	if got := metricsDomainLabel(req); got != "nostr.ae" {
		t.Fatalf("expected context domain, got %q", got)
	}
}

func TestMetricsDomainLabelNormalizesHost(t *testing.T) {
	s.Domain = "fallback.example"

	req := httptest.NewRequest("GET", "http://127.0.0.1/resource", nil)
	req.Host = "www.Nostr.At:443"

	if got := metricsDomainLabel(req); got != "nostr.at" {
		t.Fatalf("expected normalized host, got %q", got)
	}
}

func TestMetricsDomainLabelFallsBackToDefaultDomain(t *testing.T) {
	s.Domain = "fallback.example"

	req := httptest.NewRequest("GET", "http://127.0.0.1/resource", nil)
	req.Host = ""

	if got := metricsDomainLabel(req); got != "fallback.example" {
		t.Fatalf("expected fallback domain, got %q", got)
	}
}

func TestIsMetricsPath(t *testing.T) {
	cases := map[string]bool{
		"/metrics":          true,
		"/metrics/nostr-ae": true,
		"/metrics/nostr-at": true,
		"/e/test":           false,
		"/":                 false,
	}

	for path, want := range cases {
		if got := isMetricsPath(path); got != want {
			t.Fatalf("path %q: expected %v, got %v", path, want, got)
		}
	}
}
