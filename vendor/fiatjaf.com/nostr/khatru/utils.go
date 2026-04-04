package khatru

import (
	"context"

	"fiatjaf.com/nostr"
)

const (
	wsKey = iota
	subscriptionIdKey
	nip86HeaderAuthKey
	internalCallKey
)

func RequestAuth(ctx context.Context) {
	ws := GetConnection(ctx)
	ws.WriteJSON(nostr.AuthEnvelope{Challenge: &ws.Challenge})
}

func GetConnection(ctx context.Context) *WebSocket {
	wsi := ctx.Value(wsKey)
	if wsi != nil {
		return wsi.(*WebSocket)
	}
	return nil
}

// GetAuthed returns the last pubkey to have authenticated. Returns false if no one has.
//
// In a NIP-86 context it returns the single pubkey that have authenticated for that specific method call.
func GetAuthed(ctx context.Context) (nostr.PubKey, bool) {
	if conn := GetConnection(ctx); conn != nil {
		total := len(conn.AuthedPublicKeys)
		if total == 0 {
			return nostr.ZeroPK, false
		}
		return conn.AuthedPublicKeys[total-1], true
	}
	if nip86Auth := ctx.Value(nip86HeaderAuthKey); nip86Auth != nil {
		return nip86Auth.(nostr.PubKey), true
	}
	return nostr.ZeroPK, false
}

// GetAllAuthed returns all authenticated public keys.
//
// In a NIP-86 context it returns the single pubkey that authenticated for that method call.
func GetAllAuthed(ctx context.Context) []nostr.PubKey {
	if conn := GetConnection(ctx); conn != nil {
		return conn.AuthedPublicKeys
	}
	if nip86Auth := ctx.Value(nip86HeaderAuthKey); nip86Auth != nil {
		return []nostr.PubKey{nip86Auth.(nostr.PubKey)}
	}
	return []nostr.PubKey{}
}

// IsAuthed checks if the given public key is among the multiple that may have potentially authenticated.
func IsAuthed(ctx context.Context, pubkey nostr.PubKey) bool {
	if conn := GetConnection(ctx); conn != nil {
		for _, pk := range conn.AuthedPublicKeys {
			if pk == pubkey {
				return true
			}
		}
	}

	if nip86Auth := ctx.Value(nip86HeaderAuthKey); nip86Auth != nil {
		return nip86Auth.(nostr.PubKey) == pubkey
	}

	return false
}

// IsInternalCall returns true when a call to QueryEvents, for example, is being made because of a deletion
// or expiration request.
func IsInternalCall(ctx context.Context) bool {
	return ctx.Value(internalCallKey) != nil
}

func GetIP(ctx context.Context) string {
	conn := GetConnection(ctx)
	if conn == nil {
		return ""
	}

	return GetIPFromRequest(conn.Request)
}

func GetSubscriptionID(ctx context.Context) string {
	return ctx.Value(subscriptionIdKey).(string)
}

func SendNotice(ctx context.Context, msg string) {
	GetConnection(ctx).WriteJSON(nostr.NoticeEnvelope(msg))
}
