package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/khatru"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
)

type Settings struct {
	Port                     string `envconfig:"PORT" default:"2999"`
	Domain                   string `envconfig:"DOMAIN" default:"njump.me"`
	DefaultLanguage          string `envconfig:"DEFAULT_LANGUAGE" default:"en"`
	DomainConfigPath         string `envconfig:"DOMAIN_CONFIG_PATH"`
	ServiceURL               string `envconfig:"SERVICE_URL"`
	InternalDBPath           string `envconfig:"DISK_CACHE_PATH" default:"/tmp/njump-internal"`
	EventStorePath           string `envconfig:"EVENT_STORE_PATH" default:"/tmp/njump-db"`
	KVStorePath              string `envconfig:"KV_STORE_PATH" default:"/tmp/njump-kv"`
	HintsMemoryDumpPath      string `envconfig:"HINTS_SAVE_PATH" default:"/tmp/njump-hints.json"`
	TailwindDebug            bool   `envconfig:"TAILWIND_DEBUG"`
	RelayConfigPath          string `envconfig:"RELAY_CONFIG_PATH"`
	ClientsConfigPath        string `envconfig:"CLIENTS_CONFIG_PATH"`
	MediaAlertAPIKey         string `envconfig:"MEDIA_ALERT_API_KEY"`
	ErrorLogPath             string `envconfig:"ERROR_LOG_PATH" default:"/tmp/njump-errors.jsonl"`
	MetadataFetchConcurrency int    `envconfig:"METADATA_FETCH_CONCURRENCY" default:"8"`
	MetadataFetchTimeoutMs   int    `envconfig:"METADATA_FETCH_TIMEOUT_MS" default:"4000"`

	TrustedPubKeysHex []string `envconfig:"TRUSTED_PUBKEYS"`
	trustedPubKeys    []nostr.PubKey
}

//go:embed static/*
var static embed.FS

//go:embed clients.json
var embeddedClientsJSON string

var (
	s   Settings
	log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stdout}).
		With().Timestamp().Logger()
	tailwindDebugStuff template.HTML
)

func init() {
	// Set global log level to INFO to reduce noise
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func staticFileServer(fs http.FileSystem) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		ext := ""
		if idx := strings.LastIndex(path, "."); idx >= 0 {
			ext = strings.ToLower(path[idx:])
		}

		switch ext {
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".jpg", ".jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		case ".ico":
			w.Header().Set("Content-Type", "image/x-icon")
		case ".woff":
			w.Header().Set("Content-Type", "font/woff")
		case ".woff2":
			w.Header().Set("Content-Type", "font/woff2")
		case ".ttf":
			w.Header().Set("Content-Type", "font/ttf")
		case ".eot":
			w.Header().Set("Content-Type", "application/vnd.ms-fontobject")
		}

		http.FileServer(fs).ServeHTTP(w, r)
	})
}

func main() {
	err := envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig")
		return
	} else {
		s.trustedPubKeys = make([]nostr.PubKey, len(s.TrustedPubKeysHex))
		for i, pkhex := range s.TrustedPubKeysHex {
			s.trustedPubKeys[i] = nostr.MustPubKeyFromHex(pkhex)
		}
	}

	if s.DomainConfigPath != "" {
		if err := loadDomainConfigs(s.DomainConfigPath); err != nil {
			log.Fatal().Err(err).Msg("failed to load domain config")
			return
		}
	}

	if len(s.trustedPubKeys) == 0 {
		s.trustedPubKeys = defaultTrustedPubKeys
	}

	// eventstore and nostr system
	defer initSystem()()

	if s.RelayConfigPath != "" {
		configr, err := os.ReadFile(s.RelayConfigPath)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to load %q", s.RelayConfigPath)
			return
		}
		err = json.Unmarshal(configr, &relayConfig)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to load %q", s.RelayConfigPath)
			return
		}
		if len(relayConfig.Everything) > 0 {
			sys.FallbackRelays.URLs = relayConfig.Everything
		}
		if len(relayConfig.Profiles) > 0 {
			sys.MetadataRelays.URLs = relayConfig.Profiles
			sys.RelayListRelays.URLs = relayConfig.Profiles
			sys.FollowListRelays.URLs = relayConfig.Profiles
		}
		if len(relayConfig.JustIds) > 0 {
			sys.JustIDRelays.URLs = relayConfig.JustIds
		}
	}

	if s.ClientsConfigPath != "" {
		data, err := os.ReadFile(s.ClientsConfigPath)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to load clients config")
			return
		}
		if err := json.Unmarshal(data, &clientConfig); err != nil {
			log.Fatal().Err(err).Msg("failed to parse clients config")
			return
		}
	} else {
		if err := json.Unmarshal([]byte(embeddedClientsJSON), &clientConfig); err != nil {
			log.Fatal().Err(err).Msg("failed to parse embedded clients config")
			return
		}
	}

	// if we're in tailwind debug mode, initialize the runtime tailwind stuff
	if s.TailwindDebug {
		configb, err := os.ReadFile("tailwind.config.js")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to load tailwind.config.js")
			return
		}
		config := strings.Replace(
			strings.Replace(
				string(configb),
				"plugins: [require('@tailwindcss/typography')]", "", 1,
			),
			"module.exports", "tailwind.config", 1,
		)

		styleb, err := os.ReadFile("base.css")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to load base.css")
			return
		}
		style := string(styleb)

		tailwindDebugStuff = template.HTML(fmt.Sprintf("<script src=\"https://cdn.tailwindcss.com?plugins=typography\"></script><script>\n%s</script><style type=\"text/tailwindcss\">%s</style>", config, style))
	}

	// image rendering stuff
	initializeImageDrawingStuff()

	// initialize routines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go updateArchives(ctx)
	go deleteOldCachedEvents(ctx)
	go outboxHintsFileLoaderSaver(ctx)

	// expose our internal cache as a relay (mostly for debugging purposes)
	relay := khatru.NewRelay()
	relay.ServiceURL = "https://" + s.Domain
	relay.UseEventstore(sys.Store, DB_MAX_LIMIT)
	relay.OnEvent = func(ctx context.Context, event nostr.Event) (reject bool, msg string) {
		return true, "this relay is not writable"
	}

	// admin
	setupRelayManagement(relay)

	// routes
	mux := relay.Router()
	mux.Handle("/njump/static/", http.StripPrefix("/njump/", staticFileServer(http.FS(static))))

	sub := http.NewServeMux()
	sub.HandleFunc("/services/oembed", renderOEmbed)
	sub.HandleFunc("/njump/image/", renderImage)
	sub.HandleFunc("/image/", renderImage)
	sub.HandleFunc("/njump/proxy/", proxy)
	sub.HandleFunc("/proxy/", proxy)
	sub.HandleFunc("/robots.txt", renderRobots)
	sub.HandleFunc("/r/", renderRelayPage)
	sub.HandleFunc("/random", redirectToRandom)
	sub.HandleFunc("/e/", redirectFromESlash)
	sub.HandleFunc("/p/", redirectFromPSlash)
	sub.HandleFunc("/favicon.ico", redirectToFavicon)
	sub.HandleFunc("/embed/{code}", renderEmbedjs)
	sub.HandleFunc("/about", renderAbout)
	sub.HandleFunc("/{code}", renderEvent)
	sub.HandleFunc("/{$}", renderHomepage)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		loggingMiddleware(
			timeoutMiddleware(
				languageMiddleware(
					sub.ServeHTTP,
				),
			),
		)(w, r)
	})

	corsH := cors.Default()
	corsM := func(next http.HandlerFunc) http.HandlerFunc {
		return corsH.Handler(next).ServeHTTP
	}

	log.Print("listening at http://0.0.0.0:" + s.Port)
	server := &http.Server{Addr: "0.0.0.0:" + s.Port, Handler: corsM(relay.ServeHTTP)}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Error().Err(err).Msg("server error")
		}
	}()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	<-sc
	server.Close()
}
