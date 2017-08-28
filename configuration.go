package main

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
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
}

type NewRelicConfig struct {
	AppName string `json:"appname"`
	Key     string `json:"key"`
}

type Configuration struct {
	NewRelic  NewRelicConfig        `json:"newrelic"`
	JWTSecret string                `json:"jwt_secret"`
	Redis     RedisConfiguration    `json:"redis"`
	DB        DataBaseConfiguration `json:"db"`
	Debug     bool                  `json:"debug"`
}

func (this *Configuration) Init(configFile string) {
	configJson, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.Unmarshal(configJson, &this)
	if err != nil {
		log.Fatalln(err)
	}
}
