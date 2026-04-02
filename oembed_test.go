package main

import (
	"net/http/httptest"
	"testing"
)

func TestOembedTargetCodeExtractsFirstPathSegment(t *testing.T) {
	req := httptest.NewRequest("GET", "http://127.0.0.1/services/oembed?url=https%3A%2F%2Fnostr.ae%2Fnevent1qqqqqqqq%2Fembed", nil)

	got, err := oembedTargetCode(req)
	if err != nil {
		t.Fatalf("expected code, got error: %v", err)
	}
	if got != "nevent1qqqqqqqq" {
		t.Fatalf("expected nevent code, got %q", got)
	}
}

func TestOembedTargetCodeRejectsMissingURL(t *testing.T) {
	req := httptest.NewRequest("GET", "http://127.0.0.1/services/oembed", nil)

	if _, err := oembedTargetCode(req); err == nil {
		t.Fatal("expected missing url to fail")
	}
}
