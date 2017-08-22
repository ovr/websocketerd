package main

import (
	"errors"
)

type RPCUnsubscribeHandler struct {
	hub HubInterface
}

func (this RPCUnsubscribeHandler) MethodName() string {
	return "unsubscribe"
}

func (this RPCUnsubscribeHandler) Parameters() []RPCParameter {
	return []RPCParameter{
		{
			Name: "channel",
		},
	}
}

func (this RPCUnsubscribeHandler) Handle(request *RPCRequest, client *Client) (*JSONMap, error) {
	return nil, errors.New("Unimplemented")
}
