package main

import (
	"context"
	"os"
	"testing"
)

func TestDefaultLangForDomain_KnownDomain(t *testing.T) {
	orig := domainConfigs
	t.Cleanup(func() { domainConfigs = orig })
	domainConfigs = map[string]DomainConfig{
		"nostr.ae": {DefaultLanguage: "ar"},
		"nostr.at": {DefaultLanguage: "en"},
	}
	if got := defaultLangForDomain("nostr.ae"); got != "ar" {
		t.Fatalf("want ar, got %s", got)
	}
}

func TestDefaultLangForDomain_UnknownDomain_FallsBackToGlobal(t *testing.T) {
	origCfg := domainConfigs
	origLang := s.DefaultLanguage
	t.Cleanup(func() {
		domainConfigs = origCfg
		s.DefaultLanguage = origLang
	})
	domainConfigs = map[string]DomainConfig{}
	s.DefaultLanguage = "en"
	if got := defaultLangForDomain("unknown.example"); got != "en" {
		t.Fatalf("want en (global fallback), got %s", got)
	}
}

func TestDomainFromCtx_ReturnsDomainSetInContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "domain", "nostr.ae")
	if got := domainFromCtx(ctx); got != "nostr.ae" {
		t.Fatalf("want nostr.ae, got %s", got)
	}
}

func TestDomainFromCtx_FallsBackToGlobalDomain(t *testing.T) {
	origDomain := s.Domain
	t.Cleanup(func() { s.Domain = origDomain })
	s.Domain = "njump.me"
	if got := domainFromCtx(context.Background()); got != "njump.me" {
		t.Fatalf("want njump.me (global fallback), got %s", got)
	}
}

func TestLoadDomainConfigs_ParsesJSON(t *testing.T) {
	orig := domainConfigs
	t.Cleanup(func() { domainConfigs = orig })

	f, err := os.CreateTemp("", "domain-cfg-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"nostr.eu":{"defaultLanguage":"de"}}`); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if err := loadDomainConfigs(f.Name()); err != nil {
		t.Fatal(err)
	}
	if got := defaultLangForDomain("nostr.eu"); got != "de" {
		t.Fatalf("want de, got %s", got)
	}
}
