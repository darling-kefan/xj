package ndscloud

import (
	"encoding/json"
	"errors"
	"strings"
)

// 基本消息格式
type MsgType struct {
	Act string `json:"act"`
}

// 注册消息Act=1
type RegMsg struct {
	Act string `json:"act"`
	Dt  string `json:"dt,omitempty"`
	Os  string `json:"os,omitempty"`
	Vi  string `json:"vi"`
	Hw  string `json:"hw"`
}

// 本地用户注册消息Act=2|3
type LocalRegMsg struct {
	Act string             `json:"act"`
	Usr []*LocalUsrRegItem `json:"usr,omitempty"`
	Dev []*LocalDevRegItem `json:"dev,omitempty"`
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

type LocalDevRegItem struct {
	Did string `json:"did"`
	Nm  string `json:"nm"`
	Dt  string `json:"dt"`
	Vi  string `json:"vi"`
	Hw  string `json:"hw"`
}

// 普通消息Act=6
type OrdinaryMsg struct {
	Act    string      `json:"act"`
	From   string      `json:"from"`
	To     string      `json:"to"`
	Msg    interface{} `json:"msg"`
	Sender string      `json:"-"`
	Unit   string      `json:"-"`
}

// 状态消息Act=7
type ModStatusMsg struct {
	Act       string      `json:"act"`
	Mod       string      `json:"mod"`
	From      string      `json:"from"`
	To        string      `json:"to"`
	Msg       interface{} `json:"msg"`
	CreatedAt int64       `json:"created_at"`
	UpdatedAt int64       `json:"updated_at"`
	Sender    string      `json:"-"`
	Unit      string      `json:"-"`
}

// 用户上线消息Act=8
type UsrOnlineMsg struct {
	Act    string `json:"act"`
	Uid    string `json:"uid"`
	Nm     string `json:"nm"`
	Sex    string `json:"sex"`
	Idt    string `json:"idt"`
	Os     string `json:"os"`
	Vi     string `json:"vi"`
	Hw     string `json:"hw"`
	Sender string `json:"-"`
	Unit   string `json:"-"`
}

// 用户下线消息Act=9
type UsrOfflineMsg struct {
	Act    string `json:"act"`
	Uid    string `json:"uid"`
	Sender string `json:"-"`
	Unit   string `json:"-"`
}

// 设备上线消息Act=10
type DevOnlineMsg struct {
	Act    string `json:"act"`
	Did    string `json:"did"`
	Nm     string `json:"nm"`
	Dt     string `json:"dt"`
	Vi     string `json:"vi"`
	Hw     string `json:"hw"`
	Sender string `json:"-"`
	Unit   string `json:"-"`
}

// 设备下线消息Act=11
type DevOfflineMsg struct {
	Act    string `json:"act"`
	Did    string `json:"did"`
	Sender string `json:"-"`
	Unit   string `json:"-"`
}

// 单元控制(开始/结束)消息Act=12
type UnitControlMsg struct {
	Act    string      `json:"act"`
	From   string      `json:"from"`
	Msg    interface{} `json:"msg"`
	Sender string      `json:"-"`
	Unit   string      `json:"-"`
}

// 开始接收笔迹Act=13
type PullInkMsg struct {
	Act    string `json:"act"`
	From   string `json:"from"`
	Get    string `json:"get"`
	Sender string `json:"-"`
	Unit   string `json:"-"`
}

// 结束接收笔迹Act=14
type EndPullInkMsg struct {
	Act    string `json:"act"`
	From   string `json:"from"`
	Get    string `json:"get"`
	Sender string `json:"-"`
	Unit   string `json:"-"`
}

// 文字聊天消息Act=15
type ChatTextMsg struct {
	Act       string      `json:"act"`
	From      string      `json:"from"`
	Msg       interface{} `json:"msg"`
	CreatedAt int64       `json:"created_at"`
	Sender    string      `json:"-"`
	Unit      string      `json:"-"`
}

// 解组中控Json消息
func UnmarshalMessage(raw []byte) (interface{}, error) {
	message := new(MsgType)
	err := json.Unmarshal(raw, message)
	if err != nil {
		return nil, err
	}
	var dst interface{}
	switch message.Act {
	case "1":
		dst = new(RegMsg)
	case "2":
		dst = new(LocalRegMsg)
	case "3":
		dst = new(LocalRegMsg)
	case "6":
		dst = new(OrdinaryMsg)
	case "7":
		dst = new(ModStatusMsg)
	case "8":
		dst = new(UsrOnlineMsg)
	case "9":
		dst = new(UsrOfflineMsg)
	case "10":
		dst = new(DevOnlineMsg)
	case "11":
		dst = new(DevOfflineMsg)
	case "12":
		dst = new(UnitControlMsg)
	case "13":
		dst = new(PullInkMsg)
	case "14":
		dst = new(EndPullInkMsg)
	case "15":
		dst = new(ChatTextMsg)
	default:
		return nil, errors.New("Cannot identify message format.")
	}
	err = json.Unmarshal(raw, dst)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

// Parse the field to of message and return groups and individuals
func ParseFieldTo(to string) (groups, individuals []string) {
	if to == "" {
		return
	}
	parts := strings.Split(to, "@")
	switch {
	case parts[0] == "A":
		groups = []string{"A"}
		return
	case parts[0] == "T":
		groups = []string{"T"}
	case parts[0] == "S":
		groups = []string{"S"}
	case parts[0] == "D":
		groups = []string{"D"}
	case strings.Index(parts[0], "|") != -1:
		groups = strings.Split(parts[0], "|")
		hasA := false
		for _, v := range groups {
			if v == "A" {
				hasA = true
				break
			}
		}
		if hasA {
			groups = []string{"A"}
			return
		}
	case strings.Index(parts[0], ",") != -1:
		individuals = strings.Split(parts[0], ",")
		return
	default:
		individuals = []string{parts[0]}
		return
	}
	if len(parts) > 1 {
		if parts[1] != "" {
			ia := strings.Split(parts[1], ",")
			individuals = append(individuals, ia...)
		}
	}
	return
}
