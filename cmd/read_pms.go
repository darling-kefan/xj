// 脚本执行
// go run simulate_pms.go --unitid A16 --username tangshouqiang --password 123456
// go run read_pms.go --unitid A16

package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/darling-kefan/xj/config"
	"github.com/darling-kefan/xj/helper"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
)

var unitid = flag.String("unitid", "", "unit id")

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// 建立redis连接
	redconf := config.Config.Redis
	address := redconf.Host + ":" + strconv.Itoa(redconf.Port)
	redconn, err := redis.Dial("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	if redconf.Auth != "" {
		if _, err := redconn.Do("AUTH", redconf.Auth); err != nil {
			redconn.Close()
			log.Fatal(err)
		}
	}
	if _, err := redconn.Do("SELECT", redconf.DB); err != nil {
		redconn.Close()
		log.Fatal(err)
	}

	token, err := helper.AccessToken(redconn, "client_credentials", nil)
	if err != nil {
		log.Println(err)
		return
	}

	v := url.Values{}
	v.Set("token", token)
	rawquery := v.Encode()
	path := fmt.Sprintf("v2/ngx/center/units/%s/", *unitid)
	u := url.URL{
		Scheme:   config.Config.Common.NdscloudScheme,
		Host:     config.Config.Common.NdscloudDomain,
		Path:     path,
		RawQuery: rawquery,
	}
	log.Println(u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial: ", err)
	}
	defer c.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err, message)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			fmt.Println("Bye!")
			return
		case <-ticker.C:
			// 发送接收笔迹流指令
			err := c.WriteMessage(websocket.TextMessage, []byte(`{"act":"13","get":"A@129"}`))
			if err != nil {
				log.Println("write:", err)
				return
			}
			log.Println("write websocket message...........")
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message
			// and then waiting (with timeout) for the server to close
			// the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
				fmt.Println("Bye bye!")
			}
			return
		}
	}
}
