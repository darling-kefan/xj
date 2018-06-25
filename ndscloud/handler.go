package ndscloud

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/darling-kefan/xj/config"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
)

func ServeUsers(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "posted",
		"message": "hello world",
		"nick":    "tangshouqiang",
	})
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

//func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
func ServeWs(hub *Hub, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		log.Println(err)
		return
	}

	token := c.Query("token")
	if token == "" {
		conn.WriteMessage(websocket.TextMessage, []byte("token is empty."))
		conn.Close()
		return
	}

	unitId := c.Param("unit_id")
	if unitId == "" {
		conn.WriteMessage(websocket.TextMessage, []byte("unit_id is empty."))
		conn.Close()
		return
	}

	redconf := config.Config.Redis
	address := redconf.Host + ":" + strconv.Itoa(redconf.Port)
	redconn, err := redis.Dial("tcp", address)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		conn.Close()
		return
	}
	if redconf.Auth != "" {
		if _, err := redconn.Do("AUTH", redconf.Auth); err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
			conn.Close()
			redconn.Close()
			return
		}
	}
	if _, err := redconn.Do("SELECT", redconf.DB); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		conn.Close()
		redconn.Close()
		return
	}

	// create new client, then add it to the hub.
	client, err := NewClient(token, unitId, redconn, conn, hub)
	if err != nil {
		log.Println(err)
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		conn.Close()
		redconn.Close()
		return
	}

	// Forced login
	client.forceLogin()

	// Registration countdown.
	// Close the client connection if registration is not submitted within two seconds.
	go func() {
		defer func() {
			log.Println("End register countdown")
		}()

		tc := time.After(10 * time.Second)
		select {
		case <-tc:
			if client == hub.get(client.id) && !client.isRegistered {
				hub.unregister <- client
			}
		case <-client.outbound:
			// 如果客户端已经下线，则退出倒计时goroutine
			return
		}
	}()

	go client.readPump()
	go client.writePump()
}
