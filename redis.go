package main

import (
	"gopkg.in/redis.v5"
	"time"
	"log"
)

type RedisHub struct {
	connection *redis.Client
	pubSub     *redis.PubSub
}

const (
	redisSleep          time.Duration = 1 * time.Second
)

func NewRedisHub(client *redis.Client) *RedisHub {
	pubSub, err := client.Subscribe("controller")
	if err != nil {
		log.Printf("Redis subscribe to controller err: %s", err)
	}

	hub := &RedisHub{
		connection: client,
		pubSub: pubSub,
	}

	go hub.Listen();

	return hub
}

func (this *RedisHub) Listen() {
	for {
		message, err := this.pubSub.ReceiveMessage();
		if err != nil {
			log.Printf("Redis ReceiveMessage err: %s", err)
		} else {
			log.Print(message);
		}

		// Sleep until next iteration
		time.Sleep(redisSleep)
	}
}

func (this *RedisHub) Subscribe(channel string, client *Client) {
	err := this.pubSub.Subscribe(channel)
	if err != nil {
		log.Printf("Redis subscribe to %s err: %s", channel, err)
	}
}