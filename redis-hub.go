package main

import (
	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
	"sync"
)

type RedisHub struct {
	HubInterface

	connection *redis.Client
	pubSub     *redis.PubSub

	channelsToClients     ChannelsMapToClientsMap
	channelsToClientsLock sync.RWMutex

	clientsToChannels     ClientsToChannelsMap
	clientsToChannelsLock sync.RWMutex
}

func NewRedisHub(client *redis.Client) HubInterface {
	pubSub := client.Subscribe("controller")

	hub := RedisHub{
		connection:        client,
		pubSub:            pubSub,
		channelsToClients: ChannelsMapToClientsMap{},
		clientsToChannels: ClientsToChannelsMap{},
	}

	go hub.Listen()

	return hub
}

func (this RedisHub) GetChannels() ChannelsMapToClientsMap {
	return this.channelsToClients
}

func (this RedisHub) GetChannelsForClient(client *Client) ChannelsMap {
	this.clientsToChannelsLock.RLock()
	defer this.clientsToChannelsLock.RUnlock()

	if channels, ok := this.clientsToChannels[client]; ok {
		return channels
	}

	return nil
}

func (this RedisHub) GetClientsCount() int {
	this.clientsToChannelsLock.RLock()

	result := len(this.clientsToChannels)

	this.clientsToChannelsLock.RUnlock()

	return result
}

func (this RedisHub) GetChannelsCount() int {
	this.channelsToClientsLock.RLock()

	result := len(this.channelsToClients)

	this.channelsToClientsLock.RUnlock()

	return result
}

func (this RedisHub) Listen() {
	log.Debugln("listen")

	for {
		channel := this.pubSub.Channel()

		for message := range channel {
			log.Debugln(message)

			this.channelsToClientsLock.RLock()

			if clientsMap, ok := this.channelsToClients[message.Channel]; ok {
				for client := range clientsMap {
					client.sendChannel <- []byte(message.Payload)
				}
			}

			this.channelsToClientsLock.RUnlock()
		}
	}
}

func (this RedisHub) Unsubscribe(client *Client) {
	this.clientsToChannelsLock.Lock()
	defer this.clientsToChannelsLock.Unlock()

	if channels, ok := this.clientsToChannels[client]; ok {
		this.channelsToClientsLock.Lock()
		defer this.channelsToClientsLock.Unlock()

		for channel := range channels {
			if _, ok := this.channelsToClients[channel]; ok {
				delete(this.channelsToClients[channel], client)

				if len(this.channelsToClients[channel]) == 0 {
					err := this.pubSub.Unsubscribe(channel)
					if err != nil {
						log.Errorln("Redis Unsubscribe to %s err: %s", channel, err)
					}

					delete(this.channelsToClients, channel)
				}
			}
		}

		delete(this.clientsToChannels, client)
	} else {
		log.Warnln("Cannot find a client from clientsToChannels map")
	}
}

func (this RedisHub) Subscribe(channel string, client *Client) {
	err := this.pubSub.Subscribe(channel)
	if err != nil {
		log.Errorln("Redis subscribe to %s err: %s", channel, err)
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
