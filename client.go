package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

type Client struct {
	// The websocket connection.
	conn *websocket.Conn

	sendChannel chan []byte

	tokenPayload TokenPayload

	user *User
}

func NewClient(conn *websocket.Conn, tokenPayload TokenPayload, user *User) *Client {
	client := &Client{
		conn:         conn,
		sendChannel:  make(chan []byte, 256),
		tokenPayload: tokenPayload,
		user:         user,
	}

	return client
}

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 1 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 2 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type WebSocketNotification struct {
	Type   string
	Entity interface{}
}

func (this *Client) GetDefaultPubChannel() string {
	return "pubsub:user:" + this.tokenPayload.UserId.String()
}

func (this *Client) readPump() {
	defer func() {
		this.conn.Close()
	}()

	for {
		var err error

		_, plainMessage, err := this.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}

		plainMessage = bytes.TrimSpace(bytes.Replace(plainMessage, newline, space, -1))

		message := &WebSocketNotification{}

		err = json.Unmarshal(plainMessage, message)
		if err != nil {
			log.Print(err)

			continue
		}
	}
}

func (this *Client) writePump(server *Server) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		this.conn.Close()

		server.unregisterChannel <- this
	}()

	for {
		select {
		case message, ok := <-this.sendChannel:
			this.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				this.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := this.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				return
			}
		case <-ticker.C:
			this.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if err := this.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

type TokenPayload struct {
	UserId  json.Number
	TokenId json.Number
}
