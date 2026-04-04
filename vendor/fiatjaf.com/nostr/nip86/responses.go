package nip86

import "fiatjaf.com/nostr"

type IDReason struct {
	ID     nostr.ID `json:"id"`
	Reason string   `json:"reason"`
}

type PubKeyReason struct {
	PubKey nostr.PubKey `json:"pubkey"`
	Reason string       `json:"reason"`
}

type IPReason struct {
	IP     string `json:"ip"`
	Reason string `json:"reason"`
}
