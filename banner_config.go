package main

import (
	"encoding/json"
	"math/rand"
	"os"
)

// BannerPosition represents where a banner can be placed
type BannerPosition string

const (
	BannerPositionTop    BannerPosition = "top"    // Header line, next to nav links
	BannerPositionLeft   BannerPosition = "left"   // Far-left vertical sidebar (fixed)
	BannerPositionRight  BannerPosition = "right"  // Far-right vertical sidebar, right of client list (fixed)
	BannerPositionBottom BannerPosition = "bottom" // Above the footer
)

// BannerType defines the type of banner
type BannerType string

const (
	BannerTypeText  BannerType = "text"
	BannerTypeLink  BannerType = "link"
	BannerTypeHTML  BannerType = "html"
	BannerTypeImage BannerType = "image"
)

// Banner represents a single banner configuration
type Banner struct {
	ID       string       `json:"id"`
	Type     BannerType   `json:"type"`
	Position BannerPosition `json:"position"`
	Text     string       `json:"text,omitempty"`
	HTML     string       `json:"html,omitempty"`
	URL      string       `json:"url,omitempty"`
	Class    string       `json:"class,omitempty"`    // Custom CSS classes
	Style    string       `json:"style,omitempty"`    // Custom inline styles
	AltText  string       `json:"altText,omitempty"`  // For image/link banners
	Src      string       `json:"src,omitempty"`      // For image banners (image URL)
	NewTab   bool         `json:"newTab,omitempty"`   // Open in new tab
}

// BannersConfig represents the full banners configuration
type BannersConfig struct {
	Global  []Banner            `json:"global,omitempty"`  // Banners shown on all domains
	Domains map[string][]Banner `json:"domains,omitempty"` // Domain-specific banners
	Rotate  bool                `json:"rotate,omitempty"`  // When true, each position shows one random banner per page view
}

var bannersConfig BannersConfig

// LoadBannersConfig loads banner configuration from a JSON file
func LoadBannersConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &bannersConfig)
}

// GetBannersForDomain returns banners configured for a specific domain
// It merges global banners with domain-specific ones
func GetBannersForDomain(domain string, position BannerPosition) []Banner {
	var result []Banner

	// Add global banners for this position
	for _, banner := range bannersConfig.Global {
		if banner.Position == position {
			result = append(result, banner)
		}
	}

	// Add domain-specific banners for this position
	if domainBanners, ok := bannersConfig.Domains[domain]; ok {
		for _, banner := range domainBanners {
			if banner.Position == position {
				result = append(result, banner)
			}
		}
	}

	log.Info().Str("domain", domain).Str("position", string(position)).Int("count", len(result)).Msg("GetBannersForDomain")

	// Rotation: show one random banner per position per page view
	if bannersConfig.Rotate && len(result) > 1 {
		result = []Banner{result[rand.Intn(len(result))]}
	}

	return result
}

// GetAllBannersForDomain returns all banners for a domain (all positions)
func GetAllBannersForDomain(domain string) map[BannerPosition][]Banner {
	result := make(map[BannerPosition][]Banner)
	
	for _, pos := range []BannerPosition{
		BannerPositionTop,
		BannerPositionLeft,
		BannerPositionRight,
		BannerPositionBottom,
	} {
		banners := GetBannersForDomain(domain, pos)
		if len(banners) > 0 {
			result[pos] = banners
		}
	}
	
	return result
}

// GetBannerParams converts banners to template params for a domain and position
func GetBannerParams(domain string, position BannerPosition) BannerContainerParams {
	banners := GetBannersForDomain(domain, position)
	params := BannerContainerParams{
		Position: string(position),
		Banners:  make([]BannerItemParams, 0, len(banners)),
	}

	for _, b := range banners {
		params.Banners = append(params.Banners, BannerItemParams{
			ID:      b.ID,
			Type:    string(b.Type),
			Text:    b.Text,
			HTML:    b.HTML,
			URL:     b.URL,
			Src:     b.Src,
			Class:   b.Class,
			AltText: b.AltText,
			NewTab:  b.NewTab,
		})
	}

	return params
}
