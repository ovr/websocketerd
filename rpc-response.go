package main

type RPCResponse struct {
	Id     string  `json:"id"`
	Result JSONMap `json:"result"`
}

type RPCFatalError struct {
	Error JSONMap `json:"error"`
}

type RPCResponseError struct {
	Id    string  `json:"id"`
	Error JSONMap `json:"error"`
}

type SocketMessageWithPayload struct {
	Type string  `json:"type"`
	Data JSONMap `json:"data"`
}
