package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type RPCRTCOfferHandler struct {
	hub HubInterface
}

func (this RPCRTCOfferHandler) MethodName() string {
	return "rtc-offer"
}

func (this RPCRTCOfferHandler) Parameters() []RPCParameter {
	return []RPCParameter{
		{
			Name: "id",
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

func (this RPCRTCOfferHandler) Handle(request *RPCRequest, client *Client) (*JSONMap, error) {
	roomId := request.Parameters[0].(string)
	from := request.Parameters[1].(string)
	to := request.Parameters[2].(string)
	payload := request.Parameters[3].(JSONMap)

	if strings.Contains(roomId, "*") {
		return nil, errors.New("Pattern * is not allowed")
	}

	channel := fmt.Sprintf("room:%s", roomId)

	err := this.hub.Subscribe(channel, client)
	if err == nil {
		socketMessage, marshalError := json.Marshal(SocketMessageWithPayload{
			Type: "RTC_OFFER",
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
