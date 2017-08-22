package rpc

type Handler interface {
	MethodName() string

	Handle(request *RPCRequest) (*RPCResponse, error)
}

type HandlersMap map[string]Handler

type Server struct {
	handlers HandlersMap
}

func (this *Server) Add(handler Handler) {
	this.handlers[handler.MethodName()] = handler
}

func (this *Server) Handle(request *RPCRequest) {
	if handler, ok := this.handlers[request.Method]; ok {
		handler.Handle(request)
	} else {

	}
}

func NewServer() *Server {
	return &Server{
		handlers: HandlersMap{},
	}
}
