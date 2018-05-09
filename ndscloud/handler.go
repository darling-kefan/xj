package ndscloud

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
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

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(hub)
	client := &Client{
		hub:      hub,
		conn:     conn,
		outbound: make(chan []byte, 256),
	}

	go client.readPump()
	go client.writePump()
}
