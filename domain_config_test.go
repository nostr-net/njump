package main

import (
	"os"
	"testing"
)

func TestDefaultLangForDomain_KnownDomain(t *testing.T) {
	domainConfigs = map[string]DomainConfig{
		"nostr.ae": {DefaultLanguage: "ar"},
		"nostr.at": {DefaultLanguage: "en"},
	}
	if got := defaultLangForDomain("nostr.ae"); got != "ar" {
		t.Fatalf("want ar, got %s", got)
	}
}

func TestDefaultLangForDomain_UnknownDomain_FallsBackToGlobal(t *testing.T) {
	domainConfigs = map[string]DomainConfig{}
	s.DefaultLanguage = "en"
	if got := defaultLangForDomain("unknown.example"); got != "en" {
		t.Fatalf("want en (global fallback), got %s", got)
	}
}

func TestLoadDomainConfigs_ParsesJSON(t *testing.T) {
	f, _ := os.CreateTemp("", "domain-cfg-*.json")
	f.WriteString(`{"nostr.eu":{"defaultLanguage":"de"}}`)
	f.Close()
	defer os.Remove(f.Name())

	if err := loadDomainConfigs(f.Name()); err != nil {
		t.Fatal(err)
	}
	if got := defaultLangForDomain("nostr.eu"); got != "de" {
		t.Fatalf("want de, got %s", got)
	}
}
