
package main

type ClientsMap map[*Client]bool
type ChannelsMapToClientsMap map[string]ClientsMap

type ChannelsMap map[string]bool
type ClientsToChannelsMap map[*Client]ChannelsMap

type HubInterface interface {
	Listen()

	Unsubscribe(client *Client)

	Subscribe(channel string, client *Client)

	GetClientsCount() int

	GetChannelsCount() int

	GetChannelsForClient(client *Client) ChannelsMap

	GetChannels() ChannelsMapToClientsMap
}
