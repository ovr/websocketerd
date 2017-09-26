package main

import (
	"encoding/json"
	"github.com/go-redis/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	clients *ClientsConcurrentMap

	httpServer *http.Server

	rpc *RPCServer

	hub HubInterface

	db *gorm.DB

	// Finish shutdown
	done chan bool

	// Shutdown server request
	shutdownChannel chan bool

	// Register requests from clients.
	registerChannel chan *Client

	// Unregister requests from clients.
	unregisterChannel chan *Client
}

func (this *Server) Run() {
	go this.hub.Listen()
	go this.Listen()

	go func() {
		err := this.httpServer.ListenAndServe()
		if err != nil {
			log.Panic("Cannot start HTTP Server", err)
		}
	}()
}

func (this *Server) Shutdown() {
	// Don't accept new connections to hub
	close(this.registerChannel)

	this.shutdownChannel <- true

	// w8th before shutdown request finish
	<-this.done
}

func (this *Server) Listen() {
	for {
		select {
		case client, ok := <-this.registerChannel:
			if ok {
				log.Debugln("[Event] Connection open")

				this.clients.Add(client)

				this.hub.Subscribe(client.GetDefaultPubChannel(), client)
			}
		case client := <-this.unregisterChannel:
			log.Debugln("[Event] Connection closed")

			this.hub.Unsubscribe(client)
			this.clients.Delete(client)

			close(client.sendChannel)
		case <-this.shutdownChannel:
			shutdownMsg, _ := json.Marshal(WebSocketNotification{
				Type: "SERVER_SHUTDOWN",
				Entity: map[string]interface{}{
					"delay": "30000",
				},
			})

			this.clients.Map(func(client *Client) {
				client.Send(shutdownMsg)
			})

			log.Debugln("Sending SERVER_SHUTDOWN to %d client(s)...\n", this.clients.Len())
			this.done <- true
		}
	}
}

func (this *Server) Stats() JSONMap {
	return JSONMap{
		"connections": this.clients.Len(),
		"pubsub": JSONMap{
			"channels": this.hub.GetChannelsCount(),
			"clients":  this.hub.GetClientsCount(),
		},
	}
}

func (this *Server) Clients() []JSONMap {
	clients := []JSONMap{}

	this.clients.Map(func(client *Client) {
		clientMap := JSONMap{
			"uid":      strconv.FormatUint(client.user.Id, 10),
			"agent":    client.agent,
			"channels": this.hub.GetChannelsForClient(client),
		}

		clients = append(clients, clientMap)
	})

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

func newServer(config *Configuration, newRelicApp newrelic.Application) *Server {
	db, err := gorm.Open(config.DB.Dialect, config.DB.Uri)
	if err != nil {
		panic(err)
	}

	db.LogMode(config.DB.ShowLog)
	db.DB().SetMaxIdleConns(config.DB.MaxIdleConnections)
	db.DB().SetMaxOpenConns(config.DB.MaxOpenConnections)

	httpServer := &http.Server{
		Addr:           ":8484",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	server := &Server{
		clients:    NewClientsConcurrentMap(),
		httpServer: httpServer,
		hub: NewRedisHub(
			redis.NewClient(
				&redis.Options{
					Addr:       config.Redis.Addr,
					PoolSize:   config.Redis.PoolSize,
					MaxRetries: config.Redis.MaxRetries,
				},
			),
		),
		rpc:               NewRPCServer(),
		db:                db,
		done:              make(chan bool),
		shutdownChannel:   make(chan bool),
		registerChannel:   make(chan *Client, 128),
		unregisterChannel: make(chan *Client, 128),
	}

	server.rpc.Add(RPCSubscribeHandler{
		hub: server.hub,
	})
	server.rpc.Add(RPCUnsubscribeHandler{
		hub: server.hub,
	})
	server.rpc.Add(RPCMessageHandler{
		hub: server.hub,
	})
	server.rpc.Add(RPCRoomJoinHandler{
		hub: server.hub,
	})
	server.rpc.Add(RPCRTCOfferHandler{
		hub: server.hub,
	})

	mux := http.NewServeMux()

	mux.HandleFunc(newrelic.WrapHandleFunc(newRelicApp, "/v1/ws/stats", func(w http.ResponseWriter, r *http.Request) {
		data, err := json.Marshal(server.Stats())
		if err != nil {
			log.Print(err)
		} else {
			w.Write(data)
		}
	}))

	mux.HandleFunc(newrelic.WrapHandleFunc(newRelicApp, "/v1/ws/clients", func(w http.ResponseWriter, r *http.Request) {
		data, err := json.Marshal(server.Clients())
		if err != nil {
			log.Print(err)
		} else {
			w.Write(data)
		}
	}))

	mux.HandleFunc(newrelic.WrapHandleFunc(newRelicApp, "/v1/ws/pubsub", func(w http.ResponseWriter, r *http.Request) {
		data, err := json.Marshal(server.PubSubChannels())
		if err != nil {
			log.Print(err)
		} else {
			w.Write(data)
		}
	}))

	mux.HandleFunc(newrelic.WrapHandleFunc(newRelicApp, "/v1/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(config, server, w, r)
	}))

	httpServer.Handler = mux

	return server
}
