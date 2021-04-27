// from https://github.com/gotify/server/blob/3454dcd60226acf121009975d947f05d41267283/api/stream/client.go
package push

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait = 2 * time.Second
)

var ping = func(conn *websocket.Conn) error {
	return conn.WriteMessage(websocket.PingMessage, nil)
}

var writeBytes = func(conn *websocket.Conn, data []byte) error {
	return conn.WriteMessage(websocket.TextMessage, data)
}

type client struct {
	conn    *websocket.Conn
	onClose func(*client)
	write   chan *[]byte
	token   string
	once    once
}

func newClient(conn *websocket.Conn, token string, onClose func(*client)) *client {
	return &client{
		conn:    conn,
		write:   make(chan *[]byte, 1),
		token:   token,
		onClose: onClose,
	}
}

// Close closes the connection.
func (c *client) Close() {
	c.once.Do(func() {
		_ = c.conn.Close()
		close(c.write)
	})
}

// NotifyClose closes the connection and notifies that the connection was closed.
func (c *client) NotifyClose() {
	c.once.Do(func() {
		_ = c.conn.Close()
		close(c.write)
		c.onClose(c)
	})
}

// startWriteHandler starts listening on the client connection. As we do not need anything from the client,
// we ignore incoming messages. Leaves the loop on errors.
func (c *client) startReading(pongWait time.Duration) {
	defer c.NotifyClose()
	c.conn.SetReadLimit(64)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(appData string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		if _, _, err := c.conn.NextReader(); err != nil {
			printWebSocketError("ReadError", err)
			return
		}
	}
}

// startWriteHandler starts the write loop. The method has the following tasks:
// * ping the client in the interval provided as parameter
// * write messages send by the channel to the client
// * on errors exit the loop.
func (c *client) startWriteHandler(pingPeriod time.Duration) {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		c.NotifyClose()
		pingTicker.Stop()
	}()

	for {
		select {
		case message, ok := <-c.write:
			if !ok {
				return
			}

			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := writeBytes(c.conn, *message); err != nil {
				printWebSocketError("WriteError", err)
				return
			}
		case <-pingTicker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ping(c.conn); err != nil {
				printWebSocketError("PingError", err)
				return
			}
		}
	}
}

func printWebSocketError(prefix string, err error) {
	closeError, ok := err.(*websocket.CloseError)

	if ok && closeError != nil && (closeError.Code == 1000 || closeError.Code == 1001) {
		// normal closure
		return
	}

	log.Printf("error WebSocket: %s %s", prefix, err)
}
