//go:generate npm install tailwindcss
//go:generate npx tailwind -i base.css -o static/tailwind-bundle.min.css --minify
//go:generate go run -mod=mod github.com/a-h/templ/cmd/templ@latest generate

package main

import (
	_ "embed"
	"html/template"

	"fiatjaf.com/nostr/sdk"
	"github.com/a-h/templ"
)

type TemplateID int

const (
	Note TemplateID = iota
	Profile
	LongForm
	TelegramInstantView
	FileMetadata
	LiveEvent
	LiveEventMessage
	CalendarEvent
	WikiEvent
	FollowSet
	StarterPack
	Highlight
	Other
)

type OpenGraphParams struct {
	SingleTitle string
	// x (we will always render just the singletitle if we have that)
	Superscript string
	Subscript   string

	BigImage string
	// x (we will always render just the bigimage if we have that)
	Video        string
	VideoType    string
	Image        string
	ProxiedImage string

	// this is the main text we should always have
	Text string
}

type DetailsParams struct {
	HideDetails     bool
	CreatedAt       string
	EventJSON       template.HTML
	Metadata        sdk.ProfileMetadata
	Nevent          string
	Nprofile        string
	SeenOn          []string
	Kind            int
	KindNIP         string
	KindDescription string
	Extra           templ.Component
}

type HeadParams struct {
	IsHome    bool
	IsAbout   bool
	IsProfile bool
	IsEvent   bool // true on event pages (where the top banner should appear)
	Lang      string
	NaddrNaked  string
	NeventNaked string
	Oembed      string
	Domain      string
}

// withIsEvent returns a copy of h with IsEvent set to true. Used by event page
// templates so the top banner only renders on event pages (not the homepage, etc.).
func withIsEvent(h HeadParams) HeadParams {
	h.IsEvent = true
	return h
}

type BaseEventPageParams struct {
	Event EnhancedEvent
	Style Style
	Alt   string
}

