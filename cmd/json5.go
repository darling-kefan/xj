package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type TokenInfo struct {
	DeviceInfo
	UserInfo
}

type UserInfo struct {
	Uid  string `json:"uid"`
	Name string `json:"name"`
	Os   string `json:"os"`
	Vi   string `json:"vi"`
}

type DeviceInfo struct {
	Did  string `json:"did"`
	Name string `json:"name"`
	Dt   string `json:"dt"`
	Vi   string `json:"vi"`
}

func main() {
	jsonStream := `{
    "uid": "1000",
    "Os": "1"
}`

	var deviceInfo DeviceInfo
	err := json.Unmarshal([]byte(jsonStream), &deviceInfo)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("%#v\n", deviceInfo)

	var a interface{}
	a = &deviceInfo
	switch v := a.(type) {
	case *DeviceInfo:
		v.Dt = "1"
	}
	log.Printf("%#v\n", a)

	log.Println("------------------------------------------------")

	m := map[string]bool{
		"a": true,
		"b": true,
	}
	log.Println(m)

	delete(m, "a")
	delete(m, "c")
	log.Println(m)

	log.Println("------------------------------------------------")

	jsonStream = `{
    "act": "12",
    "from": "101:小新",
    "msg": {
        "stat": "1",
        "code": "1234"
    }
}`
	type UnitControlMsg struct {
		Act    string      `json:"act"`
		From   string      `json:"from"`
		Msg    interface{} `json:"msg"`
		Sender string      `json:"-"`
		Unit   string      `json:"-"`
	}
	var unitControlMsg UnitControlMsg
	if err := json.Unmarshal([]byte(jsonStream), &unitControlMsg); err != nil {
		log.Println(err)
	}
	stat := unitControlMsg.Msg.(map[string]interface{})["stat"]
	log.Printf("%#v\n", stat)

	log.Println("------------------------------------------------")

	log.Println(fmt.Sprintf(`{"errcode": 1, "errmsg": "%s"}`, "hello world"))

}
