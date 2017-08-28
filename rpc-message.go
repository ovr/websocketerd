package main

import (
	"errors"
	"strings"
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
	channel := request.Parameters[0]

	if strings.Contains(channel, "*") {
		return nil, errors.New("Pattern * is not allowed")
	}

	if !strings.Contains(channel, "room:") {
		return nil, errors.New("You can message only inside room: channel")
	}

	this.hub.PublishMessage(channel, request.Parameters[1])

	return nil, errors.New("Unknown")
}
