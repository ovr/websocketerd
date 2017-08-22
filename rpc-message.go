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

func (this RPCMessageHandler) Parameters() []RPCParameter {
	return []RPCParameter{
		{
			Name: "channel",
		},
		{
			Name: "message",
		},
	}
}

func (this RPCMessageHandler) Handle(request *RPCRequest, client *Client) (*JSONMap, error) {
	this.hub.PublishMessage(request.Parameters[0], request.Parameters[1])

	return nil, errors.New("Unknown")
}
