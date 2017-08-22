package main

import (
	"errors"
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
	this.hub.Subscribe(request.Parameters[0], client)

	return nil, errors.New("Unknown")
}
