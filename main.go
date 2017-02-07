package main

import (
	"encoding/json"
	"flag"
	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/gorm"
	"gopkg.in/redis.v5"
	"log"
	"net/http"
	"runtime"
	"time"
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
	tokenString := r.URL.Query().Get("token")
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
	})
	if err != nil {
		http.Error(w, "StatusForbidden", http.StatusForbidden)
		return
	}

	var tokenPayload TokenPayload

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		tokenPayload = TokenPayload{
			UserId:  claims["uid"].(json.Number),
			TokenId: claims["jti"].(json.Number),
		}
	} else {
		http.Error(w, "StatusForbidden", http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print(err)

		http.Error(w, "StatusInternalServerError", http.StatusInternalServerError)
		return
	}

	var user *User = new(User)
	server.db.First(user, tokenPayload.UserId.String())

	log.Print("[Event] New connection")
	client := NewClient(conn, tokenPayload, user)
	server.clients[client] = true

	server.redisHub.Subscribe("pubsub:user:"+tokenPayload.UserId.String(), client)

	go client.writePump(server)
	client.readPump()
}

type Server struct {
	clients map[*Client]bool

	httpServer *http.Server

	redis *redis.Client

	redisHub *RedisHub

	db *gorm.DB

	// Unregister requests from clients.
	unregisterChannel chan *Client
}

func (this *Server) Run() {
	err := this.httpServer.ListenAndServe()
	if err != nil {
		log.Fatal("Cannot start HTTP Server", err)
		panic(err)
	}
}

func (this *Server) RunHub() {
	for {
		select {
		case client := <-this.unregisterChannel:
			log.Print("[Event] Connection closed")

			if _, ok := this.clients[client]; ok {
				log.Print("Client Removed")

				this.redisHub.Unsubscribe(client)
				delete(this.clients, client)
				close(client.sendChannel)
			}
		}
	}
}

func (this *Server) Stats() JSONMap {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return JSONMap{
		"connections": len(this.clients),
		"memory": JSONMap{
			"alloc":       mem.Alloc,
			"total-alloc": mem.TotalAlloc,
			"heap-alloc":  mem.HeapAlloc,
			"heap-sys":    mem.HeapSys,
		},
		"pubsub": JSONMap{
			"channels": len(this.redisHub.channelsToClients),
			"clients":  len(this.redisHub.clientsToChannels),
		},
	}
}

func (this *Server) Clients() []JSONMap {
	clients := []JSONMap{}

	for client := range this.clients {
		clientMap := JSONMap{
			"uid": client.tokenPayload.UserId,
			"jti": client.tokenPayload.TokenId,
		}

		if channels, ok := this.redisHub.clientsToChannels[client]; ok {
			clientMap["channels"] = channels
		}

		clients = append(
			clients,
			clientMap,
		)
	}

	return clients
}

func (this *Server) PubSubChannels() []JSONMap {
	channels := []JSONMap{}

	for channel := range this.redisHub.channelsToClients {
		channelMap := JSONMap{
			"channel": channel,
		}

		channels = append(
			channels,
			channelMap,
		)
	}

	return channels
}

func newServer(config *Configuration) *Server {
	db, err := gorm.Open(config.DB.Dialect, config.DB.Uri)
	if err != nil {
		panic(err)
	}

	server := &Server{
		clients: map[*Client]bool{},
		httpServer: &http.Server{
			Addr:           ":8484",
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		redis: redis.NewClient(
			&redis.Options{
				Addr:       config.Redis.Addr,
				PoolSize:   config.Redis.PoolSize,
				MaxRetries: config.Redis.MaxRetries,
			},
		),
		redisHub: NewRedisHub(
			redis.NewClient(
				&redis.Options{
					Addr:       config.Redis.Addr,
					PoolSize:   config.Redis.PoolSize,
					MaxRetries: config.Redis.MaxRetries,
				},
			),
		),
		db:                db,
		unregisterChannel: make(chan *Client, 1024),
	}

	server.httpServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/ws/stats":
			data, err := json.Marshal(server.Stats())
			if err != nil {
				log.Print(err)
			} else {
				w.Write(data)
			}
			break
		case "/v1/ws/clients":
			data, err := json.Marshal(server.Clients())
			if err != nil {
				log.Print(err)
			} else {
				w.Write(data)
			}
			break
		case "/v1/ws/pubsub":
			data, err := json.Marshal(server.PubSubChannels())
			if err != nil {
				log.Print(err)
			} else {
				w.Write(data)
			}
			break
		default:
			serveWs(config, server, w, r)
			break
		}
	})

	go server.RunHub()

	return server
}

func main() {
	var (
		configFile string
	)

	flag.StringVar(&configFile, "config", "./config.json", "Config filepath")
	flag.Parse()

	configuration := &Configuration{}
	configuration.Init(configFile)

	server := newServer(configuration)
	server.Run()
}
