package main

import (
	"github.com/go-redis/redis"
	"log"
	"sync"
	"time"
)

type ClientsMap map[*Client]bool
type ChannelsMapToClientsMap map[string]ClientsMap

type ChannelsMap map[string]bool
type ClientsToChannelsMap map[*Client]ChannelsMap

type RedisHub struct {
	connection *redis.Client
	pubSub     *redis.PubSub

	channelsToClients     ChannelsMapToClientsMap
	channelsToClientsLock sync.Mutex

	clientsToChannels     ClientsToChannelsMap
	clientsToChannelsLock sync.Mutex
}

const (
	redisSleep time.Duration = 125 * time.Millisecond
)

func NewRedisHub(client *redis.Client) *RedisHub {
	pubSub := client.Subscribe("controller")

	hub := &RedisHub{
		connection:        client,
		pubSub:            pubSub,
		channelsToClients: ChannelsMapToClientsMap{},
		clientsToChannels: ClientsToChannelsMap{},
	}

	go hub.Listen()

	return hub
}

func (this *RedisHub) Listen() {
	for {
		channel := this.pubSub.Channel()

		for message := range channel {
			log.Print(message)

			this.channelsToClientsLock.Lock()

			if clientsMap, ok := this.channelsToClients[message.Channel]; ok {
				for client := range clientsMap {
					client.sendChannel <- []byte(message.Payload)
				}
			}

			this.channelsToClientsLock.Unlock()
		}
	}
}

func (this *RedisHub) Unsubscribe(client *Client) {
	this.channelsToClientsLock.Lock()
	defer this.channelsToClientsLock.Unlock()

	if channels, ok := this.clientsToChannels[client]; ok {
		for channel := range channels {
			if _, ok := this.channelsToClients[channel]; ok {
				delete(this.channelsToClients[channel], client)

				if len(this.channelsToClients[channel]) == 0 {
					err := this.pubSub.Unsubscribe(channel)
					if err != nil {
						log.Printf("Redis Unsubscribe to %s err: %s", channel, err)
					}

					delete(this.channelsToClients, channel)
				}
			}
		}

		delete(this.clientsToChannels, client)
	} else {
		log.Print("Cannot find a client from clientsToChannels map")
	}
}

func (this *RedisHub) Subscribe(channel string, client *Client) {
	err := this.pubSub.Subscribe(channel)
	if err != nil {
		log.Printf("Redis subscribe to %s err: %s", channel, err)
	} else {
		this.channelsToClientsLock.Lock()

		if channelClients, ok := this.channelsToClients[channel]; ok {
			channelClients[client] = true
		} else {
			clients := ClientsMap{}
			clients[client] = true

			this.channelsToClients[channel] = clients
		}

		this.channelsToClientsLock.Unlock()

		this.clientsToChannelsLock.Lock()

		if clientChannels, ok := this.clientsToChannels[client]; ok {
			clientChannels[channel] = true
		} else {
			channels := ChannelsMap{}
			channels[channel] = true

			this.clientsToChannels[client] = channels
		}

		this.clientsToChannelsLock.Unlock()
	}
}
