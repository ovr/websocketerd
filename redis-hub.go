package main

import (
	"github.com/go-redis/redis"
	"log"
	"sync"
)

type RedisHub struct {
	HubInterface

	connection *redis.Client
	pubSub     *redis.PubSub

	channelsToClients     ChannelsMapToClientsMap
	channelsToClientsLock sync.Mutex

	clientsToChannels     ClientsToChannelsMap
	clientsToChannelsLock sync.Mutex

	// Register requests from clients.
	registerChannel chan *Client

	// Unregister requests from clients.
	unregisterChannel chan *ClientChannelPair
}

func NewRedisHub(client *redis.Client) HubInterface {
	pubSub := client.PSubscribe("pubsub:user:*")

	hub := RedisHub{
		connection:        client,
		pubSub:            pubSub,
		registerChannel:   make(chan *Client, 1024),
		unregisterChannel: make(chan *ClientChannelPair, 1024),
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
	if channels, ok := this.clientsToChannels[client]; ok {
		return channels
	}

	return nil
}

func (this RedisHub) GetClientsCount() int {
	return len(this.clientsToChannels)
}

func (this RedisHub) GetChannelsCount() int {
	return len(this.channelsToClients)
}

func (this RedisHub) Listen() {
	for {
		channel := this.pubSub.Channel()

		select {

		case client := <-this.registerChannel:
			if channels, ok := this.clientsToChannels[client]; ok {
				for channel := range channels {
					if _, ok := this.channelsToClients[channel]; ok {
						if len(this.channelsToClients[channel]) == 1 {
							delete(this.channelsToClients, channel)
						} else {
							delete(this.channelsToClients[channel], client)
						}
					}
				}

				delete(this.clientsToChannels, client)
			} else {
				log.Print("Cannot find a client from clientsToChannels map")
			}

		case clientChannelPair := <-this.unregisterChannel:
			channel := clientChannelPair.Channel
			client := clientChannelPair.Client

			if channelClients, ok := this.channelsToClients[channel]; ok {
				channelClients[client] = true
			} else {
				clients := ClientsMap{}
				clients[client] = true

				this.channelsToClients[channel] = clients
			}

			if clientChannels, ok := this.clientsToChannels[client]; ok {
				clientChannels[channel] = true
			} else {
				channels := ChannelsMap{}
				channels[channel] = true

				this.clientsToChannels[client] = channels
			}

		case message := <-channel:
			log.Print(message)

			if clientsMap, ok := this.channelsToClients[message.Channel]; ok {
				for client := range clientsMap {
					client.sendChannel <- []byte(message.Payload)
				}
			}
		}
	}
}

func (this RedisHub) Unsubscribe(client *Client) {
	this.registerChannel <- client
}

func (this RedisHub) Subscribe(channel string, client *Client) {
	this.unregisterChannel <- &ClientChannelPair{
		Client:  client,
		Channel: channel,
	}
}
