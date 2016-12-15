package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
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

type Client struct {
	// The websocket connection.
	conn *websocket.Conn
}

func serveWs(server *Server, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{conn: conn}
	server.clients[client] = true

	log.Print("New connection");
}

type Server struct {
	clients map[*Client]bool

	httpServer *http.Server
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
