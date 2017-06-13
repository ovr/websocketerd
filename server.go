package main

import (
	"encoding/json"
	"github.com/go-redis/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"log"
	"net/http"
	"runtime"
	"time"
)

type Server struct {
	clients map[*Client]bool

	httpServer *http.Server

	redis *redis.Client

	hub HubInterface

	db *gorm.DB

	// Register requests from clients.
	registerChannel chan *Client

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
		case client := <-this.registerChannel:
			log.Print("[Event] Connection open")

			this.clients[client] = true

			this.hub.Subscribe(client.GetDefaultPubChannel(), client)
		case client := <-this.unregisterChannel:
			log.Print("[Event] Connection closed")

			if _, ok := this.clients[client]; ok {
				log.Print("Client Removed")

				this.hub.Unsubscribe(client)
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
			"channels": this.hub.GetChannelsCount(),
			"clients":  this.hub.GetClientsCount(),
		},
	}
}

func (this *Server) Clients() []JSONMap {
	clients := []JSONMap{}

	for client := range this.clients {
		clientMap := JSONMap{
			"uid":      client.tokenPayload.UserId.String(),
			"jti":      client.tokenPayload.TokenId,
			"agent":    client.agent,
			"channels": this.hub.GetChannelsForClient(client),
		}

		clients = append(clients, clientMap)
	}

	return clients
}

func (this *Server) PubSubChannels() []JSONMap {
	channels := []JSONMap{}

	for channel := range this.hub.GetChannels() {
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

	db.LogMode(config.DB.ShowLog)
	db.DB().SetMaxIdleConns(config.DB.MaxIdleConnections)
	db.DB().SetMaxOpenConns(config.DB.MaxOpenConnections)

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
		hub: NewRedisHub(
			redis.NewClient(
				&redis.Options{
					Addr:       config.Redis.Addr,
					PoolSize:   config.Redis.PoolSize,
					MaxRetries: config.Redis.MaxRetries,
				},
			),
			config.Hub,
		),
		db:                db,
		registerChannel:   make(chan *Client, 1024),
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
