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

	client, err := NewClient(token, unitId, conn, hub)
	if err != nil {
		log.Println(err)
		return
	}

	go client.readPump()
	go client.writePump()
}
