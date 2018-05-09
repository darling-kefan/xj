// 脚本执行
// go run simulate_pms.go --unitid A16 --username tangshouqiang --password 123456

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
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
var username = flag.String("username", "", "username")
var password = flag.String("password", "", "password")

type Msg struct {
	Typ byte
	Uid int32
	Act byte
	//Data string
}

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// OAuth2认证
	redconf := config.Config.Redis
	address := redconf.Host + ":" + strconv.Itoa(redconf.Port)
	red, err := redis.Dial("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	if redconf.Auth != "" {
		if _, err := red.Do("AUTH", redconf.Auth); err != nil {
			red.Close()
			log.Fatal(err)
		}
	}
	if _, err := red.Do("SELECT", redconf.DB); err != nil {
		red.Close()
		log.Fatal(err)
	}
	params := make(map[string]interface{}, 2)
	params["username"] = *username
	params["password"] = *password
	token, err := helper.AccessToken(red, "password", params)
	if err != nil {
		log.Fatal(err)
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

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial: ", err)
	}
	defer c.Close()

	// 注册
	reg := map[string]string{
		"act": "1",
		"os":  "1",
		"vi":  "1",
		"hw":  "0",
	}
	regjson, err := json.Marshal(reg)
	if err != nil {
		log.Fatal(err)
	}
	err = c.WriteMessage(websocket.TextMessage, []byte(regjson))
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
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
			//case t := <-ticker.C:
		case <-ticker.C:
			//err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			//if err != nil {
			//	log.Println("write:", err)
			//	return
			//}

			// TODO 为什么该数据类型不能以二进制流发放
			data := &Msg{1, 123, 2}
			//data := &Msg{1, 123, 2}
			buf := new(bytes.Buffer)
			binary.Write(buf, binary.BigEndian, data)
			bytes := buf.Bytes()
			bytes = append(bytes, []byte("BBB")...)
			fmt.Println(bytes)
			err := c.WriteMessage(websocket.BinaryMessage, bytes)
			if err != nil {
				log.Println("write: ", err)
				return
			}
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
