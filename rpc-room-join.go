package main

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
)

type RPCRoomJoinHandler struct {
	hub HubInterface
}

func (this RPCRoomJoinHandler) MethodName() string {
	return "room-join"
}

func (this RPCRoomJoinHandler) Parameters() []RPCParameter {
	return []RPCParameter{
		{
			Name: "id",
		},
		{
			Name: "from",
		},
	}
}

func (this RPCRoomJoinHandler) Handle(request *RPCRequest, client *Client) (*JSONMap, error) {
	roomId := request.Parameters[0].(string)
	from := request.Parameters[1].(string)

	if strings.Contains(roomId, "*") {
		return nil, errors.New("Pattern * is not allowed")
	}

	channel := fmt.Sprintf("room:%s", roomId)

	err := this.hub.Subscribe(channel, client)
	if err == nil {
		roomJoinMessage, marshalError := json.Marshal(SocketMessageWithPayload{
			Type: "ROOM_JOIN",
			Data: JSONMap{
				"from": from,
				"uid":  client.user.Id,
			},
		})

		if marshalError != nil {
			panic(marshalError)
		}

		log.Debug("ROOM_JOIN", channel)
		this.hub.PublishMessage(channel, string(roomJoinMessage))

		result := JSONMap{
			"success": true,
		}

		return &result, nil
	}

	return nil, errors.New("Cannot subscribe")
}
