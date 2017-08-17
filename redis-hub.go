package main

import (
	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
)

type SubscriptionRequest struct {
	channel string
	client  *Client
}

type RedisHub struct {
	HubInterface

	connection *redis.Client
	pubSub     *redis.PubSub

	channelsToClients ChannelsMapToClientsMap

	clientsToChannels ClientsToChannelsMap

	newSubscriptions   chan SubscriptionRequest
	newUnsubscriptions chan *Client
}

func NewRedisHub(client *redis.Client) HubInterface {
	pubSub := client.Subscribe("controller")

	return RedisHub{
		connection:         client,
		pubSub:             pubSub,
		channelsToClients:  ChannelsMapToClientsMap{},
		clientsToChannels:  ClientsToChannelsMap{},
		newSubscriptions:   make(chan SubscriptionRequest, 1000),
		newUnsubscriptions: make(chan *Client, 1000),
	}
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
	log.Debugln("listen")

	pubSubChannel := this.pubSub.Channel()

	for {
		log.Print("test")

		select {
		case subscribeRequest := <-this.newSubscriptions:
			log.Print(subscribeRequest.client, subscribeRequest.channel)
			this.handleSubscribe(subscribeRequest.client, subscribeRequest.channel)
		case client := <-this.newUnsubscriptions:
			this.handleUnsubscribe(client)
		case message := <-pubSubChannel:
			this.handleMessage(message)
		}
	}
}

func (this RedisHub) handleMessage(message *redis.Message) {
	log.Print(message)

	if clientsMap, ok := this.channelsToClients[message.Channel]; ok {
		for client := range clientsMap {
			client.Send([]byte(message.Payload))
		}
	}
}

func (this RedisHub) handleUnsubscribe(client *Client) {
	if channels, ok := this.clientsToChannels[client]; ok {
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

func (this RedisHub) Unsubscribe(client *Client) {
	this.newUnsubscriptions <- client
}

func (this RedisHub) handleSubscribe(client *Client, channel string) {
	err := this.pubSub.Subscribe(channel)
	if err != nil {
		log.Errorln("Redis subscribe to %s err: %s", channel, err)
	} else {
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
	}
}

func (this RedisHub) Subscribe(channel string, client *Client) {
	this.newSubscriptions <- SubscriptionRequest{
		channel: channel,
		client:  client,
	}
}
