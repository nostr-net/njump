package main

import (
	"net/url"
	"strings"
)

var relayBlacklist = map[string]struct{}{}

func configureRelayBlacklist(raw string) {
	relayBlacklist = make(map[string]struct{})
	if raw == "" {
		return
	}

	for _, relay := range strings.Split(raw, ",") {
		relay = strings.TrimSpace(relay)
		if relay == "" {
			continue
		}

		normalized := normalizeRelayURL(relay)
		if normalized != "" {
			relayBlacklist[normalized] = struct{}{}
			if parsed, err := url.Parse(normalized); err == nil {
				host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
				if host != "" {
					relayBlacklist[host] = struct{}{}
				}
			}
			continue
		}

		// fallback for malformed/partial values: accept hostnames only
		if parsed, err := url.Parse("wss://" + relay); err == nil {
			host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
			if host != "" {
				relayBlacklist[host] = struct{}{}
			}
		}
	}
}

func isRelayBlacklisted(relay string) bool {
	normalized := normalizeRelayURL(relay)
	if normalized == "" {
		return false
	}

	if _, ok := relayBlacklist[normalized]; ok {
		return true
	}

	parsed, err := url.Parse(normalized)
	if err != nil {
		return false
	}
	if host := strings.ToLower(strings.TrimSpace(parsed.Hostname())); host != "" {
		if _, ok := relayBlacklist[host]; ok {
			return true
		}
	}

	return false
}

func filterRelayList(relays []string) []string {
	filtered := make([]string, 0, len(relays))
	for _, relay := range relays {
		if relay == "" {
			continue
		}
		if isRelayBlacklisted(relay) {
			log.Warn().Str("relay", relay).Msg("skipping blacklisted discovery relay")
			continue
		}
		filtered = append(filtered, relay)
	}
	return filtered
}
