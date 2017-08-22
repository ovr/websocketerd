package rpc

type RPCResponse struct {
	Id     string                 `json:"id"`
	Result map[string]interface{} `json:"result"`
}

type RPCFatalError struct {
	Error map[string]interface{} `json:"error"`
}

type RPCResponseError struct {
	Id    string                 `json:"id"`
	Error map[string]interface{} `json:"error"`
}
