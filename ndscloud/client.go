package ndscloud

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/darling-kefan/xj/helper"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Client is a middleman between the websocket connection and the server.
type Client struct {
	// The Hub
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// The redis connection.
	redconn redis.Conn

	// Buffered channel of outbound messages.
	outbound chan []byte

	// Used to uniquely identity users and devices.
	id string

	// Unit ID. Each client can only belong to one unit at a time.
	unitId string

	// The identity of the client.
	identity int

	// The client details. *UserInfo or *DeviceInfo
	info interface{}

	// The unit details.
	unitInfo *UnitInfo

	// The stage of the client.
	// false: not registered; true: registered
	isRegistered bool

	// 注册时间
	registeredAt int64

	// 设备类型
	dt string

	// 用户系统
	os string

	// 是否支持视频互动
	vi string

	// 是否支持手写
	hw string

	// 读写锁(由于其它接口线程会读取LocalUsers和LocalDevices而产生竞争条件，因此需要加锁)
	mtx sync.RWMutex

	// 本地中控上报的用户
	localUsers *LocalUserSet

	// 本地中控上报的设备
	localDevices *LocalDeviceSet
}

func NewClient(token string, unitId string, redconn redis.Conn, conn *websocket.Conn, hub *Hub) (client *Client, err error) {
	// 获取系统token
	systoken, err := helper.AccessToken(redconn, "client_credentials", nil)

	// 获取单元信息
	unitInfo, err := getUnitInfo(systoken, unitId)
	if err != nil {
		return nil, err
	}

	// 验证Token
	tokenInfo, err := getTokenInfo(token)
	if err != nil {
		return nil, err
	}

	var id string
	var identity int

	if userInfo, ok := tokenInfo.(*UserInfo); ok {
		id = userInfo.Uid
		// 获取用户在课程单元中的身份
		unitidt, err := getUnitidt(systoken, unitId, userInfo.Uid)
		if err != nil {
			return nil, err
		}
		if unitidt.Identity != "" {
			identity, _ = strconv.Atoi(unitidt.Identity)
		}
	}

	if deviceInfo, ok := tokenInfo.(*DeviceInfo); ok {
		// client_id作为设备id
		id = deviceInfo.ClientId
	}

	client = &Client{
		hub:          hub,
		conn:         conn,
		redconn:      redconn,
		outbound:     make(chan []byte, 256),
		id:           id,
		unitId:       unitId,
		identity:     identity,
		info:         tokenInfo,
		unitInfo:     unitInfo,
		os:           "0",
		vi:           "0",
		hw:           "0",
		localUsers:   NewLocalUserSet(),
		localDevices: NewLocalDeviceSet(),
	}
	return
}

// Determine if the client is a device
func (c *Client) isDevice() bool {
	_, ok := c.info.(*DeviceInfo)
	return ok
}

// Determine if the client is a user
func (c *Client) isUser() bool {
	_, ok := c.info.(*UserInfo)
	return ok
}

// Determine if the client is a local control
func (c *Client) isLocalControl() bool {
	if c.dt == "1" {
		return true
	}
	return false
}

// Forced login. Force other terminals to disconnect.
func (c *Client) forceLogin() {
	// Judging whether to login
	if c.hub.exists(c) {
		// 1. 获取登录中的客户端，并向该客户端发送强制退出消息
		loginClient := c.hub.get(c.id)
		loginClient.conn.SetWriteDeadline(time.Now().Add(writeWait))
		loginClient.conn.WriteMessage(websocket.TextMessage, []byte(`{"errcode":1, "errmsg":"forced logout"}`))

		// 2. 退出登录中的客户端
		c.hub.unregister <- loginClient
	}
	// 3. 新客户端登录
	c.hub.register <- c
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// read from this goroutine.
func (c *Client) readPump() {
	defer func() {
		if c == c.hub.get(c.id) {
			c.hub.unregister <- c
		}
		c.conn.Close()
		log.Println("End readPump")
	}()
	// Set the maximum size for a message read from the peer.
	// https://godoc.org/github.com/gorilla/websocket#Conn.SetReadLimit
	c.conn.SetReadLimit(maxMessageSize)
	// Set the read deadline on the underlying network connection.
	// https://godoc.org/github.com/gorilla/websocket#Conn.SetReadDeadline
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	// TODO wireshark抓包分析ping/pong
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		switch messageType {
		case websocket.TextMessage:
			log.Printf("[%s] receive: %s\n", c.id, string(message))
			// Message processor
			c.process(message)
		case websocket.BinaryMessage:

		case websocket.CloseMessage:

		case websocket.PingMessage:

		case websocket.PongMessage:

		default:
			log.Println("Unknown messageType: ", messageType)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		log.Println("End writePump")
	}()
	for {
		select {
		case message, ok := <-c.outbound:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteMessage(websocket.TextMessage, message)

			// @TODO 由于可能存在多个goroutine同时向终端发送消息，因此此处不适合批量发送消息
			//// Send messages to terminals in batches
			//w, err := c.conn.NextWriter(websocket.TextMessage)
			//if err != nil {
			//	return
			//}
			//w.Write(message)
			//
			//// Add queued messages to the current websocket message.
			//n := len(c.outbound)
			//for i := 0; i < n; i++ {
			//	w.Write(newline)
			//	w.Write(<-c.outbound)
			//}
			//
			//if err := w.Close(); err != nil {
			//	return
			//}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
