package main

import (
	"log"
	"net/http"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	"time"
	"gopkg.in/redis.v5"
	"encoding/json"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// @todo check!
		return true
	},
}

type JSONMap map[string]interface{}

func serveWs(config *Configuration, server *Server, w http.ResponseWriter, r *http.Request) {
	tokenString := r.URL.Query().Get("token");
	if tokenString == "" {
		http.Error(w, "StatusUnauthorized", http.StatusUnauthorized)
		return
	}

	parser := &jwt.Parser{
		UseJSONNumber: true,
	}

	token, err := parser.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:

		//jwt.SigningMethodHS256.Verify()
		//if _, ok := token.Method.(*jwt.SigningMethodRS256); !ok {
		//	return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		//}

		return []byte(config.JWTSecret), nil
	});
	if err != nil {
		http.Error(w, "StatusForbidden", http.StatusForbidden)
		return
	}

	var tokenPayload TokenPayload

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		tokenPayload = TokenPayload{
			UserId: claims["uid"].(json.Number),
			TokenId: claims["jti"].(json.Number),
		}
	} else {
		http.Error(w, "StatusForbidden", http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		conn: conn,
		sendChannel: make(chan []byte, 256),
		tokenPayload: tokenPayload,
	}
	server.clients[client] = true

	log.Print("New connection");

	go func() {
		pubsub, subscribeError := server.redis.Subscribe("pubsub:user:" + client.tokenPayload.UserId.String());
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

func newServer(config *Configuration) *Server {
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
				Addr: config.Redis.Addr,
				PoolSize: config.Redis.PoolSize,
				MaxRetries: config.Redis.MaxRetries,
			},
		),
	};

	server.httpServer.Handler = http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		serveWs(config, server, w, r)
	})

	return server
}

func main() {
	configuration := &Configuration{};
	configuration.Init("./config.json");

	server := newServer(configuration);
	server.Run()
}
