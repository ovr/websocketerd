package main

import (
	"errors"
	"strings"
)

type RPCSubscribeHandler struct {
	hub HubInterface
}

func (this RPCSubscribeHandler) MethodName() string {
	return "subscribe"
}

func (this RPCSubscribeHandler) Parameters() []RPCParameter {
	return []RPCParameter{
		{
			Name: "channel",
		},
	}
}

func (this RPCSubscribeHandler) Handle(request *RPCRequest, client *Client) (*JSONMap, error) {
	channel := request.Parameters[0]

	if strings.Contains(channel, "*") {
		return nil, errors.New("Pattern * is not allowed")
	}

	if !strings.Contains(channel, "user:pub:") {
		return nil, errors.New("You can subscribe only on user:pub:")
	}

	err := this.hub.Subscribe(channel, client)
	if err == nil {
		result := JSONMap{
			"success": true,
		}

		return &result, nil
	}

	return nil, errors.New("Cannot subscribe")
}
