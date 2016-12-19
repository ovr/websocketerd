package main

import (
	"gopkg.in/redis.v5"
	"time"
	"log"
)

type RedisHub struct {
	connection *redis.Client
	pubSub     *redis.PubSub

	subscribes map[string]*Client;
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
		subscribes: map[string]*Client{},
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

			if client, ok := this.subscribes[message.Channel]; ok {
				client.sendChannel <- []byte(message.Payload)
			}
		}

		// Sleep until next iteration
		time.Sleep(redisSleep)
	}
}

func (this *RedisHub) Unsubscribe(channel string, client *Client) {
	err := this.pubSub.Unsubscribe(channel)
	if err != nil {
		log.Printf("Redis Unsubscribe to %s err: %s", channel, err)
	} else {
		if _, ok := this.subscribes[channel]; ok {
			delete(this.subscribes, channel)
		}
	}
}

func (this *RedisHub) Subscribe(channel string, client *Client) {
	err := this.pubSub.Subscribe(channel)
	if err != nil {
		log.Printf("Redis subscribe to %s err: %s", channel, err)
	} else {
		this.subscribes[channel] = client;
	}
}