package hints

import (
	"fiatjaf.com/nostr"
)

type RelayScores struct {
	Relay  string
	Scores [4]nostr.Timestamp
	Sum    int64
}

type HintsDB interface {
	TopN(pubkey nostr.PubKey, n int) []string
	Save(pubkey nostr.PubKey, relay string, key HintKey, score nostr.Timestamp)
	PrintScores()
	GetDetailedScores(pubkey nostr.PubKey, n int) []RelayScores
}
