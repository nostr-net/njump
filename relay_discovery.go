package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/sdk"
)

const relayWatchKind = nostr.Kind(30166)

var defaultRelayDiscoveryURLs = []string{
	"wss://relay.nostr.watch",
	"wss://relaypag.es",
	"wss://monitorlizard.nostr1.com",
}

func startRelayDiscoveryLoop(ctx context.Context) {
	intervalSeconds := s.RelayDiscoveryRefreshSec
	if intervalSeconds <= 0 {
		intervalSeconds = 900
	}

	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := refreshRelayPoolsFromNostrWatch(ctx); err != nil {
				recordRelayDiscoveryRun("refresh_error")
				log.Warn().Err(err).Msg("failed refreshing relay pools from discovery")
			} else {
				recordRelayDiscoveryRun("refresh_ok")
			}
		}
	}
}

func refreshRelayPoolsFromNostrWatch(ctx context.Context) error {
	discoveryRelays, source, err := discoverRelayURLs(ctx)
	if err != nil {
		setRelayDiscoveryCandidateCount("error", 0)
		return err
	}
	if len(discoveryRelays) == 0 {
		setRelayDiscoveryCandidateCount(source, 0)
		return fmt.Errorf("discovery source %q returned no relays", source)
	}
	setRelayDiscoveryCandidateCount(source, len(discoveryRelays))

	existing := normalizeRelayURLs(sys.FallbackRelays.URLs, 0)
	merged := mergeRelayPools(existing, discoveryRelays, s.RelayDiscoveryMaxRelays)
	if len(merged) == 0 {
		return fmt.Errorf("merged relay list was empty from source %q", source)
	}

	if slices.Equal(merged, existing) {
		log.Debug().Int("count", len(merged)).Str("source", source).Msg("relay pools already up-to-date")
		recordRelayDiscoveryRun("unchanged")
		return nil
	}

	oldCount := len(existing)
	sys.FallbackRelays = sdk.NewRelayStream(merged...)
	setRelayPoolSize("fallback", len(merged))
	recordRelayDiscoveryRun("success")
	log.Info().Int("old_count", oldCount).Int("new_count", len(merged)).Str("source", source).Msg("updated fallback relay pool from discovery")

	return nil
}

func discoverRelayURLs(ctx context.Context) ([]string, string, error) {
	urls := parseDiscoveryURLList(s.RelayDiscoveryURL)
	if len(urls) == 0 {
		urls = defaultRelayDiscoveryURLs
	}
	urls = filterRelayList(urls)

	if len(urls) == 0 {
		return nil, "", errors.New("no relay discovery sources configured")
	}

	wsRelays := make([]string, 0, len(urls))
	httpRelays := make([]string, 0, len(urls))
	for _, source := range urls {
		if isWebSocketURL(source) {
			wsRelays = append(wsRelays, source)
		} else {
			httpRelays = append(httpRelays, source)
		}
	}

	if len(wsRelays) == 0 && len(httpRelays) == 0 {
		return nil, "", errors.New("relay discovery URL list has no supported scheme")
	}

	if len(wsRelays) > 0 {
		log.Info().Str("sources", strings.Join(wsRelays, ",")).Msg("trying relay discovery via NIP-66 websockets")
		relays, err := discoverRelayURLsViaNip66(ctx, wsRelays)
		if err == nil {
			log.Info().Int("relays", len(relays)).Str("source", strings.Join(wsRelays, ",")).Msg("relay discovery returned relay candidates")
			return relays, strings.Join(wsRelays, ","), nil
		}
		log.Warn().Err(err).Str("sources", strings.Join(wsRelays, ",")).Msg("NIP-66 relay discovery failed")
		if len(httpRelays) == 0 {
			return nil, strings.Join(wsRelays, ","), err
		}
	}

	if len(httpRelays) > 1 {
		log.Warn().Str("primary", httpRelays[0]).Msg("multiple HTTP discovery URLs found; using first for discovery")
	}
	if len(httpRelays) > 0 {
		log.Info().Str("source", httpRelays[0]).Msg("trying relay discovery via HTTP endpoint")
	}
	relays, err := discoverRelayURLsViaHTTP(ctx, httpRelays[0])
	return relays, httpRelays[0], err
}

func discoverRelayURLsViaNip66(ctx context.Context, sourceRelays []string) ([]string, error) {
	if len(sourceRelays) == 0 {
		return nil, errors.New("no relay discovery sources for NIP-66 discovery")
	}

	timeout := time.Duration(s.RelayDiscoveryTimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	limit := s.RelayDiscoveryMaxRelays * 2
	if limit <= 0 {
		limit = 100
	}

	fetchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	events := sys.Pool.FetchMany(
		fetchCtx,
		sourceRelays,
		nostr.Filter{
			Kinds: []nostr.Kind{relayWatchKind},
			Limit: limit,
		},
		nostr.SubscriptionOptions{
			MaxWaitForEOSE: timeout,
		},
	)

	discovered := make([]string, 0, limit)
	seen := make(map[string]struct{})
	for evt := range events {
		if evt.Event.Kind != relayWatchKind || !evt.Event.VerifySignature() {
			continue
		}

		for _, relay := range relayURLsFromStatusEvent(evt.Event) {
			normalized := normalizeRelayURL(relay)
			if normalized == "" {
				continue
			}
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			discovered = append(discovered, normalized)
			if s.RelayDiscoveryMaxRelays > 0 && len(discovered) >= s.RelayDiscoveryMaxRelays {
				return discovered, nil
			}
		}
	}
	if len(discovered) == 0 {
		return nil, fmt.Errorf("no NIP-66 status events returned relays from sources %v", sourceRelays)
	}

	return discovered, nil
}

func discoverRelayURLsViaHTTP(ctx context.Context, discoveryURL string) ([]string, error) {
	if discoveryURL == "" {
		return nil, errors.New("missing HTTP discovery URL")
	}

	timeout := time.Duration(s.RelayDiscoveryTimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: timeout}
	response, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery request failed with status %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	relays, err := parseNostrWatchResponse(body)
	if err != nil {
		return nil, err
	}

	return normalizeRelayURLs(relays, s.RelayDiscoveryMaxRelays), nil
}

func parseNostrWatchResponse(body []byte) ([]string, error) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	raw := make([]any, 0)
	switch typed := payload.(type) {
	case []any:
		raw = typed
	case map[string]any:
		if relays, ok := typed["relays"].([]any); ok {
			raw = relays
			break
		}
		for _, item := range typed {
			raw = append(raw, item)
		}
	default:
		return nil, errors.New("unexpected payload from discovery endpoint")
	}

	relays := make([]string, 0, len(raw))
	for _, item := range raw {
		relay := parseNostrWatchItem(item)
		if relay == "" {
			continue
		}
		relays = append(relays, relay)
	}

	if len(relays) == 0 {
		return nil, errors.New("no relay candidates in discovery payload")
	}
	return relays, nil
}

func parseNostrWatchItem(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case map[string]any:
		for _, key := range []string{"url", "relay", "endpoint"} {
			if v, ok := typed[key]; ok {
				if s, ok := v.(string); ok {
					return s
				}
			}
		}
	}
	return ""
}

func relayURLsFromStatusEvent(evt nostr.Event) []string {
	type relayCapability struct {
		open  *bool
		read  *bool
		write *bool
	}

	seen := make(map[string]struct{})
	caps := make(map[string]relayCapability)
	hasRTag := false
	var relays []string

	for tag := range evt.Tags.FindAll("r") {
		if len(tag) < 2 {
			continue
		}
		relay := normalizeRelayURL(tag[1])
		if relay == "" {
			continue
		}
		seen[relay] = struct{}{}
		if _, ok := caps[relay]; !ok {
			caps[relay] = relayCapability{
				open:  nil,
				read:  nil,
				write: nil,
			}
		}
	}

	for tag := range evt.Tags.FindAll("R") {
		hasRTag = true
		if len(tag) < 3 {
			continue
		}
		relay := normalizeRelayURL(tag[1])
		if relay == "" {
			continue
		}
		c, _ := caps[relay]
		capability := strings.TrimSpace(strings.ToLower(tag[2]))
		val := false
		switch capability {
		case "open", "read", "write":
			val = true
		case "!open", "!read", "!write":
			val = false
		default:
			continue
		}
		switch strings.TrimPrefix(capability, "!") {
		case "open":
			c.open = &val
		case "read":
			c.read = &val
		case "write":
			c.write = &val
		}
		caps[relay] = c
	}

	ordered := make([]string, 0, len(seen))
	for relay := range seen {
		ordered = append(ordered, relay)
	}

	for _, relay := range ordered {
		c := caps[relay]
		if hasRTag {
			if c.open != nil && !*c.open {
				continue
			}
			if c.read != nil && !*c.read {
				continue
			}
		}
		relays = append(relays, relay)
	}
	return relays
}

func mergeRelayPools(base []string, discovered []string, max int) []string {
	normalizedBase := normalizeRelayURLs(base, 0)
	normalizedDiscovered := normalizeRelayURLs(discovered, 0)
	if max <= 0 {
		max = len(normalizedBase) + len(normalizedDiscovered)
	}

	merged := make([]string, 0, len(normalizedBase)+len(normalizedDiscovered))
	seen := make(map[string]struct{})
	for _, relay := range normalizedBase {
		if _, ok := seen[relay]; ok {
			continue
		}
		seen[relay] = struct{}{}
		merged = append(merged, relay)
		if len(merged) >= max {
			return merged
		}
	}

	for _, relay := range normalizedDiscovered {
		if _, ok := seen[relay]; ok {
			continue
		}
		seen[relay] = struct{}{}
		merged = append(merged, relay)
		if len(merged) >= max {
			break
		}
	}

	return merged
}

func normalizeRelayURLs(relays []string, max int) []string {
	if max < 0 {
		max = 0
	}
	normalized := make([]string, 0, len(relays))
	seen := make(map[string]struct{})
	for _, relay := range relays {
		n := normalizeRelayURL(relay)
		if n == "" {
			continue
		}
		if isRelayBlacklisted(n) {
			log.Warn().Str("relay", n).Msg("skipping blacklisted relay")
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		normalized = append(normalized, n)
		if max > 0 && len(normalized) >= max {
			break
		}
	}
	return normalized
}

func normalizeRelayURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	parsed, err := parsePossibleRelayURL(trimmed)
	if err != nil {
		return ""
	}
	parsed = strings.ToLower(parsed)
	if len(parsed) > 1 && strings.HasSuffix(parsed, "/") {
		return parsed[:len(parsed)-1]
	}
	return parsed
}

func parsePossibleRelayURL(raw string) (string, error) {
	address := strings.TrimSpace(raw)
	if !strings.Contains(address, "://") {
		address = "wss://" + address
	}
	parsed, err := url.Parse(address)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return "", fmt.Errorf("invalid scheme %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("missing host in %q", raw)
	}
	host := parsed.Hostname()
	if host == "" {
		host = parsed.Host
	}
	if host == "" {
		return "", fmt.Errorf("missing host in %q", raw)
	}
	port := parsed.Port()
	if port != "" {
		host = net.JoinHostPort(host, port)
	}
	if host != parsed.Hostname() {
		parsed.Host = host
	}

	return parsed.Scheme + "://" + parsed.Host + parsed.RequestURI(), nil
}

func parseDiscoveryURLList(raw string) []string {
	parsed := strings.Split(raw, ",")
	var urls []string
	for _, urlValue := range parsed {
		clean := strings.TrimSpace(urlValue)
		if clean == "" {
			continue
		}
		urls = append(urls, clean)
	}
	return urls
}

func isWebSocketURL(raw string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(trimmed, "wss://") || strings.HasPrefix(trimmed, "ws://")
}
