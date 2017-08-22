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

func (this RPCUnsubscribeHandler) Handle(request *RPCRequest, client *Client) (*JSONMap, error) {
	if len(request.Parameters) != 1 {
		return nil, errors.New("Required parameters: [channel]")
	}

	return nil, errors.New("Unimplemented")
}
