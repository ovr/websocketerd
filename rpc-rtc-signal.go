package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type RPCRTCSignalHandler struct {
	hub HubInterface
}

func (this RPCRTCSignalHandler) MethodName() string {
	return "rtc-signal"
}

func (this RPCRTCSignalHandler) Parameters() []RPCParameter {
	return []RPCParameter{
		{
			Name: "signalType",
		},
		{
			Name: "roomId",
		},
		{
			Name: "from",
		},
		{
			Name: "to",
		},
		{
			Name: "payload",
		},
	}
}

func (this RPCRTCSignalHandler) Handle(request *RPCRequest, client *Client) (*JSONMap, error) {
	signalType := request.Parameters[0].(string)
	roomId := request.Parameters[1].(string)
	from := request.Parameters[2].(string)
	to := request.Parameters[3].(string)
	payload := request.Parameters[4].(JSONMap)

	if strings.Contains(roomId, "*") {
		return nil, errors.New("Pattern * is not allowed")
	}

	channel := fmt.Sprintf("room:%s", roomId)

	err := this.hub.Subscribe(channel, client)
	if err == nil {
		socketMessage, marshalError := json.Marshal(SocketMessageWithPayload{
			Type: signalType,
			Data: JSONMap{
				"from":    from,
				"to":      to,
				"payload": payload,
			},
		})

		if marshalError != nil {
			panic(marshalError)
		}

		this.hub.PublishMessage(channel, string(socketMessage))

		result := JSONMap{
			"success": true,
		}

		return &result, nil
	}

	return nil, errors.New("Cannot subscribe")
}
