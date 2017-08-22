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

func (this RPCSubscribeHandler) Handle(request *RPCRequest, client *Client) (*JSONMap, error) {
	if len(request.Parameters) != 1 {
		return nil, errors.New("Required parameters: [channel]")
	}

	this.hub.Subscribe(request.Parameters[0], client)

	return nil, errors.New("Unknown")
}
