package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/fiatjaf/njump/i18n"
	"net/http"
	"net/url"
	"strings"
)

type OEmbedResponse struct {
	// for xml encoding
	XMLName xml.Name `json:"-" xml:"oembed"`

	Type            string `json:"type" xml:"type"`
	Version         string `json:"version" xml:"version"`
	Title           string `json:"title,omitempty" xml:"title,omitempty"`
	AuthorName      string `json:"author_name,omitempty" xml:"author_name,omitempty"`
	AuthorURL       string `json:"author_url,omitempty" xml:"author_url,omitempty"`
	ProviderName    string `json:"provider_name,omitempty" xml:"provider_name,omitempty"`
	ProviderURL     string `json:"provider_url,omitempty" xml:"provider_url,omitempty"`
	CacheAge        int    `json:"cache_age,omitempty" xml:"cache_age,omitempty"`
	ThumbnailURL    string `json:"thumbnail_url,omitempty" xml:"thumbnail_url,omitempty"`
	ThumbnailWidth  int    `json:"thumbnail_width,omitempty" xml:"thumbnail_width,omitempty"`
	ThumbnailHeight int    `json:"thumbnail_height,omitempty" xml:"thumbnail_height,omitempty"`

	// photo, video, rich types
	Width  int `json:"width,omitempty" xml:"width,omitempty"`
	Height int `json:"height,omitempty" xml:"height,omitempty"`

	// photo types
	URL string `json:"url,omitempty" xml:"url,omitempty"`

	// video, rich types
	HTML string `json:"html,omitempty" xml:"html,omitempty"`
}

func renderOEmbed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	targetURL, err := url.Parse(r.URL.Query().Get("url"))
	if err != nil {
		http.Error(w, i18n.Translate(ctx, "error.invalid_url", map[string]any{"err": err.Error()}), 400)
		return
	}
	code := strings.Split(targetURL.Path, "/")[1]

	if !strings.HasPrefix(code, "nevent1") {
		http.Error(w, i18n.Translate(ctx, "error.oembed_nevent_only", map[string]any{"code": code}), 400)
		return
	}

	host := r.Header.Get("X-Forwarded-Host")

	data, err := grabData(ctx, code, false)
	if err != nil {
		w.Header().Set("Cache-Control", "max-age=180")
		log.Warn().Err(err).Str("code", code).Msg("event not found on oembed")
		http.Error(w, i18n.Translate(ctx, "error.fetch_event", map[string]any{"err": err.Error()}), http.StatusNotFound)
		return
	}

	res := OEmbedResponse{
		Version:      "1.0",
		ProviderName: "njump",
		ProviderURL:  "https://" + host,
		Title:        i18n.Translate(ctx, "oembed.title", map[string]any{"name": data.event.author.Name}),
		AuthorName:   data.event.authorLong(),
		AuthorURL:    fmt.Sprintf("https://%s/%s", host, data.event.Npub()),
	}

	switch {
	case data.video != "":
		res.Type = "video"
		res.HTML = fmt.Sprintf(`<video controls><source src="%s"></video>`, data.video)
	case data.image != "":
		res.Type = "image"
		res.URL = data.image
		res.HTML = fmt.Sprintf(`<img src="%s">`, data.image)
	default:
		res.Type = "rich"
		res.HTML = data.content
	}

	format := r.URL.Query().Get("format")
	if format == "xml" {
		w.Header().Add("Content-Type", "text/xml")
		xml.NewEncoder(w).Encode(res)
	} else {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}
