package nip31

import "fiatjaf.com/nostr"

func GetAlt(event nostr.Event) string {
	for _, tag := range event.Tags {
		if len(tag) >= 2 && tag[0] == "alt" {
			return tag[1]
		}
	}
	return ""
}
