package main

import (
	"gopkg.in/redis.v5"
	"time"
	"log"
	"sync"
)

type ClientsMap map[*Client]bool
type ChannelsMapToClientsMap map[string]ClientsMap

type RedisHub struct {
	connection *redis.Client
	pubSub     *redis.PubSub

	channelsToClients ChannelsMapToClientsMap
	channelsToClientsLock sync.Mutex
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
		channelsToClients: ChannelsMapToClientsMap{},
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

			this.channelsToClientsLock.Lock();

			if clientsMap, ok := this.channelsToClients[message.Channel]; ok {
				for client := range clientsMap {
					client.sendChannel <- []byte(message.Payload)
				}
			}

			this.channelsToClientsLock.Unlock();
		}

		// Sleep until next iteration
		time.Sleep(redisSleep)
	}
}

func (this *RedisHub) Unsubscribe(client *Client) {
	this.channelsToClientsLock.Lock();
	defer this.channelsToClientsLock.Unlock();

	// @todo Yet another map for fast delete client!
	for channel, clients := range this.channelsToClients {
		if _, ok := clients[client]; ok {
			delete(clients, client)
		}

		if len(clients) == 0 {
			err := this.pubSub.Unsubscribe(channel)
			if err != nil {
				log.Printf("Redis Unsubscribe to %s err: %s", channel, err)
			}
		}
	}
}

func (this *RedisHub) Subscribe(channel string, client *Client) {
	err := this.pubSub.Subscribe(channel)
	if err != nil {
		log.Printf("Redis subscribe to %s err: %s", channel, err)
	} else {
		this.channelsToClientsLock.Lock();

		if channelClients, ok := this.channelsToClients[channel]; ok {
			channelClients[client] = true;
		} else {
			clients := ClientsMap{};
			clients[client] = true;

			this.channelsToClients[channel] = clients;
		}

		this.channelsToClientsLock.Unlock();
	}
}