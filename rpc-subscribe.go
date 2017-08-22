package main

import (
	"errors"
	rpc "github.com/interpals/websocketerd/rpc"
)

type RPCSubscribeHandler struct {
	hub HubInterface
}

func (this RPCSubscribeHandler) MethodName() string {
	return "subscribe"
}

func (this RPCSubscribeHandler) Handle(request *rpc.RPCRequest) (*rpc.RPCResponse, error) {
	if len(request.Parameters) != 1 {

	}

	return nil, errors.New("Unknown")
}
