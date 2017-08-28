package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// @todo check!
		return true
	},
}

func authByLT(r *http.Request, db *gorm.DB) *json.Number {
	lt, err := r.Cookie("lt")
	if err == nil {
		autologinToken, err := parseAutoLoginToken(lt.Value)
		if err != nil {
			return nil
		}

		row := LoginToken{}
		notFound := db.Where("token = UNHEX(?) and user_id = ?", autologinToken.Token, string(autologinToken.UserId)).Find(&row).RecordNotFound()

		if notFound {
			return nil
		}

		return &autologinToken.UserId
	}

	return nil
}

func authByJWT(r *http.Request, jwtSecret string) (*json.Number, error) {
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		return nil, nil
	}

	parser := &jwt.Parser{
		UseJSONNumber: true,
	}

	token, err := parser.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		uid := claims["uid"].(json.Number)
		return &uid, nil
	}

	return nil, errors.New("Unknown claim uid")
}

func serveWs(config *Configuration, server *Server, w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Recovery from panic\n%s", err)

			http.Error(w, "StatusInternalServerError", http.StatusInternalServerError)
		}
	}()

	userId, err := authByJWT(r, config.JWTSecret)
	if err == nil {
		if userId == nil {
			// legacy compatibility auth
			userId = authByLT(r, server.db)
		}
	} else {
		log.Debug(err)
	}

	if userId == nil {
		http.Error(w, "StatusForbidden", http.StatusForbidden)
		return
	}

	var user *User = new(User)

	if server.db.Find(user, userId.String()).RecordNotFound() {
		http.Error(w, "StatusForbidden", http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warningln(err)

		// We don't needed to response, upgrader.returnError will do it automatically
		return
	}

	log.Debugln("[Event] New connection")

	client := NewClient(conn, user, r.Header.Get("User-Agent"))
	server.registerChannel <- client

	// exit from HTTP goroutine, now http server can free unneeded things
	go client.writePump(server)
	go client.readPump(server)
}
