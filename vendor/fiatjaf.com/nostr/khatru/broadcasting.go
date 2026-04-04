package khatru

import (
	"fiatjaf.com/nostr"
)

// BroadcastEvent emits an event to all listeners whose filters' match, skipping all filters and actions
// it also doesn't attempt to store the event or trigger any reactions or callbacks
func (rl *Relay) BroadcastEvent(evt nostr.Event) int {
	return rl.notifyListeners(evt, false)
}

// ForceBroadcastEvent is like BroadcastEvent, but it skips the PreventBroadcast hook.
func (rl *Relay) ForceBroadcastEvent(evt nostr.Event) int {
	return rl.notifyListeners(evt, true)
}
