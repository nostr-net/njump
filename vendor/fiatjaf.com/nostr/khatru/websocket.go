package khatru

import (
	"context"
	"net/http"
	"sync"

	"fiatjaf.com/nostr"
	"github.com/fasthttp/websocket"
	"github.com/puzpuzpuz/xsync/v3"
)

type WebSocket struct {
	conn  *websocket.Conn
	mutex sync.Mutex

	// original request
	Request *http.Request

	// this Context will be canceled whenever the connection is closed from the client side or server-side.
	Context context.Context
	cancel  context.CancelFunc

	// nip42
	Challenge        string
	AuthedPublicKeys []nostr.PubKey
	authLock         sync.Mutex

	// nip77
	negentropySessions *xsync.MapOf[string, *NegentropySession]
}

func (ws *WebSocket) WriteJSON(any any) error {
	ws.mutex.Lock()
	err := ws.conn.WriteJSON(any)
	ws.mutex.Unlock()
	return err
}

func (ws *WebSocket) WriteMessage(t int, b []byte) error {
	ws.mutex.Lock()
	err := ws.conn.WriteMessage(t, b)
	ws.mutex.Unlock()
	return err
}
