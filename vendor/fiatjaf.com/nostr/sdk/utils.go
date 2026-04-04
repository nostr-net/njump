package sdk

import (
	"math"
	"strings"
	"testing"

	"fiatjaf.com/nostr"
	"github.com/tidwall/gjson"
)

// IsVirtualRelay returns true if the given normalized relay URL shouldn't be considered for outbox-model calculations.
func IsVirtualRelay(url string) bool {
	if len(url) < 6 {
		// this is just invalid
		return true
	}

	if strings.HasPrefix(url, "wss://feeds.nostr.band") ||
		strings.HasPrefix(url, "wss://filter.nostr.wine") ||
		strings.HasPrefix(url, "wss://cache") {
		return true
	}

	if !testing.Testing() &&
		strings.HasPrefix(url, "ws://localhost") ||
		strings.HasPrefix(url, "ws://127.0.0.1") {
		return true
	}

	return false
}

// PerQueryLimitInBatch tries to make an educated guess for the batch size given the total filter limit and
// the number of abstract queries we'll be conducting at the same time.
func PerQueryLimitInBatch(totalFilterLimit int, numberOfQueries int) int {
	if numberOfQueries == 1 || totalFilterLimit*numberOfQueries < 50 {
		return totalFilterLimit
	}

	return max(4,
		int(
			math.Ceil(
				float64(totalFilterLimit)/
					math.Pow(float64(numberOfQueries), 0.4),
			),
		),
	)
}

// GetMainContent returns the user-provided text of the event. This is often the "content", but sometimes,
// like on kind:9802 highlights' "comment" tag, it's on a tag.
// for many other events it is nowhere, as the event doesn't contain any user-provided free text.
// (incomplete)
func GetMainContent(event nostr.Event) string {
	switch event.Kind {
	case 9802:
		// for highlights, check if the comment is in the desired language
		// only check the quote language if there is no comment
		if tag := event.Tags.Find("comment"); tag != nil && len(tag[1]) > 0 {
			return tag[1]
		}
		return ""
	case 0:
		return gjson.Get(event.Content, "about").Str
	case 443, 27235, 22242, 1059, 13:
		return ""
	default:
		return event.Content
	}
}
