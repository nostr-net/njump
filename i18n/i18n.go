package i18n

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/language"
)

// languageKey is used to store the preferred language in context
// via context.WithValue.
type languageKey struct{}

var bundle *goi18n.Bundle

// isLanguageAllowed checks if a language is allowed based on ALLOWED_LANGUAGE env variable.
// If ALLOWED_LANGUAGE is empty or undefined, all languages are allowed.
func isLanguageAllowed(langTag string) bool {
	allowedLangs := os.Getenv("ALLOWED_LANGUAGE")
	if allowedLangs == "" {
		return true // if not set, allow all languages
	}

	// split by comma and check if langTag is in the list
	for _, allowed := range strings.Split(allowedLangs, ",") {
		if strings.TrimSpace(allowed) == langTag {
			return true
		}
	}
	return false
}

func init() {
	// Set global log level to INFO to reduce noise
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	bundle = goi18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	localesPath := os.Getenv("LOCALE_PATH")
	if localesPath == "" {
		localesPath = "locales"
	}
	// load translation files from locales directory (support mapping JSON and go-i18n V2 files)
	filepath.WalkDir(localesPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Warn().Err(err).Str("path", localesPath).Msg("i18n: error walking locale files")
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		// use filename (without extension) as language tag for filtering
		langTag := strings.ToLower(strings.TrimSuffix(filepath.Base(path), ext))

		// check if this language is allowed
		if !isLanguageAllowed(langTag) {
			log.Debug().Str("path", path).Str("langTag", langTag).Msg("i18n: skipping language file (not in ALLOWED_LANGUAGE)")
			return nil
		}

		// attempt to load flat JSON mapping (legacy v1 format)
		if ext == ".json" {
			data, err := os.ReadFile(path)
			if err != nil {
				log.Warn().Err(err).Str("path", path).Msg("i18n: failed to read mapping file")
				return nil
			}
			var mapping map[string]string
			if err := json.Unmarshal(data, &mapping); err == nil {
				for id, other := range mapping {
					bundle.AddMessages(language.Make(langTag), &goi18n.Message{ID: id, Other: other})
				}
				log.Debug().Str("path", path).Msg("i18n: loaded JSON mapping file")
				return nil
			}
		}
		// load go-i18n V2 files (TOML or JSON)
		if ext == ".toml" || ext == ".json" {
			if _, err := bundle.LoadMessageFile(path); err != nil {
				log.Warn().Err(err).Str("path", path).Msg("i18n: failed to load translation file")
			} else {
				log.Debug().Str("path", path).Msg("i18n: loaded translation file")
			}
		}
		return nil
	})
}

// WithLanguage returns a copy of ctx that stores the preferred language.
func WithLanguage(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, languageKey{}, lang)
}

// LanguageFromContext extracts the stored language from ctx.
// If no language was set it returns an empty string.
func LanguageFromContext(ctx context.Context) string {
	lang, _ := ctx.Value(languageKey{}).(string)
	return lang
}

// Translate returns the localized string for id using the language stored in ctx.
// If translation is missing, it falls back to English and finally to the id itself.
func Translate(ctx context.Context, id string, data map[string]any) string {
	lang, _ := ctx.Value(languageKey{}).(string)
	// always include English so that it is used as a fallback
	localizer := goi18n.NewLocalizer(bundle, lang, "en")
	msg, err := localizer.Localize(&goi18n.LocalizeConfig{MessageID: id, TemplateData: data})
	if err != nil {
		var nfe *goi18n.MessageNotFoundErr
		if errors.As(err, &nfe) {
			log.Debug().Str("id", id).Str("lang", lang).Msg("i18n: message not found, falling back to default language")
			return msg
		}
		log.Warn().Err(err).Str("id", id).Str("lang", lang).Msg("i18n: translation error")
		return id
	}
	return msg
}

// LanguageOption represents a language choice with code and display label
type LanguageOption struct {
	Code  string
	Label string
}

func supportedLanguages() []LanguageOption {
	return []LanguageOption{
		{"en", "English"},
		{"de", "Deutsch"},
		{"es", "Español"},
		{"fr", "Français"},
		{"it", "Italiano"},
		{"pt", "Português"},
		{"nl", "Nederlands"},
		{"hu", "Magyar"},
		{"cz", "Čeština"},
		{"ar", "العربية"},
		{"he", "עברית"},
		{"fa", "فارسی"},
		{"tr", "Türkçe"},
	}
}

// GetAvailableLanguagesForAllowed returns the languages visible for the provided allow-list.
// An empty allow-list means all supported languages are available.
func GetAvailableLanguagesForAllowed(allowed []string) []LanguageOption {
	allLanguages := supportedLanguages()
	if len(allowed) == 0 {
		return allLanguages
	}

	allowedMap := make(map[string]bool, len(allowed))
	for _, lang := range allowed {
		allowedMap[strings.TrimSpace(strings.ToLower(lang))] = true
	}

	availableLanguages := make([]LanguageOption, 0, len(allLanguages))
	for _, lang := range allLanguages {
		if allowedMap[lang.Code] {
			availableLanguages = append(availableLanguages, lang)
		}
	}

	return availableLanguages
}

// GetAvailableLanguages returns the list of available languages based on ALLOWED_LANGUAGE env variable.
// If ALLOWED_LANGUAGE is empty or undefined, returns all supported languages.
func GetAvailableLanguages() []LanguageOption {
	allowedLangs := os.Getenv("ALLOWED_LANGUAGE")
	if allowedLangs == "" {
		return supportedLanguages()
	}

	allowed := make([]string, 0, len(strings.Split(allowedLangs, ",")))
	for _, item := range strings.Split(allowedLangs, ",") {
		if lang := strings.TrimSpace(item); lang != "" {
			allowed = append(allowed, lang)
		}
	}

	return GetAvailableLanguagesForAllowed(allowed)
}
