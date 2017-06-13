package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type RedisConfiguration struct {
	Addr       string `json:"addr"`
	MaxRetries int    `json:"max_retries"`
	PoolSize   int    `json:"pool_size"`
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

type HubConfiguration struct {
	RegisterChannelSize   int `json:"register_channel_size"`
	UnregisterChannelSize int `json:"unregister_channel_size"`
}

type Configuration struct {
	NewRelicLicenseKey string                `json:"newrelic_license_key"`
	JWTSecret          string                `json:"jwt_secret"`
	Redis              RedisConfiguration    `json:"redis"`
	DB                 DataBaseConfiguration `json:"db"`
	Hub                HubConfiguration      `json:"hub"`
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
