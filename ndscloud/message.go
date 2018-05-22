package ndscloud

import (
	"encoding/json"
)

// 基本消息格式
type Message struct {
	Act     string `json:"act"`
	MsgType json.RawMessage
}

// 互联网用户注册消息
type UsrRegMsg struct {
	Act string `json:"act"`
	Os  string `json:"os"`
	Vi  string `json:"vi"`
	Hw  string `json:"hw"`
}

// 设备注册消息
type DevRegMsg struct {
	Act string `json:"act"`
	Dt  string `json:"dt"`
	Vi  string `json:"vi"`
	Hw  string `json:"hw"`
}

// 本地用户注册消息
type LocalUsrRegMsg struct {
	Act string            `json:"act"`
	Usr []LocalUsrRegItem `json:"usr"`
}

type LocalUsrRegItem struct {
	Uid string `json:"uid"`
	Nm  string `json:"nm"`
	Sex string `json:"sex"`
	Idt string `json:"idt"`
	Os  string `json:"os"`
	Vi  string `json:"vi"`
	Hw  string `json:"hw"`
}

// 本地设备注册消息
type LocalDevRegMsg struct {
	Act string            `json:"act"`
	Dev []LocalDevRegItem `json:"dev"`
}

type LocalDevRegItem struct {
	Did string `json:"did"`
	Nm  string `json:"nm"`
	Dt  string `json:"dt"`
	Vi  string `json:"vi"`
	Hw  string `json:"hw"`
}

// 本地用户/设备全量i消息
type LocalRegMsg struct {
	Act string `json:"act"`
	LocalUsrRegMsg
	LocalDevRegMsg
}

// 解组中控Json消息
func UnmarshalMessage(raw []byte) (*Message, error) {
	message := new(Message)
	err := json.Unmarshal(raw, message)
	if err != nil {
		return nil, err
	}
	var dst interface{}
	switch message.Act {
	case "1":
		dst = new(UsrRegMsg)
	case "2":
		dst = new(DevRegMsg)
	}
	return nil, nil
}
