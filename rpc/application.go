package rpc

type Handler interface {
	Handle(request RPCRequest) RPCResponse
}

type HandlersMap map[string]Handler

type Application struct {
	handlers HandlersMap
}

func NewApplication() Application {
	return Application{
		handlers: HandlersMap{},
	}
}
