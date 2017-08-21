package main

import (
	"encoding/json"
	"strconv"
)

type JSONMap map[string]interface{}

type AutoLoginToken struct {
	UserId      json.Number
	Token       string
	BrowserHash string
}

type RPCRequest struct {
	Id         string   `json:"id"`
	Method     string   `json:"method"`
	Parameters []string `json:"parameters"`
}

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

type WebSocketNotification struct {
	Type   string      `json:"type"`
	Entity interface{} `json:"entity"`
}

type User struct {
	Id             uint64 `gorm:"primary_key"`
	ThreadsShardNo uint64 `gorm:"column:threads_shard_no"`
}

func (User) TableName() string {
	return "users"
}

type Thread struct {
	threadSharedId uint64
}

func (this Thread) TableName() string {
	return "users" + strconv.FormatUint(this.threadSharedId, 64)
}

type LoginToken struct {
	UserId  uint64
	Token   string
	Updated string
}
