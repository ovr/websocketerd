package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"strconv"
	"time"
)

type Client struct {
	// The websocket connection.
	conn *websocket.Conn

	sendChannel chan []byte

	user *User

	// HTTP Header "User-Agent"
	agent string
}

func NewClient(conn *websocket.Conn, user *User, agent string) *Client {
	client := &Client{
		conn:        conn,
		sendChannel: make(chan []byte, 256),
		user:        user,
		agent:       agent,
	}

	return client
}

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 30 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

func (this *Client) GetDefaultPubChannel() string {
	return "pubsub:user:" + strconv.FormatUint(this.user.Id, 10)
}

func (this *Client) readPump(server *Server) {
	defer func() {
		server.unregisterChannel <- this
		this.conn.Close()
	}()

	this.conn.SetReadLimit(maxMessageSize)
	this.conn.SetReadDeadline(time.Now().Add(pongWait))
	this.conn.SetPongHandler(
		func(string) error {
			this.conn.SetReadDeadline(time.Now().Add(pongWait))

			return nil
		},
	)

	for {
		var err error

		_, plainMessage, err := this.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Warnln("Error: %v", err)
			}

			break
		}

		plainMessage = bytes.TrimSpace(bytes.Replace(plainMessage, newline, space, -1))

		message := &WebSocketNotification{}

		err = json.Unmarshal(plainMessage, message)
		if err != nil {
			log.Warnln(err)

			continue
		}
	}
}

func (this *Client) Send(message []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in Send", r)
		}
	}()

	this.sendChannel <- message
}

func (this *Client) writePump(server *Server) {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		this.conn.Close()
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
