package main

import (
	log "github.com/sirupsen/logrus"
)

type Handler interface {
	MethodName() string

	Handle(request *RPCRequest, client *Client) (*JSONMap, error)
}

type HandlersMap map[string]Handler

type RPCServer struct {
	handlers HandlersMap
}

func (this *RPCServer) Add(handler Handler) {
	this.handlers[handler.MethodName()] = handler
}

func (this *RPCServer) Handle(request *RPCRequest, client *Client) {
	defer func() {
		if r := recover(); r != nil {
			log.Warnln("Recovered in Handle:", r)

			client.WriteRPCResponseError(
				request,
				JSONMap{
					"message": "Unknown exception",
				},
			)
		}
	}()

	if handler, ok := this.handlers[request.Method]; ok {
		result, err := handler.Handle(request, client)
		if err != nil {
			client.WriteRPCResponseError(
				request,
				JSONMap{
					"message": err.Error(),
				},
			)
		} else {
			client.WriteRPCResponseError(
				request,
				*result,
			)
		}

	} else {
		client.WriteRPCResponseError(
			request,
			JSONMap{
				"message": "Unsupported method",
			},
		)
	}
}

func NewRPCServer() *RPCServer {
	return &RPCServer{
		handlers: HandlersMap{},
	}
}
