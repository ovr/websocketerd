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

	log.Print("[Event] New connection");
	client := NewClient(conn, tokenPayload);
	server.clients[client] = true


	server.redisHub.Subscribe("pubsub:user:" + tokenPayload.UserId.String(), client);

	go client.writePump(server)
	client.readPump();
}

type Server struct {
	clients map[*Client]bool

	httpServer *http.Server

	redis *redis.Client

	redisHub *RedisHub

	// Unregister requests from clients.
	unregisterChannel chan *Client
}

func (this *Server) Run()  {
	err := this.httpServer.ListenAndServe()
	if err != nil {
		log.Fatal("Cannot start HTTP Server", err);
		panic(err)
	}
}

func (this *Server) RunHub() {
	for {
		select {
		case client := <- this.unregisterChannel:
			log.Print("[Event] Connection closed");

			if _, ok := this.clients[client]; ok {
				log.Print("Client Removed");

				this.redisHub.Unsubscribe(client)
				delete(this.clients, client)
				close(client.sendChannel)
			}
		}
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
		redisHub: NewRedisHub(
			redis.NewClient(
				&redis.Options{
					Addr: config.Redis.Addr,
					PoolSize: config.Redis.PoolSize,
					MaxRetries: config.Redis.MaxRetries,
				},
			),
		),
		unregisterChannel: make(chan *Client, 1024),
	};

	server.httpServer.Handler = http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		serveWs(config, server, w, r)
	})

	go server.RunHub();

	return server
}

func main() {
	configuration := &Configuration{};
	configuration.Init("./config.json");

	server := newServer(configuration);
	server.Run()
}
