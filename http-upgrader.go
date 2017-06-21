package main

import (
	"net/http"
	log "github.com/sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
	"encoding/json"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// @todo check!
		return true
	},
}

func serveWs(config *Configuration, server *Server, w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Recovery from panic\n%s", err)

			http.Error(w, "StatusInternalServerError", http.StatusInternalServerError)
		}
	}()

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
		notFound := server.db.Where("token = UNHEX(?) and user_id = ?", autologinToken.Token, string(autologinToken.UserId)).Find(&row).RecordNotFound()

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

	if server.db.Find(user, tokenPayload.UserId.String()).RecordNotFound() {
		http.Error(w, "StatusForbidden", http.StatusForbidden)
		return
	}

	log.Print("[Event] New connection")

	client := NewClient(conn, tokenPayload, user, r.Header.Get("User-Agent"))
	server.registerChannel <- client

	// exit from HTTP goroutine, now http server can free unneeded things
	go client.writePump(server)
	go client.readPump(server)
}
