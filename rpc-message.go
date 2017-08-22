package main

import (
	"errors"
)

type RPCMessageHandler struct {
	hub HubInterface
}

func (this RPCMessageHandler) MethodName() string {
	return "message"
}

func (this RPCMessageHandler) Handle(request *RPCRequest, client *Client) (*JSONMap, error) {
	if len(request.Parameters) != 2 {
		return nil, errors.New("Required parameters: [channel, message]")
	}

	this.hub.PublishMessage(request.Parameters[0], request.Parameters[1])

	return nil, errors.New("Unknown")
}
