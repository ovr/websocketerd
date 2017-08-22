package main

type RPCRequest struct {
	Id         string   `json:"id"`
	Method     string   `json:"method"`
	Parameters []string `json:"parameters"`
}
