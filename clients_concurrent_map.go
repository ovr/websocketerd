package main

import "sync"

type ClientsConcurrentMap struct {
	lock sync.RWMutex
	m    ClientsMap
}

func NewClientsConcurrentMap() *ClientsConcurrentMap {
	return &ClientsConcurrentMap{
		m: make(ClientsMap),
	}
}

func (this *ClientsConcurrentMap) Len() int {
	this.lock.RLock()

	result := len(this.m)

	this.lock.RUnlock()

	return result
}

func (this *ClientsConcurrentMap) Get(client *Client) bool {
	this.lock.RLock()

	_, ok := this.m[client]

	this.lock.RUnlock()

	return ok
}

func (this *ClientsConcurrentMap) Add(client *Client) {
	this.lock.Lock()

	this.m[client] = true

	this.lock.Unlock()
}

func (this *ClientsConcurrentMap) Delete(client *Client) {
	this.lock.Lock()

	delete(this.m, client)

	this.lock.Unlock()
}

func (this *ClientsConcurrentMap) Map(f func(client *Client)) {
	this.lock.RLock()

	clients := make([]*Client, 0, len(this.m))
	for k := range this.m {
		clients = append(clients, k)
	}

	this.lock.RUnlock()

	for _, client := range clients {
		f(client)
	}
}
