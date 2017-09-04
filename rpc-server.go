package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
)

type RPCParameter struct {
	Name string
}

type Handler interface {
	MethodName() string

	Parameters() []RPCParameter

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
		if len(handler.Parameters()) != len(request.Parameters) {
			client.WriteRPCResponseError(
				request,
				JSONMap{
					"message": fmt.Sprintf(
						"Required parameters: [%s]",
						strings.Join(ParametersNames(handler.Parameters()), ", "),
					),
				},
			)

			return
		}

		result, err := handler.Handle(request, client)
		if err != nil {
			client.WriteRPCResponseError(
				request,
				JSONMap{
					"message": err.Error(),
				},
			)
		} else {
			client.WriteRPCResponse(
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

func ParametersNames(parameters []RPCParameter) []string {
	result := make([]string, len(parameters))

	for index, parameter := range parameters {
		result[index] = parameter.Name
	}

	return result
}
