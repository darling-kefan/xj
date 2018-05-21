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
var configFilePath = flag.String("config_file_path", "", "The config file path.(Required)")

type BasicMsg struct {
	Typ byte    // 1字节整数
	Uid [5]byte // 5字节整数
	Act byte    // 1字节整数
}

// 0-重写; 4-清空; 5-撤销
func rcuMsg(act byte) (bts []byte, err error) {
	msg := &BasicMsg{
		Typ: 1,
		Act: act,
	}

	var uid int64 = 27
	buf := new(bytes.Buffer)
	if err = binary.Write(buf, binary.BigEndian, uid); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[3:] {
		msg.Uid[k] = v
	}
	buf.Reset()

	if err = binary.Write(buf, binary.BigEndian, msg); err != nil {
		return nil, err
	}

	bts = buf.Bytes()
	fmt.Printf("% x\n", bts)

	return
}

// 注册指令
type RegisterMsg struct {
	BasicMsg
	Width  [2]byte // 2字节整数
	Height [2]byte // 2字节整数
}

func registerMsg() (bts []byte, err error) {
	var msg RegisterMsg
	msg.Typ = 1
	msg.Act = 1

	var uid int64 = 94
	buf := new(bytes.Buffer)
	if err = binary.Write(buf, binary.BigEndian, uid); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[3:] {
		msg.Uid[k] = v
	}
	buf.Reset()

	var width int32 = 100
	if err = binary.Write(buf, binary.BigEndian, width); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[2:] {
		msg.Width[k] = v
	}
	buf.Reset()

	var height int32 = 50
	if err = binary.Write(buf, binary.BigEndian, height); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[2:] {
		msg.Height[k] = v
	}
	buf.Reset()

	if err = binary.Write(buf, binary.BigEndian, msg); err != nil {
		return nil, err
	}

	bts = buf.Bytes()
	fmt.Printf("% x\n", bts)

	return
}

type UpdateMsg struct {
	BasicMsg
	R    byte // 1字节整数
	G    byte // 1字节整数
	B    byte // 1字节整数
	Size byte // 1字节整数
}

func updateMsg() (bts []byte, err error) {
	var msg UpdateMsg
	msg.Typ = 1
	msg.Act = 2
	msg.R = 0
	msg.G = 0
	msg.B = 0
	msg.Size = 2

	var uid int64 = 94
	buf := new(bytes.Buffer)
	if err = binary.Write(buf, binary.BigEndian, uid); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[3:] {
		msg.Uid[k] = v
	}
	buf.Reset()

	if err = binary.Write(buf, binary.BigEndian, msg); err != nil {
		return nil, err
	}
	bts = buf.Bytes()
	fmt.Printf("% x\n", bts)

	return
}

type CoordinateMsg struct {
	BasicMsg
	X        [2]byte
	Y        [2]byte
	Pressure byte
	State    byte
}

func coordinateMsg() (bts []byte, err error) {
	var msg CoordinateMsg
	msg.Typ = 1
	msg.Act = 3
	msg.Pressure = 9
	msg.State = 0

	buf := new(bytes.Buffer)

	var uid int64 = 94
	if err = binary.Write(buf, binary.BigEndian, uid); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[3:] {
		msg.Uid[k] = v
	}
	buf.Reset()

	var x int32 = 10
	if err = binary.Write(buf, binary.BigEndian, x); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[2:] {
		msg.X[k] = v
	}
	buf.Reset()

	var y int32 = 10
	if err = binary.Write(buf, binary.BigEndian, y); err != nil {
		return nil, err
	}
	for k, v := range buf.Bytes()[2:] {
		msg.Y[k] = v
	}
	buf.Reset()

	if err = binary.Write(buf, binary.BigEndian, msg); err != nil {
		return nil, err
	}
	bts = buf.Bytes()
	fmt.Printf("% x\n", bts)

	return
}

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// 加载配置文件
	if *configFilePath == "" {
		log.Fatal("config_file_path is required.")
	}
	config.Load(*configFilePath)

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

	// OAuth2 使用用户凭证授权方式获取token
	params := make(map[string]interface{}, 2)
	params["username"] = *username
	params["password"] = *password
	token, err := helper.AccessToken(redconn, "password", params)
	if err != nil {
		log.Fatal(err)
	}

	// OAuth2 使用用客户端授权方式获取token
	//token, err := helper.AccessToken(redconn, "client_credentials", nil)
	//if err != nil {
	//	log.Println(err)
	//	return
	//}

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
				log.Println("read:", err, message)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	// 发送笔迹注册指令
	regbytes, err := registerMsg()
	if err != nil {
		log.Println(err)
		return
	}
	err = c.WriteMessage(websocket.BinaryMessage, regbytes)
	if err != nil {
		log.Println("write: ", err)
		return
	}

	// 发送笔迹更新指令
	updatebytes, err := updateMsg()
	if err != nil {
		log.Println(err)
		return
	}
	err = c.WriteMessage(websocket.BinaryMessage, updatebytes)
	if err != nil {
		log.Println("write: ", err)
		return
	}

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

			bts, err := coordinateMsg()
			if err != nil {
				log.Println(err)
				return
			}
			err = c.WriteMessage(websocket.BinaryMessage, bts)
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
