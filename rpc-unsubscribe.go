package main

import (
	"errors"
	rpc "github.com/interpals/websocketerd/rpc"
)

type RPCUnsubscribeHandler struct {
	hub HubInterface
}

func (this RPCUnsubscribeHandler) MethodName() string {
	return "unsubscribe"
}

func (this RPCUnsubscribeHandler) Handle(request *rpc.RPCRequest) (*rpc.RPCResponse, error) {
	if len(request.Parameters) != 1 {

	}

	return nil, errors.New("Unknown")
}
