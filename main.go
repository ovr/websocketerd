package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/gorm"
	"log"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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

type AutoLoginToken struct {
	UserId      json.Number
	Token       string
	BrowserHash string
}

func RawUrlDecode(str string) string {
	re := regexp.MustCompile(`(?Ui)%[0-9A-F]{2}`)
	str = re.ReplaceAllStringFunc(str, func(s string) string {
		b, err := hex.DecodeString(s[1:])
		if err == nil {
			return string(b)
		}
		return s
	})
	return str
}

func parseAutoLoginToken(token string) (*AutoLoginToken, error) {
	var err error

	tokenValue := RawUrlDecode(token)

	parts := strings.Split(tokenValue, ",")
	if len(parts) != 3 {
		return nil, errors.New("Wrong login token")
	}

	_, err = strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return nil, err
	}

	loginToken := &AutoLoginToken{
		UserId:      json.Number(parts[0]),
		Token:       parts[1],
		BrowserHash: parts[2],
	}

	return loginToken, nil
}

func serveWs(config *Configuration, server *Server, w http.ResponseWriter, r *http.Request) {
	if err := recover(); err != nil {
		log.Printf("Recovery from panic\n%s", err)

		http.Error(w, "StatusInternalServerError", http.StatusInternalServerError)
	}

	var tokenPayload TokenPayload

	lt, err := r.Cookie("lt")
	if err == nil {
		autologinToken, err := parseAutoLoginToken(lt.Value)
		if err != nil {
			http.Error(w, "StatusUnauthorized", http.StatusUnauthorized)
			return
		}

		tokenPayload = TokenPayload{
			UserId: autologinToken.UserId,
		}

		row := LoginToken{}
		notFound := server.db.Where("token = UNHEX(?) and user_id = ?", autologinToken.Token, string(autologinToken.UserId)).First(&row).RecordNotFound()

		if notFound {
			http.Error(w, "StatusUnauthorized", http.StatusUnauthorized)
			return
		}
	} else {
		log.Print(err)

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

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			tokenPayload = TokenPayload{
				UserId:  claims["uid"].(json.Number),
				TokenId: claims["jti"].(json.Number),
			}
		} else {
			http.Error(w, "StatusForbidden", http.StatusForbidden)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print(err)

		http.Error(w, "StatusInternalServerError", http.StatusInternalServerError)
		return
	}

	var user *User = new(User)

	if server.db.First(user, tokenPayload.UserId.String()).RecordNotFound() {
		http.Error(w, "StatusForbidden", http.StatusForbidden)
		return
	}

	log.Print("[Event] New connection")


	client := NewClient(conn, tokenPayload, user, r.Header.Get("User-Agent"))
	server.registerChannel <- client

	go client.writePump(server)
	client.readPump(server)
}

type Server struct {
	clients map[*Client]bool

	httpServer *http.Server

	redis *redis.Client

	redisHub *RedisHub

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

			this.redisHub.Subscribe(client.GetDefaultPubChannel(), client)
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
			"agent": client.agent,
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
		registerChannel: make(chan *Client, 1024),
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
