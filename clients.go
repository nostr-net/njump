package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/a-h/templ"
)

type ClientReference struct {
	ID       string
	Name     string
	Base     string
	URL      templ.SafeURL
	Platform string
}

type ClientConfig struct {
	Clients map[string]ClientDefinition `json:"clients"`
}

type ClientDefinition struct {
	Name     string `json:"name"`
	Base     string `json:"base"`
	Platform string `json:"platform"`
}

type KindClients struct {
	Kind    int      `json:"kind"`
	Clients []string `json:"clients"`
}

type ClientMappings struct {
	KindMappings []KindClients `json:"kind_mappings"`
	Default      []string      `json:"default"`
}

const (
	platformWeb     = "web"
	platformIOS     = "ios"
	platformAndroid = "android"
)

var (
	native     = ClientReference{ID: "native", Name: "Your default app", Base: "nostr:{code}", Platform: "native"}
	defaultWeb = ClientReference{ID: "default-web", Name: "Your default web app", Base: "web+nostr:{code}", Platform: "web"}

	// Configuration loaded from file (if provided)
	customClientConfig *ClientConfig
	customClientMappings *ClientMappings
)

var (
	nosotros       = ClientReference{ID: "nosotros", Name: "Nosotros", Base: "https://dev.nosotros.app/{code}", Platform: platformWeb}
	nosotrosRelay  = ClientReference{ID: "nosotros", Name: "Nosotros", Base: "https://dev.nosotros.app/feed?kind=%5B1%2C6%5D&limit=100&type=relayfeed&relay=wss%3A%2F%2F{code}", Platform: platformWeb}
	nosta          = ClientReference{ID: "nosta", Name: "Nosta", Base: "https://nosta.me/{code}", Platform: platformWeb}
	phoenix        = ClientReference{ID: "phoenix", Name: "Phoenix", Base: "https://phoenix.social/{code}", Platform: platformWeb}
	olasWeb        = ClientReference{ID: "olas", Name: "Olas", Base: "https://olas.app/e/{code}", Platform: platformWeb}
	primalWeb      = ClientReference{ID: "primal", Name: "Primal", Base: "https://primal.net/e/{code}", Platform: platformWeb}
	nostrudel      = ClientReference{ID: "nostrudel", Name: "Nostrudel", Base: "https://nostrudel.ninja/l/{code}", Platform: platformWeb}
	nostter        = ClientReference{ID: "nostter", Name: "Nostter", Base: "https://nostter.app/{code}", Platform: platformWeb}
	nostterRelay   = ClientReference{ID: "nostter", Name: "Nostter", Base: "https://nostter.app/relays/wss%3A%2F%2F{code}", Platform: platformWeb}
	jumble         = ClientReference{ID: "jumble", Name: "Jumble", Base: "https://jumble.social/{code}", Platform: platformWeb}
	jumbleRelay    = ClientReference{ID: "jumble", Name: "Jumble", Base: "https://jumble.social/?r=wss://{code}", Platform: platformWeb}
	coracle        = ClientReference{ID: "coracle", Name: "Coracle", Base: "https://coracle.social/{code}", Platform: platformWeb}
	coracleRelay   = ClientReference{ID: "coracle", Name: "Coracle", Base: "https://coracle.social/relays/wss%3A%2F%2F{code}", Platform: platformWeb}
	relayTools     = ClientReference{ID: "relay.tools", Name: "relay.tools", Base: "https://relay.tools/posts/?relay=wss://{code}"}
	iris           = ClientReference{ID: "iris", Name: "Iris", Base: "https://iris.to/{code}", Platform: "web"}
	lumilumi       = ClientReference{ID: "lumilumi", Name: "Lumilumi", Base: "https://lumilumi.app/{code}", Platform: platformWeb}
	lumilumiRelay  = ClientReference{ID: "lumilumi", Name: "Lumilumi", Base: "https://lumilumi.app/relay/wss%3A%2F%2F{code}", Platform: platformWeb}
	chachiRelay    = ClientReference{ID: "chachi", Name: "chachi", Base: "https://chachi.chat/relay/wss%3A%2F%2F{code}/feed"}
	yakihonne      = ClientReference{ID: "yakihonne", Name: "YakiHonne", Base: "https://yakihonne.com/{code}", Platform: platformWeb}
	yakihonneRelay = ClientReference{ID: "yakihonne", Name: "YakiHonne", Base: "https://yakihonne.com/r/?r=wss://{code}", Platform: platformWeb}

	zapStream = ClientReference{ID: "zap.stream", Name: "zap.stream", Base: "https://zap.stream/{code}", Platform: platformWeb}
	shosho    = ClientReference{ID: "shosho", Name: "Shosho", Base: "https://shosho.live/live/{code}", Platform: platformWeb}

	habla  = ClientReference{ID: "habla", Name: "Habla", Base: "https://habla.news/a/{code}", Platform: platformWeb}
	pareto = ClientReference{ID: "pareto", Name: "Pareto", Base: "https://pareto.space/a/{code}", Platform: platformWeb}

	voyage           = ClientReference{ID: "voyage", Name: "Voyage", Base: "intent:{code}#Intent;scheme=nostr;package=com.dluvian.voyage;end`;", Platform: platformAndroid}
	olasAndroid      = ClientReference{ID: "olas", Name: "Olas", Base: "intent:{code}#Intent;scheme=nostr;package=com.pablof7z.snapstr;end`;", Platform: platformAndroid}
	primalAndroid    = ClientReference{ID: "primal", Name: "Primal", Base: "intent:{code}#Intent;scheme=nostr;package=net.primal.android;end`;", Platform: platformAndroid}
	yakihonneAndroid = ClientReference{ID: "yakihonne", Name: "Yakihonne", Base: "intent:{code}#Intent;scheme=nostr;package=com.yakihonne.yakihonne;end`;", Platform: platformAndroid}
	freeFromAndroid  = ClientReference{ID: "freefrom", Name: "FreeFrom", Base: "intent:{code}#Intent;scheme=nostr;package=com.freefrom;end`;", Platform: platformAndroid}
	yanaAndroid      = ClientReference{ID: "yana", Name: "Yana", Base: "intent:{code}#Intent;scheme=nostr;package=yana.nostr;end`;", Platform: platformAndroid}
	amethyst         = ClientReference{ID: "amethyst", Name: "Amethyst", Base: "intent:{code}#Intent;scheme=nostr;package=com.vitorpamplona.amethyst;end`;", Platform: platformAndroid}

	nos          = ClientReference{ID: "nos", Name: "Nos", Base: "nos:{code}", Platform: platformIOS}
	damus        = ClientReference{ID: "damus", Name: "Damus", Base: "damus:{code}", Platform: platformIOS}
	nostur       = ClientReference{ID: "nostur", Name: "Nostur", Base: "nostur:{code}", Platform: platformIOS}
	olasIOS      = ClientReference{ID: "olas", Name: "Olas", Base: "olas:{code}", Platform: platformIOS}
	primalIOS    = ClientReference{ID: "primal", Name: "Primal", Base: "primal:{code}", Platform: platformIOS}
	freeFromIOS  = ClientReference{ID: "freefrom", Name: "FreeFrom", Base: "freefrom:{code}", Platform: platformIOS}
	yakihonneIOS = ClientReference{ID: "yakihonne", Name: "Yakihonne", Base: "yakihhone:{code}", Platform: platformIOS}

	wikistr     = ClientReference{ID: "wikistr", Name: "Wikistr", Base: "https://Wikistr.com/{handle}*{authorPubkey}", Platform: "web"}
	wikifreedia = ClientReference{ID: "wikifreedia", Name: "Wikifreedia", Base: "https://wikifreedia.xyz/{handle}/{npub}", Platform: "web"}
)

func loadClientsConfig() {
	if s.ClientsConfigPath == "" {
		return
	}

	// Load client definitions
	clientsConfigPath := s.ClientsConfigPath
	if data, err := os.ReadFile(clientsConfigPath); err == nil {
		var config ClientConfig
		if err := json.Unmarshal(data, &config); err != nil {
			log.Warn().Err(err).Str("path", clientsConfigPath).Msg("failed to parse clients config")
		} else {
			customClientConfig = &config
			log.Info().Str("path", clientsConfigPath).Int("count", len(config.Clients)).Msg("loaded custom clients config")
		}
	} else {
		log.Warn().Err(err).Str("path", clientsConfigPath).Msg("failed to read clients config")
	}

	// Try to load mappings file (same path with .mappings suffix)
	mappingsConfigPath := strings.TrimSuffix(clientsConfigPath, ".json") + ".mappings.json"
	if data, err := os.ReadFile(mappingsConfigPath); err == nil {
		var mappings ClientMappings
		if err := json.Unmarshal(data, &mappings); err != nil {
			log.Warn().Err(err).Str("path", mappingsConfigPath).Msg("failed to parse clients mappings config")
		} else {
			customClientMappings = &mappings
			log.Info().Str("path", mappingsConfigPath).Int("mappings", len(mappings.KindMappings)).Msg("loaded custom clients mappings")
		}
	} else {
		log.Debug().Err(err).Str("path", mappingsConfigPath).Msg("no custom clients mappings file found")
	}
}

func getClientReference(id string) ClientReference {
	if customClientConfig != nil {
		if def, exists := customClientConfig.Clients[id]; exists {
			return ClientReference{
				ID:       id,
				Name:     def.Name,
				Base:     def.Base,
				Platform: def.Platform,
			}
		}
	}

	// Fallback to hardcoded clients for backward compatibility
	switch id {
	case "nosotros": return nosotros
	case "nosotrosRelay": return nosotrosRelay
	case "nosta": return nosta
	case "phoenix": return phoenix
	case "olasWeb": return olasWeb
	case "primalWeb": return primalWeb
	case "nostrudel": return nostrudel
	case "nostter": return nostter
	case "nostterRelay": return nostterRelay
	case "jumble": return jumble
	case "jumbleRelay": return jumbleRelay
	case "coracle": return coracle
	case "coracleRelay": return coracleRelay
	case "relayTools": return relayTools
	case "iris": return iris
	case "lumilumi": return lumilumi
	case "lumilumiRelay": return lumilumiRelay
	case "chachiRelay": return chachiRelay
	case "yakihonne": return yakihonne
	case "yakihonneRelay": return yakihonneRelay
	case "zapStream": return zapStream
	case "shosho": return shosho
	case "habla": return habla
	case "pareto": return pareto
	case "voyage": return voyage
	case "olasAndroid": return olasAndroid
	case "primalAndroid": return primalAndroid
	case "yakihonneAndroid": return yakihonneAndroid
	case "freeFromAndroid": return freeFromAndroid
	case "yanaAndroid": return yanaAndroid
	case "amethyst": return amethyst
	case "nos": return nos
	case "damus": return damus
	case "nostur": return nostur
	case "olasIOS": return olasIOS
	case "primalIOS": return primalIOS
	case "freeFromIOS": return freeFromIOS
	case "yakihonneIOS": return yakihonneIOS
	case "wikistr": return wikistr
	case "wikifreedia": return wikifreedia
	default: return native // fallback
	}
}

func generateClientList(
	kind int,
	code string,
	withModifiers ...func(ClientReference, string) string,
) []ClientReference {
	var clientIDs []string

	// If custom mappings are available, use them
	if customClientMappings != nil {
		for _, mapping := range customClientMappings.KindMappings {
			if mapping.Kind == kind {
				clientIDs = mapping.Clients
				break
			}
		}
		// If no specific mapping found, use default
		if len(clientIDs) == 0 && len(customClientMappings.Default) > 0 {
			clientIDs = customClientMappings.Default
		}
	}

	// Fallback to hardcoded client lists if no custom mappings
	if len(clientIDs) == 0 {
		switch kind {
		case -1: // relays
			clientIDs = []string{
				"native", "nostur", "yakihonneAndroid", "yakihonneIOS",
				"jumbleRelay", "yakihonneRelay", "chachiRelay", "nosotrosRelay", "lumilumiRelay", "coracleRelay", "relayTools", "nostterRelay",
				"default-web",
			}
		case 1, 6:
			clientIDs = []string{
				"native", "damus", "nostur", "freeFromIOS", "yakihonneIOS", "nos", "primalIOS",
				"voyage", "yakihonneAndroid", "primalAndroid", "freeFromAndroid", "yanaAndroid",
				"nosotros", "jumble", "coracle", "lumilumi", "nostter", "nostrudel", "phoenix", "primalWeb", "iris",
			}
		case 20:
			clientIDs = []string{
				"native", "olasAndroid", "olasIOS",
				"nosotros", "lumilumi", "jumble", "olasWeb", "coracle", "default-web",
			}
		case 0:
			clientIDs = []string{
				"native", "nos", "damus", "nostur", "primalIOS", "freeFromIOS", "yakihonneIOS",
				"voyage", "yakihonneAndroid", "yanaAndroid", "freeFromAndroid", "primalAndroid",
				"nosotros", "jumble", "nosta", "coracle", "phoenix", "nostter", "nostrudel", "primalWeb", "iris", "default-web",
			}
		case 30023, 30024:
			clientIDs = []string{
				"native", "damus", "nos", "nostur", "yakihonneIOS",
				"yakihonneAndroid", "amethyst",
				"yakihonne", "lumilumi", "coracle", "pareto", "habla", "default-web",
			}
		case 1063:
			clientIDs = []string{
				"native", "amethyst", "lumilumi", "phoenix", "coracle", "nostrudel", "default-web",
			}
		case 9802:
			clientIDs = []string{
				"coracle", "nostrudel", "lumilumi", "jumble", "default-web",
			}
		case 30311:
			clientIDs = []string{
				"native", "amethyst", "nostur",
				"zapStream", "shosho", "lumilumi", "nostrudel", "default-web",
			}
		case 30818:
			clientIDs = []string{
				"native", "wikistr", "wikifreedia", "default-web",
			}
		case 31922, 31923:
			clientIDs = []string{
				"native", "coracle", "default-web",
			}
		default:
			clientIDs = []string{
				"native", "yakihonneIOS", "nos", "damus", "nostur", "primalIOS", "freeFromIOS",
				"voyage", "amethyst", "yakihonneAndroid", "yanaAndroid", "freeFromAndroid", "voyage",
				"yakihonne", "coracle", "phoenix", "nostter", "nostrudel", "primalWeb", "iris", "default-web",
			}
		}
	}

	// Convert client IDs to ClientReference objects
	var clients []ClientReference
	for _, id := range clientIDs {
		clients = append(clients, getClientReference(id))
	}

	for i, c := range clients {
		clients[i].URL = templ.SafeURL(strings.Replace(c.Base, "{code}", code, -1))
		for _, modifier := range withModifiers {
			clients[i].URL = templ.SafeURL(modifier(c, string(clients[i].URL)))
		}
	}

	return clients
}
