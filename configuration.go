package main

import (
	"io/ioutil"
	"fmt"
	"os"
	"encoding/json"
)

type RedisConfiguration struct {
	Addr string `json:"addr"`
	MaxRetries int `json:"max_retries"`
	PoolSize int `json:"pool_size"`
}

type DataBaseConfiguration struct {
	Dialect            string `json:"dialect"`
	Uri                string `json:"uri"`
	MaxIdleConnections int    `json:"max-idle-connections"`
	MaxOpenConnections int    `json:"max-open-connections"`
	ShowLog            bool   `json:"log"`
	Threads            uint8  `json:"threads"`
	Limit              uint16 `json:"limit"`
}

type Configuration struct {
	JWTSecret string `json:"jwt_secret"`
	Redis RedisConfiguration `json:"redis"`
	DB DataBaseConfiguration `json:"db"`
}

func (this *Configuration) Init(configFile string) {
	configJson, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = json.Unmarshal(configJson, &this)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}