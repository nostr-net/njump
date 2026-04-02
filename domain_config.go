package main

import (
	"encoding/json"
	"os"
	"strings"
)

type DomainConfig struct {
	DefaultLanguage  string   `json:"defaultLanguage"`
	AllowedLanguages []string `json:"allowedLanguages"`
}

var domainConfigs map[string]DomainConfig

func loadDomainConfigs(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg map[string]DomainConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	domainConfigs = cfg
	return nil
}

func defaultLangForDomain(domain string) string {
	if cfg, ok := domainConfigs[domain]; ok && cfg.DefaultLanguage != "" {
		return cfg.DefaultLanguage
	}
	return s.DefaultLanguage
}

func allowedLanguagesForDomain(domain string) []string {
	if cfg, ok := domainConfigs[domain]; ok && len(cfg.AllowedLanguages) > 0 {
		return cfg.AllowedLanguages
	}

	allowed := strings.TrimSpace(os.Getenv("ALLOWED_LANGUAGE"))
	if allowed == "" {
		return nil
	}

	parts := strings.Split(allowed, ",")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if lang := strings.TrimSpace(strings.ToLower(part)); lang != "" {
			filtered = append(filtered, lang)
		}
	}
	return filtered
}

func isLanguageAllowedForDomain(domain, lang string) bool {
	allowed := allowedLanguagesForDomain(domain)
	if len(allowed) == 0 {
		return true
	}

	for _, candidate := range allowed {
		if strings.EqualFold(candidate, lang) {
			return true
		}
	}
	return false
}
