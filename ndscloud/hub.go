package ndscloud

import (
	"encoding/json"
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
	unitmap map[string]*UnitCache

	// The lock used for Hub.clients and Hub.unitmap
	mutex sync.RWMutex

	// Inbound messages from the clients. Handle ordinary text messages.
	inbound chan interface{}

	// Penmanship binary stream.
	inbound_pms chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// End unit
	endunit chan string
}

// Classification by identity, and cache it.
type UnitCache struct {
	All map[string]struct{} // 定义set类型
	Tea map[string]struct{}
	Stu map[string]struct{}
	Dev map[string]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients:     make(map[string]*Client),
		unitmap:     make(map[string]*UnitCache),
		inbound:     make(chan interface{}),
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

	var found bool
	var uc *UnitCache
	for _, client := range clients {
		h.clients[client.id] = client

		// 获取客户端缓存，存在返回；不存在，则初始化。
		if uc, found = h.unitmap[client.unitId]; !found {
			uc = &UnitCache{
				All: make(map[string]struct{}),
				Tea: make(map[string]struct{}),
				Stu: make(map[string]struct{}),
				Dev: make(map[string]struct{}),
			}
			h.unitmap[client.unitId] = uc
		}
		// 全体
		uc.All[client.id] = struct{}{}
		if client.identity == 1 {
			// 老师
			uc.Tea[client.id] = struct{}{}
		} else if client.identity == 2 {
			// 学生
			uc.Stu[client.id] = struct{}{}
		}
		if _, ok := client.info.(*DeviceInfo); ok {
			// 设备
			uc.Dev[client.id] = struct{}{}
		}
	}
}

// 移除客户端
func (h *Hub) remove(clients ...*Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for _, client := range clients {
		if _, ok := h.clients[client.id]; ok {
			// close client websocket connection
			close(client.outbound)
			delete(h.clients, client.id)
		}

		// Clear unit cache
		if uc, ok := h.unitmap[client.unitId]; ok {
			delete(uc.All, client.id)
			if client.identity == 1 {
				// 老师
				delete(uc.Tea, client.id)
			} else if client.identity == 2 {
				// 学生
				delete(uc.Stu, client.id)
			}
			if _, ok := client.info.(*DeviceInfo); ok {
				// 设备
				delete(uc.Dev, client.id)
			}
		}
	}
}

// 根据单元id移除客户端
func (h *Hub) removebyunitid(unitid string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if uc, ok := h.unitmap[unitid]; ok {
		for id, _ := range uc.All {
			// close client websocket connection
			close(h.clients[id].outbound)
			delete(h.clients, id)
		}
		delete(h.unitmap, unitid)
	}
}

// 判断客户端是否存在
func (h *Hub) exists(client *Client) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	_, ok := h.clients[client.id]
	return ok
}

// 根据id获取客户端
func (h *Hub) get(id string) (client *Client) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if client, ok := h.clients[id]; ok {
		return client
	}
	return nil
}

// 获取所有客户端
// TODO 为防止数据竞争，此处返回[]Client而不是[]*Client
func (h *Hub) list() []Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	list := make([]Client, 0)
	for _, item := range h.clients {
		list = append(list, *item)
	}
	return list
}

// Calculate message receivers.
// Determine which clients to send to
func (h *Hub) msgrecvers(message interface{}) (receivers []string) {
	// Is it send back to the sender
	var toSender bool = true
	// The sender of the message
	var sender string
	// The set of receivers
	var receiverSet map[string]struct{}

	switch msg := message.(type) {
	case *OrdinaryMsg:
		sender = msg.Sender
		toSender = false

	case *ModStatusMsg:
		sender = msg.Sender
		toSender = false
	case *UsrOnlineMsg:
		sender = msg.Sender
		toSender = false
		receiverSet = h.unitmap[msg.Unit].All
	case *UsrOfflineMsg:
		sender = msg.Sender
		toSender = false
		receiverSet = h.unitmap[msg.Unit].All
	case *DevOnlineMsg:
		sender = msg.Sender
		toSender = false
		receiverSet = h.unitmap[msg.Unit].All
	case *DevOfflineMsg:
		sender = msg.Sender
		toSender = false
		receiverSet = h.unitmap[msg.Unit].All
	case *UnitControlMsg:
		sender = msg.Sender
		toSender = false
	case *ChatTextMsg:
		sender = msg.Sender
		toSender = false
	}

	// Remove sender
	// TODO ids存的是map的地址，此处执行delete会真正删除hub.unitmap里的内容。
	//if !toSender {
	//	delete(ids, sender)
	//}

	// 分配内存
	if _, ok := receiverSet[sender]; ok {
		receivers = make([]string, len(receiverSet)-1)
	} else {
		receivers = make([]string, len(receiverSet))
	}

	var i int = 0
	for id, _ := range receiverSet {
		// Remove sender
		if !toSender && sender == id {
			continue
		}
		receivers[i] = id
		i = i + 1
	}

	return
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.add(client)
		case client := <-h.unregister:
			h.remove(client)
		case unitid := <-h.endunit:
			h.removebyunitid(unitid)
		case message := <-h.inbound:
			log.Printf("%#v\n", message)
			if msg, err := json.Marshal(message); err == nil {
				receivers := h.msgrecvers(message)
				log.Printf("Message receivers: %v", receivers)
				for _, id := range receivers {
					h.clients[id].outbound <- msg
				}
			} else {
				log.Println(err)
			}
		}
	}
}
