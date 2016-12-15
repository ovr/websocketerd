package main

import (
	"log"
	"net/http"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	"time"
	"gopkg.in/redis.v5"
	"bytes"
	"strconv"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// @todo check!
		return true
	},
}

type Client struct {
	// The websocket connection.
	conn *websocket.Conn

	sendChannel chan []byte

	tokenPayload TokenPayload
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

func (this *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		this.conn.Close()
	}()

	for {
		select {
		case message, ok := <- this.sendChannel:
			//this.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				this.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := this.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				return
			}
		case <-ticker.C:
			//this.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if err := this.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

type TokenPayload struct {
	UserId int64
	TokenId int64
}

type JSONMap map[string]interface{}

func serveWs(server *Server, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		conn: conn,
		sendChannel: make(chan []byte, 256),
	}
	server.clients[client] = true

	tokenString := r.URL.Query().Get("token");
	if tokenString != "" {
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:

			//jwt.SigningMethodHS256.Verify()
			//if _, ok := token.Method.(*jwt.SigningMethodRS256); !ok {
			//	return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			//}

			return []byte(""), nil
		})


		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			client.tokenPayload.UserId, _ = strconv.ParseInt(claims["uid"].(string), 10, 64);
		} else {
			log.Print(err)
			return
		}
	} else {
		log.Print(err)
		return
	}

	log.Print("New connection");

	go func() {
		pubsub, subscribeError := server.redis.Subscribe("pubsub:user:" + strconv.FormatInt(client.tokenPayload.UserId, 10));
		if subscribeError != nil {
			panic(subscribeError);
		}

		for {
			message, err := pubsub.ReceiveMessage();
			if err != nil {
				log.Print(err)
			}

			if message != nil {
				client.sendChannel <- []byte(message.Payload)
				log.Print(message);
			}
		}
	}();

	go client.writePump()
	client.readPump();
}

type Server struct {
	clients map[*Client]bool

	httpServer *http.Server

	redis *redis.Client
}

func (this *Server) Run()  {
	err := this.httpServer.ListenAndServe()
	if err != nil {
		log.Fatal("Cannot start HTTP Server", err);
		panic(err)
	}
}

func newServer() *Server {
	server := &Server{
		clients: map[*Client]bool{},
		httpServer: &http.Server{
			Addr: ":8484",
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		redis: redis.NewClient(
			&redis.Options{
				Addr:     "127.0.0.1:6379",
			},
		),
	};

	server.httpServer.Handler = http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		serveWs(server, w, r)
	})

	return server
}

func main() {
	server := newServer();
	server.Run()
}
