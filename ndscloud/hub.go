package ndscloud

import (
	"log"
	"sync"
)

// 基于目前的设计：只有一个goroutine可访问Hub，因此Hub.clients和Hub.unittoids是并发安全的～
// 避免竞争条件的三种方式：
// > 不要去写变量
// > 避免多个goroutine访问变量
// > 允许很多goroutine去访问变量，但是在同一个时刻最多只有一个goroutine在访问，使用使用互斥锁等方式。
//
// 基于需求，Hub.clients, Hub.unitotids也提供于其它接口使用，因此对其操作应该加锁。

// Hub maintains the set of active clients and broadcast messages to the
// clients.
type Hub struct {
	// Registered clients. ID to Client mapping.
	clients map[string]*Client

	// UnitId to IDs mapping
	unittoids map[string][]string

	// The lock used for Hub.clients and Hub.unittoids
	mutex sync.RWMutex

	// Inbound messages from the clients. Handle ordinary text messages.
	inbound chan []byte

	// Penmanship binary stream.
	inbound_pms chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// End unit
	endunit chan string
}

func NewHub() *Hub {
	return &Hub{
		clients:     make(map[string]*Client),
		unittoids:   make(map[string][]string),
		inbound:     make(chan []byte),
		inbound_pms: make(chan []byte),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		endunit:     make(chan string),
	}
}

// 新增客户端
func (h *Hub) add(clients ...*Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for _, client := range clients {
		h.clients[client.ID] = client

		if ids, ok := h.unittoids[client.UnitId]; ok {
			// 防止ID重复，首先判断ID是否已经存在, 如果不存在则写入。
			isIn := false
			for _, id := range ids {
				if id == client.ID {
					isIn = true
				}
			}
			if !isIn {
				ids = append(ids, client.ID)
				h.unittoids[client.UnitId] = ids
			}
		} else {
			h.unittoids[client.UnitId] = []string{client.ID}
		}

		log.Println(h.clients, h.unittoids)
	}
}

// 移除客户端
func (h *Hub) remove(clients ...*Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for _, client := range clients {
		if _, ok := h.clients[client.ID]; ok {
			// close client websocket connection
			close(client.outbound)
			delete(h.clients, client.ID)
		}

		if ids, ok := h.unittoids[client.UnitId]; ok {
			for i := range ids {
				if ids[i] == client.ID {
					ids = append(ids[:i], ids[i+1:]...)
				}
			}
			h.unittoids[client.UnitId] = ids
		}
	}
}

// 根据单元id移除客户端
func (h *Hub) removebyunitid(unitid string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if ids, ok := h.unittoids[unitid]; ok {
		for _, id := range ids {
			// close client websocket connection
			close(h.clients[id].outbound)
			delete(h.clients, id)
		}
		delete(h.unittoids, unitid)
	}
}

// 判断客户端是否存在
func (h *Hub) exists(client *Client) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	_, ok := h.clients[client.ID]
	return ok
}

// 根据id获取客户端
func (h *Hub) get(id string) (client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if client, ok := h.clients[id]; ok {
		return client
	}
	return nil
}

// Parse packet
// Determine which clients to send to

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.add(client)
		case client := <-h.unregister:
			h.remove(client)
		case unitid := <-h.endunit:
			h.removebyunitid(unitid)
		}
	}
}
