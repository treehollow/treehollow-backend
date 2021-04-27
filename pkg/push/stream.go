// from https://github.com/gotify/server/blob/3454dcd60226acf121009975d947f05d41267283/api/stream/stream.go
package push

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"net/http"
	"sync"
	"time"
)

// The API provides a handler for a WebSocket stream API.
type API struct {
	clients     map[string]*client
	lock        sync.RWMutex
	pingPeriod  time.Duration
	pongTimeout time.Duration
	upgrader    *websocket.Upgrader
}

var Api *API

// New creates a new instance of API.
// pingPeriod: is the interval, in which is server sends the a ping to the client.
// pongTimeout: is the duration after the connection will be terminated, when the client does not respond with the
// pong command.
func New(pingPeriod, pongTimeout time.Duration) *API {
	return &API{
		clients:     make(map[string]*client),
		pingPeriod:  pingPeriod,
		pongTimeout: pingPeriod + pongTimeout,
		upgrader:    newUpgrader(),
	}
}

// NotifyDeletedUser closes existing connections for the given user.
func (a *API) NotifyDeletedUser(token string) error {
	a.lock.Lock()
	defer a.lock.Unlock()
	if c, ok := a.clients[token]; ok {
		c.Close()
		delete(a.clients, token)
	}
	return nil
}

// Notify notifies the clients with the given userID that a new messages was created.
func (a *API) Notify(token string, msg *[]byte) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	if c, ok := a.clients[token]; ok {
		if viper.GetBool("is_debug") {
			fmt.Printf("WebSocket Notify: %s", *msg)
		}
		c.write <- msg
	} else {
		if viper.GetBool("is_debug") {
			fmt.Printf("WebSocket Error: token not connected: %s", *msg)
		}
	}
}

func (a *API) remove(remove *client) {
	a.lock.Lock()
	defer a.lock.Unlock()
	if c, ok := a.clients[remove.token]; ok {
		c.Close()
		delete(a.clients, remove.token)
	}
}

func (a *API) register(client *client) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.clients[client.token] = client
}

// Handle handles incoming requests. First it upgrades the protocol to the WebSocket protocol and then starts listening
// for read and writes.
// swagger:operation GET /stream message streamMessages
//
// Websocket, return newly created messages.
//
// ---
// schema: ws, wss
// produces: [application/json]
// security: [clientTokenHeader: [], clientTokenQuery: [], basicAuth: []]
// responses:
//   200:
//     description: Ok
//     schema:
//         $ref: "#/definitions/Message"
//   400:
//     description: Bad Request
//     schema:
//         $ref: "#/definitions/Error"
//   401:
//     description: Unauthorized
//     schema:
//         $ref: "#/definitions/Error"
//   403:
//     description: Forbidden
//     schema:
//         $ref: "#/definitions/Error"
//   500:
//     description: Server Error
//     schema:
//         $ref: "#/definitions/Error"
func (a *API) Handle(ctx *gin.Context) {
	token := ctx.GetHeader("TOKEN")
	conn, err := a.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	client := newClient(conn, token, a.remove)
	a.remove(client)
	a.register(client)
	go client.startReading(a.pongTimeout)
	go client.startWriteHandler(a.pingPeriod)
}

// Close closes all client connections and stops answering new connections.
func (a *API) Close() {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, c := range a.clients {
		c.Close()
	}
	for k := range a.clients {
		delete(a.clients, k)
	}
}

//func isAllowedOrigin(r *http.Request, allowedOrigins []*regexp.Regexp) bool {
//	origin := r.Header.Get("origin")
//	if origin == "" {
//		return true
//	}
//
//	u, err := url.Parse(origin)
//	if err != nil {
//		return false
//	}
//
//	if strings.EqualFold(u.Host, r.Host) {
//		return true
//	}
//
//	for _, allowedOrigin := range allowedOrigins {
//		if allowedOrigin.Match([]byte(strings.ToLower(u.Hostname()))) {
//			return true
//		}
//	}
//
//	return false
//}

func newUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			//if mode.IsDev() {
			return true
			//}
			//return isAllowedOrigin(r, compiledAllowedOrigins)
		},
	}
}

//func compileAllowedWebSocketOrigins(allowedOrigins []string) []*regexp.Regexp {
//	var compiledAllowedOrigins []*regexp.Regexp
//	for _, origin := range allowedOrigins {
//		compiledAllowedOrigins = append(compiledAllowedOrigins, regexp.MustCompile(origin))
//	}
//
//	return compiledAllowedOrigins
//}
