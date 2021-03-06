package main

import (
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
	maxMessageSize = 8 * 1024 // 8KB
)

func (this *Client) GetDefaultPubChannel() string {
	return "pubsub:user:" + strconv.FormatUint(this.user.Id, 10)
}

func (this *Client) WriteRPCResponseError(request *RPCRequest, result JSONMap) {
	this.WriteRPC(
		RPCResponseError{
			Id:    request.Id,
			Error: result,
		},
	)
}

func (this *Client) WriteRPCResponse(request *RPCRequest, result JSONMap) {
	this.WriteRPC(
		RPCResponse{
			Id:     request.Id,
			Result: result,
		},
	)
}

func (this *Client) WriteRPC(response interface{}) {
	r, err := json.Marshal(response)

	if err == nil {
		this.Send(r)
	} else {
		log.Warningln(err)
	}
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
		request := &RPCRequest{}

		_, r, err := this.conn.NextReader()
		if err != nil {
			log.Debugln(err)

			break
		}

		err = json.NewDecoder(r).Decode(request)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Warnln("Error: %v", err)

				// exit from connection
				break
			}

			this.WriteRPC(
				RPCFatalError{
					Error: JSONMap{
						"message": "Cannot decode RPC request",
					},
				},
			)

			continue
		}

		if request.Id == "" {
			this.WriteRPC(
				RPCFatalError{
					Error: JSONMap{
						"message": "Id is a required field",
					},
				},
			)

			continue
		}

		go server.rpc.Handle(request, this)
	}
}

func (this *Client) Send(message []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in Send:", r)
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
