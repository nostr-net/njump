# khatru, a relay framework [![docs badge](https://img.shields.io/badge/docs-reference-blue)](https://pkg.go.dev/fiatjaf.com/nostr/khatru#Relay)

[![Run Tests](https://github.com/fiatjaf/khatru/actions/workflows/test.yml/badge.svg)](https://github.com/fiatjaf/khatru/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/fiatjaf.com/nostr/khatru.svg)](https://pkg.go.dev/fiatjaf.com/nostr/khatru)
[![Go Report Card](https://goreportcard.com/badge/fiatjaf.com/nostr/khatru)](https://goreportcard.com/report/fiatjaf.com/nostr/khatru)

Khatru makes it easy to write very very custom relays:

  - custom event or filter acceptance policies
  - custom `AUTH` handlers
  - custom storage and pluggable databases
  - custom webpages and other HTTP handlers

Here's a sample:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/khatru"
)

func main() {
	// create the relay instance
	relay := khatru.NewRelay()

	// set up some basic properties (will be returned on the NIP-11 endpoint)
	relay.Info.Name = "my relay"
	relay.Info.PubKey = nostr.MustPubKeyFromHex("79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798")
	relay.Info.Description = "this is my custom relay"
	relay.Info.Icon = "https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Fliquipedia.net%2Fcommons%2Fimages%2F3%2F35%2FSCProbe.jpg&f=1&nofb=1&ipt=0cbbfef25bce41da63d910e86c3c343e6c3b9d63194ca9755351bb7c2efa3359&ipo=images"

	// you must bring your own storage scheme -- if you want to have any
	store := make(map[nostr.ID]*nostr.Event, 120)

	// set up the basic relay functions
	relay.StoreEvent = func(ctx context.Context, event *nostr.Event) error {
		store[event.ID] = event
		return nil
	}
	relay.QueryStored = func(ctx context.Context, filter nostr.Filter) iter.Seq[*nostr.Event] {
		return func(yield func(*nostr.Event) bool) {
			for _, evt := range store {
				if filter.Matches(evt) {
					if !yield(evt) {
						break
					}
				}
			}
		}
	}
	relay.DeleteEvent = func(ctx context.Context, id nostr.ID) error {
		delete(store, id)
		return nil
	}

	// there are many other configurable things you can set
	relay.RejectEvent = append(relay.RejectEvent,
		// built-in policies
		policies.ValidateKind,
		policies.RejectUnprefixedNostrReferences,

		// define your own policies
		func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
			if event.PubKey == nostr.MustPubKeyFromHex("fa984bd7dbb282f07e16e7ae87b26a2a7b9b90b7246a44771f0cf5ae58018f52") {
				return true, "we don't allow this person to write here"
			}
			return false, "" // anyone else can
		},
	)

	// you can request auth by rejecting an event or a request with the prefix "auth-required: "
	relay.RejectFilter = append(relay.RejectFilter,
		// built-in policies
		policies.NoComplexFilters,

		// define your own policies
		func(ctx context.Context, filter nostr.Filter) (reject bool, msg string) {
			if authed, is := khatru.GetAuthed(ctx); is {
				log.Printf("request from %s\n", authed)
				return false, ""
			}
			return true, "auth-required: only authenticated users can read from this relay"
			// (this will cause an AUTH message to be sent and then a CLOSED message such that clients can
			//  authenticate and then request again)
		},
	)
	// check the docs for more goodies!

	mux := relay.Router()
	// set up other http handlers
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html")
		fmt.Fprintf(w, `<b>welcome</b> to my relay!`)
	})

	// start the server
	fmt.Println("running on :3334")
	http.ListenAndServe(":3334", relay)
}
```

### But I don't want to write my own database!

Fear no more. Using the [`fiatjaf.com/nostr/eventstore`](../eventstore) module you get a bunch of compatible databases out of the box and you can just plug them into your relay. For example, [`lmdb`](../eventstore/lmdb):

```go
	db := lmdb.LMDBackend{Path: "/tmp/khatru-lmdb-tmp"}
	if err := db.Init(); err != nil {
		panic(err)
	}

	relay.UseEventstore(db, 500)
```

### But I don't want to write a bunch of custom policies!

Fear no more. We have a bunch of common policies written in the [`fiatjaf.com/nostr/khatru/policies`](policies) package and also a handpicked selection of base sane defaults, which you can apply with:

```go
	policies.ApplySaneDefaults(relay)
```

Contributions to this are very much welcomed.
