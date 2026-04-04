package khatru

import (
	"context"
	"errors"
	"iter"

	"fiatjaf.com/lib/set"
	"fiatjaf.com/nostr"
)

var ErrSubscriptionClosedByClient = errors.New("subscription closed by client")

type listenerSpec struct {
	ssid   int    // internal numeric id for a listener
	sid    string // client-provided subscription id
	cancel context.CancelCauseFunc
}

type listener struct {
	id     string // duplicated here so we can easily send it on notifyListeners
	filter nostr.Filter
	ws     *WebSocket
}

type subscription struct {
	id     string
	filter nostr.Filter
	ws     *WebSocket
}

type dispatcher struct {
	serial        int
	subscriptions map[int]subscription
	byAuthor      map[nostr.PubKey]set.Set[int]
	byKind        map[nostr.Kind]set.Set[int]
	fallback      set.Set[int]
}

func newDispatcher() dispatcher {
	return dispatcher{
		subscriptions: make(map[int]subscription, 100),
		byAuthor:      make(map[nostr.PubKey]set.Set[int]),
		byKind:        make(map[nostr.Kind]set.Set[int]),
		fallback:      set.NewSliceSet[int](),
	}
}

func (d *dispatcher) addSubscription(sub subscription) int {
	d.serial++
	ssid := d.serial

	d.subscriptions[ssid] = sub

	indexed := false
	if sub.filter.Authors != nil {
		indexed = true
		for _, author := range sub.filter.Authors {
			s, ok := d.byAuthor[author]
			if !ok {
				s = set.NewSliceSet[int]()
				d.byAuthor[author] = s
			}
			s.Add(ssid)
		}
	}

	if sub.filter.Kinds != nil {
		indexed = true
		for _, kind := range sub.filter.Kinds {
			s, ok := d.byKind[kind]
			if !ok {
				s = set.NewSliceSet[int]()
				d.byKind[kind] = s
			}
			s.Add(ssid)
		}
	}

	if !indexed {
		d.fallback.Add(ssid)
	}

	return ssid
}

func (d *dispatcher) removeSubscription(ssid int) {
	sub, ok := d.subscriptions[ssid]
	if !ok {
		return
	}
	delete(d.subscriptions, ssid)

	indexed := false
	if sub.filter.Authors != nil {
		indexed = true
		for _, author := range sub.filter.Authors {
			s, ok := d.byAuthor[author]
			if !ok {
				return
			}
			s.Remove(ssid)
			if s.Len() == 0 {
				delete(d.byAuthor, author)
			}
		}
	}

	if sub.filter.Kinds != nil {
		indexed = true
		for _, kind := range sub.filter.Kinds {
			s, ok := d.byKind[kind]
			if !ok {
				return
			}
			s.Remove(ssid)
			if s.Len() == 0 {
				delete(d.byKind, kind)
			}
		}
	}

	if !indexed {
		d.fallback.Remove(ssid)
	}
}

func (d *dispatcher) candidates(event nostr.Event) iter.Seq[subscription] {
	return func(yield func(subscription) bool) {
		authorSubs, hasAuthorSubs := d.byAuthor[event.PubKey]
		kindSubs, hasKindSubs := d.byKind[event.Kind]

		if hasAuthorSubs && hasKindSubs {
			for _, ssid := range authorSubs.Slice() {
				sub, _ := d.subscriptions[ssid]

				if kindSubs.Has(ssid) {
					if filterMatchesTimestampConstraintsAndTags(sub.filter, event) {
						if !yield(sub) {
							return
						}
					}
				} else {
					// matched author but not tags, so this event doesn't qualify for any filter
					continue
				}
			}
		} else if hasAuthorSubs {
			for _, ssid := range authorSubs.Slice() {
				sub, _ := d.subscriptions[ssid]
				if sub.filter.Kinds != nil {
					// if there are any kinds in the filter we already know this doesn't qualify
					continue
				}

				if filterMatchesTimestampConstraintsAndTags(sub.filter, event) {
					if !yield(sub) {
						return
					}
				}
			}
		} else if hasKindSubs {
			for _, ssid := range kindSubs.Slice() {
				sub, _ := d.subscriptions[ssid]
				if sub.filter.Authors != nil {
					// if there are any authors in the filter we already know this doesn't qualify
					continue
				}

				if filterMatchesTimestampConstraintsAndTags(sub.filter, event) {
					if !yield(sub) {
						return
					}
				}
			}
		}

		for _, ssid := range d.fallback.Slice() {
			sub, _ := d.subscriptions[ssid]
			if filterMatchesTimestampConstraintsAndTags(sub.filter, event) {
				if !yield(sub) {
					return
				}
			}
		}
	}
}

//go:inline
func filterMatchesTimestampConstraintsAndTags(filter nostr.Filter, event nostr.Event) bool {
	if filter.Since != 0 && event.CreatedAt < filter.Since {
		return false
	}

	if filter.Until != 0 && event.CreatedAt > filter.Until {
		return false
	}

	for f, v := range filter.Tags {
		if !event.Tags.ContainsAny(f, v) {
			return false
		}
	}

	return true
}

//go:inline
func tagKeyValueKey(tagKey, tagValue string) string {
	return tagKey + "\x00" + tagValue
}

func (rl *Relay) GetListeningFilters() []nostr.Filter {
	respfilters := make([]nostr.Filter, 0, len(rl.dispatcher.subscriptions))
	for _, sub := range rl.dispatcher.subscriptions {
		respfilters = append(respfilters, sub.filter)
	}
	return respfilters
}

// addListener may be called multiple times for each id and ws -- in which case each filter will
// be added as an independent listener
func (rl *Relay) addListener(
	ws *WebSocket,
	id string,
	filter nostr.Filter,
	cancel context.CancelCauseFunc,
) {
	select {
	case <-rl.clientsMutex.C():
		defer rl.clientsMutex.Unlock()
	case <-ws.Context.Done():
		return
	}

	if specs, ok := rl.clients[ws]; ok /* this will always be true unless client has disconnected very rapidly */ {
		ssid := rl.dispatcher.addSubscription(subscription{
			ws:     ws,
			id:     id,
			filter: filter,
		})
		rl.clients[ws] = append(specs, listenerSpec{
			ssid:   ssid,
			cancel: cancel,
			sid:    id,
		})
	}
}

// remove a specific subscription id from listeners for a given ws client
// and cancel its specific context
func (rl *Relay) removeListenerId(ws *WebSocket, id string) {
	rl.clientsMutex.Lock()
	defer rl.clientsMutex.Unlock()

	if specs, ok := rl.clients[ws]; ok {
		kept := specs[:0]
		for _, spec := range specs {
			if spec.sid == id {
				spec.cancel(ErrSubscriptionClosedByClient)
				rl.dispatcher.removeSubscription(spec.ssid)
				continue
			}
			kept = append(kept, spec)
		}
		rl.clients[ws] = kept
	}
}

func (rl *Relay) removeClientAndListeners(ws *WebSocket) {
	rl.clientsMutex.Lock()
	defer rl.clientsMutex.Unlock()
	if specs, ok := rl.clients[ws]; ok {
		for _, spec := range specs {
			// no need to cancel contexts since they inherit from the main connection context
			rl.dispatcher.removeSubscription(spec.ssid)
		}
	}
	delete(rl.clients, ws)
}

// returns how many listeners were notified
func (rl *Relay) notifyListeners(event nostr.Event, skipPrevent bool) int {
	count := 0
listenersloop:
	for sub := range rl.dispatcher.candidates(event) {
		if !skipPrevent && nil != rl.PreventBroadcast {
			if rl.PreventBroadcast(sub.ws, sub.filter, event) {
				continue listenersloop
			}
		}
		sub.ws.WriteJSON(nostr.EventEnvelope{SubscriptionID: &sub.id, Event: event})
		count++
	}
	return count
}
