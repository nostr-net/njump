package main

import (
	"encoding/json"
	"os"
)

type DomainConfig struct {
	DefaultLanguage string `json:"defaultLanguage"`
}

var domainConfigs map[string]DomainConfig

func loadDomainConfigs(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &domainConfigs)
}

func defaultLangForDomain(domain string) string {
	if cfg, ok := domainConfigs[domain]; ok && cfg.DefaultLanguage != "" {
		return cfg.DefaultLanguage
	}
	return s.DefaultLanguage
}
