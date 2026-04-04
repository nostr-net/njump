package nip42

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"fiatjaf.com/nostr"
)

// CreateUnsignedAuthEvent creates an event which should be sent via an "AUTH" command.
// If the authentication succeeds, the user will be authenticated as pubkey.
func CreateUnsignedAuthEvent(challenge string, pubkey nostr.PubKey, relayURL string) nostr.Event {
	return nostr.Event{
		PubKey:    pubkey,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindClientAuthentication,
		Tags: nostr.Tags{
			nostr.Tag{"relay", relayURL},
			nostr.Tag{"challenge", challenge},
		},
		Content: "",
	}
}

func GetRelayURLFromAuthEvent(event nostr.Event) string {
	for _, tag := range event.Tags {
		if len(tag) >= 2 && tag[0] == "relay" {
			return tag[1]
		}
	}
	return ""
}

// helper function for ValidateAuthEvent.
func parseURL(input string) (*url.URL, error) {
	return url.Parse(
		strings.ToLower(
			strings.TrimSuffix(input, "/"),
		),
	)
}

// ValidateAuthEvent checks whether event is a valid NIP-42 event for given challenge and relayURL.
// The result of the validation is encoded in the ok bool.
func ValidateAuthEvent(event nostr.Event, challenge string, relayURL string) (nostr.PubKey, error) {
	if event.Kind != nostr.KindClientAuthentication {
		return nostr.ZeroPK, fmt.Errorf("wanted kind %d, got %d", nostr.KindClientAuthentication, event.Kind)
	}

	if event.Tags.FindWithValue("challenge", challenge) == nil {
		if ctag := event.Tags.Find("challenge"); ctag == nil {
			return nostr.ZeroPK, fmt.Errorf("missing challenge")
		} else {
			return nostr.ZeroPK, fmt.Errorf("expected challenge '%s', got '%s'", challenge, ctag[1])
		}
	}

	expected, err := parseURL(relayURL)
	if err != nil {
		return nostr.ZeroPK, fmt.Errorf("server has misconfigured relay url")
	}

	if tag := event.Tags.Find("relay"); tag == nil {
		return nostr.ZeroPK, fmt.Errorf("missing 'relay' tag")
	} else {
		found, err := parseURL(tag[1])
		if err != nil {
			return nostr.ZeroPK, fmt.Errorf("invalid 'relay' tag '%s'", tag[1])
		}

		if expected.Scheme != found.Scheme ||
			expected.Host != found.Host ||
			expected.Path != found.Path {
			return nostr.ZeroPK, fmt.Errorf("expected relay URL '%s', got '%s'", expected, found)
		}
	}

	now := time.Now()
	if event.CreatedAt.Time().After(now.Add(10 * time.Minute)) {
		return nostr.ZeroPK, fmt.Errorf("auth event too much in the future")
	} else if event.CreatedAt.Time().Before(now.Add(-10 * time.Minute)) {
		return nostr.ZeroPK, fmt.Errorf("auth event too much in the past")
	}

	// save for last, as it is most expensive operation
	if !event.VerifySignature() {
		return nostr.ZeroPK, fmt.Errorf("invalid signature")
	}

	return event.PubKey, nil
}
