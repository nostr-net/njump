package main

import (
	"os"
	"testing"
)

func TestLoadBannersConfig(t *testing.T) {
	// Create a temporary config file
	configContent := `{
		"global": [
			{
				"id": "global-banner",
				"type": "text",
				"position": "top",
				"text": "Global banner text"
			},
			{
				"id": "global-link",
				"type": "link",
				"position": "right",
				"text": "Click here",
				"url": "https://example.com"
			}
		],
		"domains": {
			"nostr.ae": [
				{
					"id": "nostrae-banner",
					"type": "html",
					"position": "bottom",
					"html": "<strong>Arabic Nostr</strong>"
				}
			],
			"nostr.at": [
				{
					"id": "nostrat-banner",
					"type": "text",
					"position": "top",
					"text": "Austrian Nostr"
				}
			]
		}
	}`

	tmpFile, err := os.CreateTemp("", "banners-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Reset global config
	bannersConfig = BannersConfig{}

	// Load config
	if err := LoadBannersConfig(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test global banners
	globalBanners := GetBannersForDomain("unknown.domain", BannerPositionTop)
	if len(globalBanners) != 1 {
		t.Errorf("Expected 1 global top banner, got %d", len(globalBanners))
	}
	if globalBanners[0].ID != "global-banner" {
		t.Errorf("Expected global-banner, got %s", globalBanners[0].ID)
	}

	// Test domain-specific banners
	nostraeBanners := GetBannersForDomain("nostr.ae", BannerPositionBottom)
	if len(nostraeBanners) != 1 {
		t.Errorf("Expected 1 nostr.ae bottom banner, got %d", len(nostraeBanners))
	}

	// Test merged banners (global + domain)
	nostraeTopBanners := GetBannersForDomain("nostr.ae", BannerPositionTop)
	if len(nostraeTopBanners) != 1 {
		t.Errorf("Expected 1 merged top banner for nostr.ae, got %d", len(nostraeTopBanners))
	}

	// Test unknown domain returns no banners
	unknownBanners := GetBannersForDomain("unknown.domain", BannerPositionRight)
	if len(unknownBanners) != 1 { // Should have global right banner
		t.Errorf("Expected 1 right banner for unknown domain (from global), got %d", len(unknownBanners))
	}
}

func TestGetAllBannersForDomain(t *testing.T) {
	configContent := `{
		"global": [
			{
				"id": "g1",
				"type": "text",
				"position": "top",
				"text": "Global top"
			},
			{
				"id": "g2",
				"type": "text",
				"position": "left",
				"text": "Global left"
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "banners-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	bannersConfig = BannersConfig{}
	if err := LoadBannersConfig(tmpFile.Name()); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	allBanners := GetAllBannersForDomain("any.domain")

	if len(allBanners[BannerPositionTop]) != 1 {
		t.Errorf("Expected 1 top banner, got %d", len(allBanners[BannerPositionTop]))
	}

	if len(allBanners[BannerPositionLeft]) != 1 {
		t.Errorf("Expected 1 left banner, got %d", len(allBanners[BannerPositionLeft]))
	}

	// Should not have right or bottom
	if _, ok := allBanners[BannerPositionRight]; ok {
		t.Error("Should not have right banners")
	}
}

func TestEmptyConfig(t *testing.T) {
	bannersConfig = BannersConfig{}
	
	banners := GetBannersForDomain("any.domain", BannerPositionTop)
	if len(banners) != 0 {
		t.Errorf("Expected 0 banners for empty config, got %d", len(banners))
	}
}
