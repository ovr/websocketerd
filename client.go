package main

import (
	"log"
	"github.com/gorilla/websocket"
	"time"
	"bytes"
	"encoding/json"
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
		conn: conn,
		sendChannel: make(chan []byte, 256),
		tokenPayload: tokenPayload,
		user: user,
	}

	return client;
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

func (this *Client) readPump() {
	defer func() {
		this.conn.Close()
	}()

	for {
		_, message, err := this.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		log.Print(string(message));
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
		case message, ok := <- this.sendChannel:
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
	UserId json.Number
	TokenId json.Number
}
